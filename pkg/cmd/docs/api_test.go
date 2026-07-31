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

	cmd "github.com/stripe/stripe-cli/pkg/cmd/docs"
	"github.com/stripe/stripe-cli/pkg/docs"
	"github.com/stripe/stripe-cli/pkg/docs/markdown"
)

func TestAPICommand_MissingArgs(t *testing.T) {
	root := cmd.New().Root()
	root.SetArgs([]string{"api"})

	err := root.ExecuteContext(context.Background())
	assert.Error(t, err)
}

func TestAPICommand_FollowsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_endpoint/api-reference-locator":
			assert.Equal(t, "GET /v1/products", r.URL.Query().Get("q"))
			http.Redirect(w, r, "/api/products/list", http.StatusFound)
		case "/api/products/list":
			fmt.Fprint(w, "# List Products\n\nReturns a list of products.")
		default:
			http.NotFound(w, r)
		}
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
	root.SetArgs([]string{"api", "GET", "/v1/products"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Returns a list of")
}

func TestAPICommand_Authenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/docs/page", r.URL.Path)
		assert.Equal(t, "/_endpoint/api-reference-locator?q=GET+%2Fv1%2Fproducts", r.URL.Query().Get("path"))
		assert.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"content":"# List Products\n\nReturns a list of products."}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(
		docs.WithAPIBaseURL(server.URL),
		docs.WithAPIKey("sk_test_123"),
	)
	renderer, err := markdown.NewRenderer(markdown.WithStyle("notty"))
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"api", "GET", "/v1/products"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Returns a list of")
}

func TestAPICommand_ResourceLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_endpoint/api-reference-locator":
			assert.Equal(t, "product", r.URL.Query().Get("q"))
			http.Redirect(w, r, "/api/products", http.StatusFound)
		case "/api/products":
			fmt.Fprint(w, "# Products\n\nProducts describe items.")
		default:
			http.NotFound(w, r)
		}
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
	root.SetArgs([]string{"api", "product"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Products describe items")
}
