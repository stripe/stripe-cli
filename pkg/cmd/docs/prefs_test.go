package docs_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/stripe/stripe-cli/pkg/config"

	cmd "github.com/stripe/stripe-cli/pkg/cmd/docs"
	"github.com/stripe/stripe-cli/pkg/docs"
)

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
