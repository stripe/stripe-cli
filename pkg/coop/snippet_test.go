package coop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchSDKSnippet(t *testing.T) {
	snippets := map[string]string{
		"node":   "const stripe = require('stripe')('sk_test');\nawait stripe.customers.create({name: 'Test'});",
		"python": "import stripe\nstripe.Customer.create(name='Test')",
		"ruby":   "Stripe::Customer.create(name: 'Test')",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/customers", r.URL.Query().Get("path"))
		assert.Equal(t, "post", r.URL.Query().Get("verb"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snippets)
	}))
	defer server.Close()

	// Override the endpoint for testing
	origEndpoint := snippetEndpoint
	snippetEndpoint = server.URL
	defer func() { snippetEndpoint = origEndpoint }()

	snippet, err := FetchSDKSnippet("/v1/customers", "post", map[string]string{"name": "Test"}, "node")
	require.NoError(t, err)
	assert.Contains(t, snippet, "stripe.customers.create")

	snippet, err = FetchSDKSnippet("/v1/customers", "post", nil, "python")
	require.NoError(t, err)
	assert.Contains(t, snippet, "stripe.Customer.create")
}

func TestFetchSDKSnippetInvalidLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"node": "code"})
	}))
	defer server.Close()

	origEndpoint := snippetEndpoint
	snippetEndpoint = server.URL
	defer func() { snippetEndpoint = origEndpoint }()

	_, err := FetchSDKSnippet("/v1/customers", "post", nil, "cobol")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cobol")
}

func TestFetchSDKSnippetServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	origEndpoint := snippetEndpoint
	snippetEndpoint = server.URL
	defer func() { snippetEndpoint = origEndpoint }()

	_, err := FetchSDKSnippet("/v1/customers", "post", nil, "node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestFetchSDKSnippetReturnsParamMarshalError(t *testing.T) {
	_, err := FetchSDKSnippet("/v1/customers", "post", map[string]interface{}{"bad": func() {}}, "node")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshaling snippet params")
}

func TestShouldFetchSDKSnippetRequiresParamsForMutatingCalls(t *testing.T) {
	assert.False(t, ShouldFetchSDKSnippet(&APIRequest{Path: "/v1/checkout/sessions", Method: "post"}))
	assert.False(t, ShouldFetchSDKSnippet(&APIRequest{Path: "/v1/checkout/sessions", Method: "post", Params: map[string]string{}}))
	assert.True(t, ShouldFetchSDKSnippet(&APIRequest{Path: "/v1/checkout/sessions", Method: "post", Params: map[string]string{"mode": "payment"}}))
	assert.True(t, ShouldFetchSDKSnippet(&APIRequest{Path: "/v1/invoices/in_123", Method: "get"}))
	assert.False(t, ShouldFetchSDKSnippet(&APIRequest{Path: "/v1/invoices/${node.main.create-invoice:id}", Method: "get"}))
}

func TestSDKSnippetGuidanceUsesLanguageComments(t *testing.T) {
	req := &APIRequest{Path: "/v1/checkout/sessions", Method: "post"}

	node := SDKSnippetGuidance(req, "node")
	assert.Contains(t, node, "// Blueprint request: POST /v1/checkout/sessions")
	assert.Contains(t, node, "does not include canonical request params")
	assert.Contains(t, node, "// For existing apps, derive Checkout line items")
	assert.Contains(t, node, "rather than the success URL")

	python := SDKSnippetGuidance(req, "python")
	assert.Contains(t, python, "# Blueprint request: POST /v1/checkout/sessions")
}

func TestSDKSnippetGuidanceIncludesPaymentIntentProductSafety(t *testing.T) {
	req := &APIRequest{Path: "/v1/payment_intents", Method: "post"}

	guidance := SDKSnippetGuidance(req, "node")

	assert.Contains(t, guidance, "derive amount, currency, customer identity")
	assert.Contains(t, guidance, "never by passing raw card numbers")
	assert.Contains(t, guidance, "do not accept an arbitrary destination account ID from the client")
}

func TestSDKSnippetGuidancePreservesBlueprintReferences(t *testing.T) {
	req := &APIRequest{
		Path:   "/v1/invoices/${node.main.create-invoice:id}",
		Method: "get",
	}

	guidance := SDKSnippetGuidance(req, "node")

	assert.Contains(t, guidance, "// Blueprint request: GET /v1/invoices/${node.main.create-invoice:id}")
	assert.Contains(t, guidance, "Resolve blueprint references from prior step outputs")
	assert.Contains(t, guidance, "${node.main.create-invoice:id}")
}
