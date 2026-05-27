package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-cli-docs-plugin/cmd"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/markdown"
)

func TestNew(t *testing.T) {
	root := cmd.New().Root()

	assert.Equal(t, "docs <path>", root.Use)
	assert.NotEmpty(t, root.Short)
}

func TestRootPrefixesPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/connect/accounts", r.URL.Path)
		fmt.Fprint(w, "# Accounts\n\nManage connected accounts.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"connect", "accounts"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Accounts")
}

func TestFetchPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept payments with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Payments")
}

func TestVersionCommand(t *testing.T) {
	root := cmd.New().WithOptions(cmd.WithVersion("0.0.1")).Root()

	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetArgs([]string{"version"})

	err := root.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "stripe docs version 0.0.1\n", out.String())
}
