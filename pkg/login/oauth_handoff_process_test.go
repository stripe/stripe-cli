package login

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// A real on-disk credential store for subprocess tests. It never opens the
// user's OS keychain. Handoff operations serialize access with the real lock.
type handoffDiskTestStore struct{ root string }

func (s handoffDiskTestStore) Get(key string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(s.root, key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, keyring.ErrKeyNotFound
	}
	return data, err
}

func (s handoffDiskTestStore) Set(key string, data []byte, _ string) error {
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, key), data, 0600)
}

func (s handoffDiskTestStore) Remove(key string) error {
	err := os.Remove(filepath.Join(s.root, key))
	if errors.Is(err, os.ErrNotExist) {
		return keyring.ErrKeyNotFound
	}
	return err
}

func TestHandoffProcessHelper(t *testing.T) {
	if os.Getenv("STRIPE_HANDOFF_TEST_PROCESS") != "1" {
		return
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	require.True(t, filepath.IsAbs(root))
	config.KeyRing = handoffDiskTestStore{root: filepath.Join(root, "test-keyring")}
	cfg := &config.Config{LogLevel: "info", Profile: config.Profile{ProfileName: "default"}, ProfilesFile: filepath.Join(root, "config.toml")}
	cfg.InitConfig()
	target, err := url.Parse(os.Getenv("STRIPE_HANDOFF_TEST_SERVER"))
	require.NoError(t, err)
	accessSrvHTTPClient = &http.Client{Transport: handoffTestTransport{target: target, transport: http.DefaultTransport}}
	handoffNow = func() time.Time { return time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	operation := os.Getenv("STRIPE_HANDOFF_TEST_OPERATION")
	if operation == "hold-lock" {
		unlock, err := lockOAuthHandoff(ctx)
		require.NoError(t, err)
		defer unlock()
		fmt.Println("locked")
		select {}
	}
	handoff, err := BeginOrResumeLogin(ctx, DefaultAccessBaseURL, cfg)
	require.NoError(t, err)
	if operation == "check" {
		handoff, err = CheckLogin(ctx, DefaultAccessBaseURL, cfg, handoff.ID)
		require.NoError(t, err)
	}
	require.NoError(t, json.NewEncoder(os.Stdout).Encode(handoff))
	if operation == "wait" {
		// A pending process waiting for human input holds no file lock.
		require.NoError(t, waitForLoginHandoff(ctx, DefaultAccessBaseURL, cfg, handoff.ID))
	}
}

func handoffProcess(t *testing.T, root, server, operation string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(executable, "-test.run=^TestHandoffProcessHelper$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "STRIPE_HANDOFF_TEST_PROCESS=1", "STRIPE_HANDOFF_TEST_OPERATION="+operation,
		"STRIPE_HANDOFF_TEST_SERVER="+server, "XDG_CONFIG_HOME="+root, "XDG_CONFIG="+root)
	return cmd
}

func decodeProcessHandoff(t *testing.T, data string) *LoginHandoff {
	t.Helper()
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "{\"state\"") {
			var h LoginHandoff
			require.NoError(t, json.Unmarshal([]byte(line), &h))
			return &h
		}
	}
	t.Fatalf("subprocess did not return a handoff: %s", data)
	return nil
}

