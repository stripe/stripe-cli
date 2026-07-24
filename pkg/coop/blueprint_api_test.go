package coop

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

type recordingKeyProvider struct {
	key      string
	err      error
	livemode []bool
}

func (p *recordingKeyProvider) GetAPIKey(livemode bool) (string, error) {
	p.livemode = append(p.livemode, livemode)
	return p.key, p.err
}

func TestWorkbenchClientListAndRetrieve(t *testing.T) {
	listFixture, err := os.ReadFile("testdata/blueprints-list.json")
	require.NoError(t, err)
	retrieveFixture, err := os.ReadFile("testdata/blueprint-retrieve.json")
	require.NoError(t, err)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer sk_test_blueprints", r.Header.Get("Authorization"))
		assert.Equal(t, requests.StripePreviewVersionHeaderValue, r.Header.Get("Stripe-Version"))
		switch r.URL.Path {
		case workbenchBlueprintsPath:
			_, _ = w.Write(listFixture)
		case workbenchBlueprintsPath + "/sample-payment":
			_, _ = w.Write(retrieveFixture)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	profile := &recordingKeyProvider{key: "sk_test_blueprints"}
	client := NewWorkbenchClient(profile, server.URL, server.Client())
	summaries, err := client.List(context.Background())
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "Sample payment", summaries[0].Title.DefaultMessage)
	assert.Equal(t, "learning", summaries[0].BlueprintType)

	blueprint, err := client.Retrieve(context.Background(), "sample-payment")
	require.NoError(t, err)
	assert.Equal(t, "sample-payment", blueprint.Key)
	assert.Equal(t, 7, blueprint.BlueprintVersion)
	require.Len(t, blueprint.Steps, 1)
	assert.Equal(t, NodeAPIRequest, blueprint.Steps[0].Nodes[0].NodeType)
	assert.NotEmpty(t, blueprint.raw)
	assert.Equal(t, []bool{false, false}, profile.livemode)
	assert.Equal(t, []string{workbenchBlueprintsPath, workbenchBlueprintsPath + "/sample-payment"}, paths)
}

func TestWorkbenchClientReturnsStructuredErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid test key","type":"invalid_request_error","code":"api_key_invalid"}}`))
	}))
	defer server.Close()

	client := NewWorkbenchClient(&recordingKeyProvider{key: "sk_test_bad"}, server.URL, server.Client())
	_, err := client.List(context.Background())
	require.Error(t, err)
	var apiErr *WorkbenchAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "api_key_invalid", apiErr.Code)
	assert.Contains(t, err.Error(), "invalid test key")
}

func TestWorkbenchClientPropagatesAuthenticationAndDecodeErrors(t *testing.T) {
	profile := &recordingKeyProvider{err: errors.New("not logged in")}
	client := NewWorkbenchClient(profile, "https://api.example.test", nil)
	_, err := client.List(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test-mode API key")
	assert.Equal(t, []bool{false}, profile.livemode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()
	client = NewWorkbenchClient(&recordingKeyProvider{key: "sk_test_decode"}, server.URL, server.Client())
	_, err = client.List(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding Workbench blueprint response")
}

func TestWorkbenchHTTPRepositoryCompilesVariantsAndPinsSessions(t *testing.T) {
	retrieveFixture, err := os.ReadFile("testdata/blueprint-retrieve.json")
	require.NoError(t, err)

	retrieveCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case workbenchBlueprintsPath + "/sample-payment":
			retrieveCount++
			if retrieveCount == 1 {
				_, _ = w.Write(retrieveFixture)
				return
			}
			changed := strings.Replace(string(retrieveFixture), `"blueprint_version": 7`, `"blueprint_version": 8`, 1)
			changed = strings.Replace(changed, `"/v1/payment_intents"`, `"/v1/upstream_changed"`, 1)
			_, _ = w.Write([]byte(changed))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewWorkbenchClient(&recordingKeyProvider{key: "sk_test_end_to_end"}, server.URL, server.Client())
	compiled, err := LoadBlueprint(context.Background(), client, "sample-payment", map[string]string{"country": "GB"})
	require.NoError(t, err)
	params := compiled.Steps[0].Nodes[0].Request.Params.(map[string]any)
	assert.Equal(t, map[string]any{"country": "GB", "controller": "application"}, params["account"])
	assert.Equal(t, "pm_card_visa", params["payment_method"])
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, compiled.Pin.Digest)

	session := NewSessionFromBlueprint(compiled, "coop_http_pin", nil, nil)
	updated, err := LoadBlueprint(context.Background(), client, "sample-payment", nil)
	require.NoError(t, err)
	assert.Equal(t, 8, updated.Pin.BlueprintVersion)
	assert.Equal(t, "/v1/upstream_changed", updated.Steps[0].Nodes[0].Request.Path)

	require.NotNil(t, session.BlueprintPin)
	assert.Equal(t, 7, session.BlueprintPin.BlueprintVersion)
	assert.Equal(t, compiled.Pin.Digest, session.BlueprintPin.Digest)
	assert.Equal(t, "/v1/payment_intents", session.Steps[1].Nodes[0].Request.Path)
}
