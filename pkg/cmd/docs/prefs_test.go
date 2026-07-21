package docs_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/stripe/stripe-cli/pkg/config"

	cmd "github.com/stripe/stripe-cli/pkg/cmd/docs"
	"github.com/stripe/stripe-cli/pkg/docs"
)

func setupPrefsTestConfig(t *testing.T) (*cliconfig.Config, func()) {
	t.Helper()
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(profilesFile, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	viper.SetConfigType("toml")
	viper.SetConfigFile(profilesFile)
	viper.SetConfigPermissions(os.FileMode(0600))

	cfg := &cliconfig.Config{
		ProfilesFile: profilesFile,
		Profile: cliconfig.Profile{
			ProfileName: "default",
		},
	}

	return cfg, func() { viper.Reset() }
}

func TestPrefsListCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/_endpoint/prefs", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"lang","category":"code","description":"Programming language","values":["ruby","python","go"],"default":"ruby"},{"id":"theme","category":null,"description":"Color theme","values":["light","dark"],"default":null}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"prefs", "list"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "lang")
	assert.Contains(t, plainOutput, "Programming language")
	assert.Contains(t, plainOutput, "ruby, python, go")
	assert.Contains(t, plainOutput, "default: ruby")
	assert.Contains(t, plainOutput, "theme")
	assert.Contains(t, plainOutput, "Color theme")
	assert.Contains(t, plainOutput, "light, dark")
	assert.Contains(t, plainOutput, "•")
}

func TestPrefsListCommand_ColorOff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"lang","category":null,"description":"Programming language","values":["ruby","python"],"default":"ruby"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(&cliconfig.Config{Color: "off"}),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"prefs", "list"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	output := out.String()
	assert.NotContains(t, output, "\x1b[", "expected no ANSI escape codes with color off")
	assert.Contains(t, output, "lang")
	assert.Contains(t, output, "Programming language")
	assert.Contains(t, output, "default: ruby")
}

func TestPrefsListCommand_NoDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"theme","category":null,"description":"Color theme","values":["light","dark"],"default":null}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"prefs", "list"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "light, dark")
	assert.NotContains(t, plainOutput, "default:")
}

func TestPrefsListCommand_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"prefs", "list"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prefs:")
}

func TestPrefsListCommand_ShowsCurrentValue(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"lang","category":null,"description":"Programming language","values":["ruby","python","go"],"default":"ruby"}]}`)
	}))
	defer server.Close()

	// Pre-set a value in config.
	require.NoError(t, cfg.Profile.WriteConfigField("docs_prefs.lang", "python"))

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(&cliconfig.Config{Color: "off", Profile: cfg.Profile}),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"prefs", "list"})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Contains(t, out.String(), "[current: python]")
}

func TestPrefsSetCommand(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"lang","category":null,"description":"Programming language","values":["ruby","python","go"],"default":"ruby"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client), cmd.WithConfig(cfg)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"prefs", "set", "lang", "python"})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Contains(t, stripANSI(out.String()), "Preference lang set to python")
	assert.Equal(t, "python", viper.GetString("default.docs_prefs.lang"))
}

func TestPrefsSetCommand_InvalidValue(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"lang","category":null,"description":"Programming language","values":["ruby","python","go"],"default":"ruby"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	root := cmd.New().WithOptions(cmd.WithClient(client), cmd.WithConfig(cfg)).Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"prefs", "set", "lang", "javascript"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
	assert.Contains(t, err.Error(), "javascript")
}

func TestPrefsSetCommand_UnknownPref(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"prefs":[{"id":"lang","category":null,"description":"Programming language","values":["ruby","python"],"default":"ruby"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	root := cmd.New().WithOptions(cmd.WithClient(client), cmd.WithConfig(cfg)).Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"prefs", "set", "unknown_pref", "value"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown preference")
}

func TestPrefsUnsetCommand(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()

	require.NoError(t, cfg.Profile.WriteConfigField("docs_prefs.lang", "python"))
	assert.Equal(t, "python", viper.GetString("default.docs_prefs.lang"))

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL("http://unused"))
	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client), cmd.WithConfig(cfg)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"prefs", "unset", "lang"})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Contains(t, stripANSI(out.String()), "Preference lang unset")
	assert.Equal(t, "", viper.GetString("default.docs_prefs.lang"))
}
