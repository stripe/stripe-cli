package canary

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stripe/stripe-cli/canary/testutil"
	"github.com/stripe/stripe-cli/rpc"
)

// =============================================================================
// Daemon Tests
// =============================================================================

// daemonConfig is the first line that `stripe daemon` prints to stdout. Clients like the Stripe
// VS Code extension parse it to connect.
type daemonConfig struct {
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Token string `json:"token"`
}

// startDaemon runs `stripe daemon` with an isolated config and returns a client connected to it.
func startDaemon(t *testing.T, env map[string]string) (rpc.StripeCLIClient, daemonConfig) {
	t.Helper()

	configDir, err := testutil.CreateTempConfigDir("daemon")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(configDir) })

	runner := getRunner(t, testutil.WithConfigDir(configDir), testutil.WithEnv(env))
	daemon, err := runner.RunBackground("daemon")
	if err != nil {
		fatalf(t, "Failed to start 'stripe daemon': %v", err)
	}
	t.Cleanup(func() { daemon.Stop() })

	if err := daemon.WaitForOutput(`"token"`, 30*time.Second); err != nil {
		fatalf(t, "'stripe daemon' didn't print its config: %v", err)
	}

	stdout, _ := daemon.GetOutput()
	firstLine, _, _ := strings.Cut(stdout, "\n")
	var cfg daemonConfig
	if err := json.Unmarshal([]byte(firstLine), &cfg); err != nil {
		fatalf(t, "First line of 'stripe daemon' stdout isn't its config: %v", err)
	}
	if cfg.Host == "" || cfg.Port == 0 || cfg.Token == "" {
		fatalf(t, "'stripe daemon' config is incomplete: host=%q port=%d has_token=%t", cfg.Host, cfg.Port, cfg.Token != "")
	}

	conn, err := grpc.NewClient(net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fatalf(t, "Failed to create gRPC client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return rpc.NewStripeCLIClient(conn), cfg
}

func withDaemonToken(token string) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "sec-x-stripe-cli", token)
}

func TestOfflineDaemonRequiresToken(t *testing.T) {
	client, cfg := startDaemon(t, nil)

	if _, err := client.Version(context.Background(), &rpc.VersionRequest{}); status.Code(err) != codes.Unauthenticated {
		errorf(t, "Expected a request without the sec-x-stripe-cli header to be rejected, got: %v", err)
	}

	// Clients written before the token existed send "1"
	if _, err := client.Version(withDaemonToken("1"), &rpc.VersionRequest{}); status.Code(err) != codes.Unauthenticated {
		errorf(t, "Expected a request with the legacy header value to be rejected, got: %v", err)
	}

	if _, err := client.Version(withDaemonToken(cfg.Token), &rpc.VersionRequest{}); err != nil {
		errorf(t, "Expected a request with the daemon's token to succeed, got: %v", err)
	}
}

func TestOfflineDaemonTriggerRejectsFixtureAPIBaseOutsideStripe(t *testing.T) {
	var serverCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled.Store(true)
	}))
	defer server.Close()

	client, cfg := startDaemon(t, map[string]string{"STRIPE_API_KEY": "sk_test_canary_placeholder"})

	raw := `{
		"_meta": {"template_version": 0},
		"fixtures": [{"name": "customer", "path": "/v1/customers", "method": "post", "api_base": "` + server.URL + `"}]
	}`
	_, err := client.Trigger(withDaemonToken(cfg.Token), &rpc.TriggerRequest{Raw: raw})
	if err == nil || !strings.Contains(err.Error(), "not an allowed Stripe API URL") {
		errorf(t, "Expected the fixture's api_base to be rejected, got: %v", err)
	}
	if serverCalled.Load() {
		errorf(t, "'stripe daemon' sent the fixture's request, with its credentials, to the fixture's api_base")
	}
}
