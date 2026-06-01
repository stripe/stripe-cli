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

func TestSearchCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/_endpoint/search", r.URL.Path)
		assert.Equal(t, "payment methods", r.URL.Query().Get("query"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hits":[{"title":"Accept a payment","route":"/payments/accept-a-payment"},{"title":"Payment Element","route":"/payments/elements"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer(markdown.WithStyle("notty"))
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"search", "payment methods"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Accept a payment")
	assert.Contains(t, out.String(), "/payments/accept-a-payment")
	assert.Contains(t, out.String(), "Payment Element")
	assert.Contains(t, out.String(), "/payments/elements")
}

func TestSearchCommand_MissingQuery(t *testing.T) {
	root := cmd.New().Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"search"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search: missing search query argument")
}
