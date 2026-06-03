package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli-docs-plugin/cmd"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
)

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func TestSearchCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/_endpoint/search", r.URL.Path)
		assert.Equal(t, "payment methods", r.URL.Query().Get("query"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hits":[{"title":"Accept a payment","url":"https://docs.stripe.com/payments/accept-a-payment"},{"title":"Payment Element","url":"https://docs.stripe.com/payments/elements"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"search", "payment methods"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "Accept a payment")
	assert.Contains(t, plainOutput, "stripe docs /payments/accept-a-payment")
	assert.Contains(t, plainOutput, "Payment Element")
	assert.Contains(t, plainOutput, "stripe docs /payments/elements")
}

func TestSearchCommand_MissingQuery(t *testing.T) {
	root := cmd.New().Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"search"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search: missing search query argument")
}
