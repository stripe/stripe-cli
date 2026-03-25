package pluginhints

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCmd builds a pluginHintCmd with all side effects mocked out.
func newTestCmd(name string, privatePreview bool) *pluginHintCmd {
	p := &pluginHintCmd{
		name:           name,
		description:    "Test description.",
		privatePreview: privatePreview,
		stdout:         &bytes.Buffer{},
		stdin:          strings.NewReader(""),
	}
	p.Command = &cobra.Command{Use: name, RunE: p.run}
	return p
}

func (p *pluginHintCmd) output() string {
	return p.stdout.(*bytes.Buffer).String()
}

// --- run ---

func TestRun_PluginFound_CallsPromptInstall(t *testing.T) {
	p := newTestCmd("generate", true)
	installCalled := false
	p.lookupFn = func(ctx context.Context) error { return nil }
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.Contains(t, p.output(), "The \"generate\" plugin is required")
}

func TestRun_PluginNotFound_PrivatePreviewFalse_ReturnsNil(t *testing.T) {
	p := newTestCmd("apps", false)
	p.lookupFn = func(ctx context.Context) error { return errors.New("not found") }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.Empty(t, p.output())
}

func TestRun_PluginNotFound_PrivatePreviewTrue_CallsSuggestNotAvailable(t *testing.T) {
	p := newTestCmd("generate", true)
	p.lookupFn = func(ctx context.Context) error { return errors.New("not found") }
	p.accountIDFn = func() (string, error) { return "acct_123", nil }
	p.openBrowserFn = func(url string) error { return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.Contains(t, p.output(), "private preview")
	assert.Contains(t, p.output(), "acct_123")
}

// --- promptInstall ---

func TestPromptInstall_EnterKey_InstallsPlugin(t *testing.T) {
	p := newTestCmd("generate", true)
	p.stdin = strings.NewReader("\n")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.promptInstall(context.Background())

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.Contains(t, p.output(), "installation complete")
}

func TestPromptInstall_OtherInput_CancelsInstall(t *testing.T) {
	p := newTestCmd("generate", true)
	p.stdin = strings.NewReader("n\n")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.promptInstall(context.Background())

	assert.EqualError(t, err, "installation canceled")
	assert.False(t, installCalled)
}

func TestPromptInstall_InstallError_ReturnsError(t *testing.T) {
	p := newTestCmd("generate", true)
	p.stdin = strings.NewReader("\n")
	p.installFn = func(ctx context.Context) error { return errors.New("install failed") }

	err := p.promptInstall(context.Background())

	assert.EqualError(t, err, "install failed")
}

// --- suggestNotAvailable ---

func TestSuggestNotAvailable_NoAccountID_ReturnsLoginError(t *testing.T) {
	p := newTestCmd("generate", true)
	p.accountIDFn = func() (string, error) { return "", nil }

	err := p.suggestNotAvailable()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe login")
}

func TestSuggestNotAvailable_AccountIDError_ReturnsLoginError(t *testing.T) {
	p := newTestCmd("generate", true)
	p.accountIDFn = func() (string, error) { return "", errors.New("not configured") }

	err := p.suggestNotAvailable()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe login")
}

func TestSuggestNotAvailable_EnterKey_OpensBrowser(t *testing.T) {
	p := newTestCmd("generate", true)
	p.stdin = strings.NewReader("\n")
	p.accountIDFn = func() (string, error) { return "acct_123", nil }
	browserURL := ""
	p.openBrowserFn = func(url string) error { browserURL = url; return nil }

	err := p.suggestNotAvailable()

	require.NoError(t, err)
	assert.Equal(t, accessRequestURL, browserURL)
	assert.Contains(t, p.output(), "Opening")
}

func TestSuggestNotAvailable_OtherInput_DoesNotOpenBrowser(t *testing.T) {
	p := newTestCmd("generate", true)
	p.stdin = strings.NewReader("n\n")
	p.accountIDFn = func() (string, error) { return "acct_123", nil }
	browserOpened := false
	p.openBrowserFn = func(url string) error { browserOpened = true; return nil }

	err := p.suggestNotAvailable()

	require.NoError(t, err)
	assert.False(t, browserOpened)
}

func TestSuggestNotAvailable_ShowsAccountID(t *testing.T) {
	p := newTestCmd("generate", true)
	p.stdin = strings.NewReader("n\n")
	p.accountIDFn = func() (string, error) { return "acct_abc456", nil }
	p.openBrowserFn = func(url string) error { return nil }

	err := p.suggestNotAvailable()

	require.NoError(t, err)
	assert.Contains(t, p.output(), "acct_abc456")
}
