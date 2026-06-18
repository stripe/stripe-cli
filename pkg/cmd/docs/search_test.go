package docs_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/stripe/stripe-cli/pkg/config"

	"github.com/stripe/stripe-cli/internal/docs"
	cmd "github.com/stripe/stripe-cli/pkg/cmd/docs"
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

	// Results should be formatted as a bullet list.
	assert.Contains(t, plainOutput, "•")

	// Each title should appear before its corresponding route.
	paymentIdx := strings.Index(plainOutput, "Accept a payment")
	routeIdx := strings.Index(plainOutput, "stripe docs /payments/accept-a-payment")
	assert.Less(t, paymentIdx, routeIdx)
}

func TestSearchCommand_ColorOff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hits":[{"title":"API Keys","url":"https://docs.stripe.com/keys"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(&cliconfig.Config{Color: "off"}),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"search", "api keys"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	output := out.String()
	assert.NotContains(t, output, "\x1b[", "expected no ANSI escape codes with --color off")
	assert.Contains(t, output, "API Keys")
	assert.Contains(t, output, "stripe docs /keys")
}

func TestSearchCommand_MissingQuery(t *testing.T) {
	root := cmd.New().Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"search"})

	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search: missing search query argument")
}

func TestSearchCommand_MultiWordArgs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "payment methods", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hits":[]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"search", "payment", "methods"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)
}

func TestSearchCommand_NoTUI_FallsBackToHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/_endpoint/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hits":[{"title":"Payments","url":"https://docs.stripe.com/payments"}]}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(cmd.WithClient(client)).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"--non-interactive", "search", "payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, stripANSI(out.String()), "Payments")
}