func TestHandoffSeparateProcessesResumeAfterKill(t *testing.T) {
	for _, approveBeforeRetry := range []bool{true, false} {
		t.Run(fmt.Sprintf("approve-before-retry=%t", approveBeforeRetry), func(t *testing.T) {
			f := newHandoffFixture(t)
			root := t.TempDir()
			server := accessSrvHTTPClient.Transport.(handoffTestTransport).target.String()
			waiter := handoffProcess(t, root, server, "wait")
			stdout, err := waiter.StdoutPipe()
			require.NoError(t, err)
			require.NoError(t, waiter.Start())
			t.Cleanup(func() { _ = waiter.Process.Kill() })
			scanner := bufio.NewScanner(stdout)
			require.True(t, scanner.Scan())
			original := decodeProcessHandoff(t, scanner.Text())
			require.NoError(t, waiter.Process.Kill())
			require.Error(t, waiter.Wait())
			if approveBeforeRetry {
				f.approved.Store(true)
			}
			out, err := handoffProcess(t, root, server, "begin").CombinedOutput()
			require.NoError(t, err, "%s", out)
			retried := decodeProcessHandoff(t, string(out))
			assert.Equal(t, original.ID, retried.ID)
			assert.Equal(t, original.VerificationCode, retried.VerificationCode)
			assert.Equal(t, original.BrowserURL, retried.BrowserURL)
			assert.Equal(t, original.ExpiresAt, retried.ExpiresAt)
			f.approved.Store(true)
			// Let the persisted throttle elapse with a deterministic clock in
			// the checking process, rather than sleeping in the test.
			path := filepath.Join(root, "stripe", "oauth_pending.json")
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			var cont oauthContinuation
			require.NoError(t, json.Unmarshal(data, &cont))
			cont.NextPollAt = time.Time{}
			data, err = json.Marshal(cont)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, data, 0600))
			out, err = handoffProcess(t, root, server, "check").CombinedOutput()
			require.NoError(t, err, "%s", out)
			assert.Equal(t, LoginHandoffAuthenticated, decodeProcessHandoff(t, string(out)).State)
			assert.EqualValues(t, 1, f.issued.Load(), "no replacement authorization after process death")
		})
	}
}

func TestHandoffProcessDeathReleasesKernelLock(t *testing.T) {
	newHandoffFixture(t)
	root := t.TempDir()
	server := accessSrvHTTPClient.Transport.(handoffTestTransport).target.String()
	holder := handoffProcess(t, root, server, "hold-lock")
	stdout, err := holder.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, holder.Start())
	t.Cleanup(func() { _ = holder.Process.Kill() })
	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan())
	require.Equal(t, "locked", scanner.Text())
	require.NoError(t, holder.Process.Kill())
	require.Error(t, holder.Wait())
	out, err := handoffProcess(t, root, server, "begin").CombinedOutput()
	require.NoError(t, err, "%s", out)
	assert.Equal(t, LoginHandoffPending, decodeProcessHandoff(t, string(out)).State)
}

func TestHandoffConcurrentProcessesCreateOneAuthorization(t *testing.T) {
	f := newHandoffFixture(t)
	root := t.TempDir()
	server := accessSrvHTTPClient.Transport.(handoffTestTransport).target.String()
	var commands []*exec.Cmd
	var outputs []*bytes.Buffer
	for i := 0; i < 4; i++ {
		cmd := handoffProcess(t, root, server, "begin")
		output := new(bytes.Buffer)
		cmd.Stdout, cmd.Stderr = output, output
		require.NoError(t, cmd.Start())
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		commands = append(commands, cmd)
		outputs = append(outputs, output)
	}
	var id string
	for i, cmd := range commands {
		err := cmd.Wait()
		require.NoError(t, err, "%s", outputs[i])
		handoff := decodeProcessHandoff(t, outputs[i].String())
		if id == "" {
			id = handoff.ID
		}
		assert.Equal(t, id, handoff.ID)
	}
	assert.EqualValues(t, 1, f.issued.Load())
}

func TestHandoffSeparateProcessesRecoverCheckpoint(t *testing.T) {
	f := newHandoffFixture(t)
	root := t.TempDir()
	server := accessSrvHTTPClient.Transport.(handoffTestTransport).target.String()
	f.approved.Store(true)
	f.accountsFail.Store(true)
	output, err := handoffProcess(t, root, server, "check").CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "account_lookup_failed_resume_same_handoff")
	f.accountsFail.Store(false)
	output, err = handoffProcess(t, root, server, "check").CombinedOutput()
	require.NoError(t, err, "%s", output)
	result := decodeProcessHandoff(t, string(output))
	assert.Equal(t, LoginHandoffAuthenticated, result.State)
	assert.Equal(t, "acct_fixture", result.AccountID)
	assert.EqualValues(t, 1, f.issued.Load())
	assert.EqualValues(t, 1, f.polled.Load(), "a new process must install the checkpoint without redeeming again")
}
