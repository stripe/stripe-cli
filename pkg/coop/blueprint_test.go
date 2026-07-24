package coop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryBlueprintRepository struct {
	blueprints   map[string]*WorkbenchBlueprint
	retrieveErr  error
	retrievedKey string
}

func (r *memoryBlueprintRepository) List(context.Context) ([]WorkbenchBlueprintSummary, error) {
	return nil, nil
}

func (r *memoryBlueprintRepository) Retrieve(_ context.Context, key string) (*WorkbenchBlueprint, error) {
	r.retrievedKey = key
	if r.retrieveErr != nil {
		return nil, r.retrieveErr
	}
	blueprint, ok := r.blueprints[key]
	if !ok {
		return nil, errors.New("missing fixture")
	}
	return blueprint, nil
}

func loadTestBlueprint(t *testing.T) *WorkbenchBlueprint {
	t.Helper()
	raw, err := os.ReadFile("testdata/blueprint-retrieve.json")
	require.NoError(t, err)
	var blueprint WorkbenchBlueprint
	require.NoError(t, json.Unmarshal(raw, &blueprint))
	blueprint.raw = raw
	return &blueprint
}

func TestResolveBlueprintKeepsWorkbenchShape(t *testing.T) {
	blueprint, settings, params, err := resolveBlueprint(loadTestBlueprint(t), map[string]string{"country": "GB"}, nil)
	require.NoError(t, err)

	assert.Equal(t, "sample-payment", blueprint.Key)
	assert.Equal(t, "Sample payment", blueprint.Title.DefaultMessage)
	assert.Equal(t, "Build and verify a representative payment flow.", blueprint.Description.DefaultMessage)
	assert.Equal(t, []string{"Payments"}, blueprint.Metadata.Products)
	assert.Equal(t, "GB", settings["country"])
	assert.Equal(t, "2000", params["amount"])
	require.Len(t, blueprint.Steps, 1)
	assert.Equal(t, "sample-payment--setup", blueprint.Steps[0].Key)
	assert.Equal(t, "${blueprint_settings.payment:simulation}", blueprint.Steps[0].Settings["simulation"])
	assert.Equal(t, "${blueprint_params.payment:amount}", blueprint.Steps[0].Params["amount"])
	require.Len(t, blueprint.Steps[0].SettingsSchema, 1)
	require.Len(t, blueprint.Steps[0].ParamsSchema, 1)
	require.Len(t, blueprint.Steps[0].Outputs, 1)
	assert.Equal(t, "${node.create-payment:id}", blueprint.Steps[0].Outputs[0].Source)
	require.Len(t, blueprint.Steps[0].Nodes, 4)

	apiNode := blueprint.Steps[0].Nodes[0]
	require.NotNil(t, apiNode.APIRequestDetails)
	request := apiNode.APIRequestDetails.Fixture
	assert.Equal(t, "POST", request.Method)
	assert.Equal(t, "ctx_test", request.Headers["Stripe-Context"])
	assert.Empty(t, request.ConfiguredDetails)
	assert.Equal(t, "client_secret", request.ProcessingDetails.OutputField)
	assert.Equal(t, map[string]any{
		"amount":         float64(2000),
		"currency":       "usd",
		"payment_method": "pm_card_visa",
		"account":        map[string]any{"country": "GB", "controller": "application"},
		"metadata":       map[string]any{"source": "base", "baseline": "yes"},
	}, request.Params)

	asyncNode := blueprint.Steps[0].Nodes[1]
	require.NotNil(t, asyncNode.AsyncHandlerDetails)
	require.Len(t, asyncNode.AsyncHandlerDetails.Events, 1)
	assert.Equal(t, "payment_intent.succeeded", asyncNode.AsyncHandlerDetails.Events[0].EventType)
	assert.Equal(t, "${node.create-payment:id}", asyncNode.AsyncHandlerDetails.Events[0].ObjectID)

	testNode := blueprint.Steps[0].Nodes[2]
	require.NotNil(t, testNode.TestHelperDetails)
	require.Len(t, testNode.TestHelperDetails.Requests, 1)
	assert.Equal(t, "0", testNode.TestHelperDetails.Requests[0].Key)
	assert.Equal(t, "acct_test", testNode.TestHelperDetails.Requests[0].Headers["Stripe-Account"])
	assert.Equal(t, "GB", testNode.TestHelperDetails.Requests[0].Params["expected_country"])

	uiNode := blueprint.Steps[0].Nodes[3]
	require.NotNil(t, uiNode.UIComponentDetails)
	assert.Equal(t, "inline", uiNode.UIComponentDetails.Display)
	assert.Equal(t, "ui_component.payment", uiNode.UIComponentDetails.DisplayComponentRef.ID)
	require.Len(t, uiNode.UIComponentDetails.Options, 2)
	assert.Equal(t, "Open the hosted payment", uiNode.UIComponentDetails.Options[0].Title.DefaultMessage)
	assert.Contains(t, uiNode.UIComponentDetails.Options[0].Link, "${node.create-payment:id}")
	require.Len(t, uiNode.UIComponentDetails.Options[1].Requests, 2)
}

func TestResolveBlueprintMergesEveryMatchingVariantWithoutMutatingSource(t *testing.T) {
	source := loadTestBlueprint(t)
	defaults, _, _, err := resolveBlueprint(source, nil, nil)
	require.NoError(t, err)
	defaultRequest := defaults.Steps[0].Nodes[0].APIRequestDetails.Fixture
	assert.Equal(t, "pm_card_visa", defaultRequest.Params["payment_method"])
	assert.Equal(t, map[string]any{"country": "US", "controller": "application"}, defaultRequest.Params["account"])
	assert.Empty(t, defaultRequest.ExpectedErrorType)

	selected, _, _, err := resolveBlueprint(source, map[string]string{
		"simulation": "declined",
		"country":    "GB",
	}, nil)
	require.NoError(t, err)
	selectedRequest := selected.Steps[0].Nodes[0].APIRequestDetails.Fixture
	assert.Equal(t, "pm_card_chargeDeclined", selectedRequest.Params["payment_method"])
	assert.Equal(t, map[string]any{"country": "GB", "controller": "application"}, selectedRequest.Params["account"])
	assert.Equal(t, "card_error", selectedRequest.ExpectedErrorType)
	selectedUI := selected.Steps[0].Nodes[3].UIComponentDetails
	assert.Equal(t, "modal", selectedUI.Display)
	require.Len(t, selectedUI.Options, 1)
	assert.Equal(t, "Review the declined payment", selectedUI.Options[0].Title.DefaultMessage)

	assert.NotEmpty(t, source.Steps[0].Nodes[0].APIRequestDetails.Fixture.ConfiguredDetails)
	assert.NotContains(t, source.Steps[0].Nodes[0].APIRequestDetails.Fixture.Params, "payment_method")
}

func TestResolveBlueprintAppliesSingleStaticConfiguredDetail(t *testing.T) {
	source := loadTestBlueprint(t)
	request := &source.Steps[0].Nodes[0].APIRequestDetails.Fixture
	request.ConfiguredDetails = []WorkbenchConfiguredDetails{{
		ConfigValue: map[string]string{"us": "US"},
		Params:      map[string]any{"country": "US"},
	}}

	resolved, _, _, err := resolveBlueprint(source, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "US", resolved.Steps[0].Nodes[0].APIRequestDetails.Fixture.Params["country"])
}

func TestResolveBlueprintSettingsUsesExplicitStepMappingAndTestMode(t *testing.T) {
	source := loadTestBlueprint(t)
	source.Steps[0].Settings = map[string]string{
		"merchant_country": "${blueprint_settings.payment:country}",
	}

	defaults := resolveBlueprintSettings(source, nil)
	assert.Equal(t, "US", defaults["merchant_country"])

	selected := resolveBlueprintSettings(source, map[string]string{"country": "GB"})
	assert.Equal(t, "GB", selected["merchant_country"])

	selected = resolveBlueprintSettings(source, map[string]string{"merchant_country": "CA"})
	assert.Equal(t, "CA", selected["merchant_country"])
}

func TestResolveBlueprintParamsUsesDefaultsSelectionsAndTestMode(t *testing.T) {
	source := loadTestBlueprint(t)
	source.Steps[0].Params["env_livemode"] = "${env:livemode}"

	defaults := resolveBlueprintParams(source, nil)
	assert.Equal(t, "2000", defaults["amount"])
	assert.Equal(t, "false", defaults["env_livemode"])

	selected := resolveBlueprintParams(source, map[string]string{"amount": "3500"})
	assert.Equal(t, "3500", selected["amount"])
}

func TestResolveBlueprintAppliesKnownInclusionConditions(t *testing.T) {
	source := loadTestBlueprint(t)
	source.Steps[0].Params["env_livemode"] = "${env:livemode}"
	source.Steps[0].Nodes[0].IsIncluded = map[string]any{
		"==": []any{"${params:env_livemode}", "false"},
	}
	source.Steps[0].Nodes[1].IsIncluded = map[string]any{
		"==": []any{"${params:env_livemode}", "true"},
	}

	resolved, _, _, err := resolveBlueprint(source, nil, nil)
	require.NoError(t, err)
	require.Len(t, resolved.Steps, 1)
	require.Len(t, resolved.Steps[0].Nodes, 3)
	assert.Equal(t, "create-payment", resolved.Steps[0].Nodes[0].Key)
	assert.NotContains(t, []string{
		resolved.Steps[0].Nodes[0].Key,
		resolved.Steps[0].Nodes[1].Key,
		resolved.Steps[0].Nodes[2].Key,
	}, "handle-payment")
}

func TestDeriveReviewMetadataUsesSingleEventType(t *testing.T) {
	node := WorkbenchBlueprintNode{
		NodeType: NodeAsyncHandler,
		AsyncHandlerDetails: &WorkbenchAsyncHandlerDetails{
			Events: []AsyncEvent{{EventType: "checkout.session.completed"}},
		},
	}
	assert.Equal(t, "stripe trigger checkout.session.completed", deriveReviewCommand(node))
	assert.Contains(t, deriveReviewPrompt(node), "stripe trigger checkout.session.completed")

	node.AsyncHandlerDetails.Events = append(node.AsyncHandlerDetails.Events, AsyncEvent{EventType: "second.event"})
	assert.Empty(t, deriveReviewCommand(node))
	node.AsyncHandlerDetails.Events = []AsyncEvent{{EventType: "event; echo unsafe"}}
	assert.Empty(t, deriveReviewCommand(node))
	assert.Empty(t, deriveReviewCommand(WorkbenchBlueprintNode{NodeType: NodeAPIRequest}))
}

func TestBlueprintDigestPinsRetrievedSnapshot(t *testing.T) {
	source := loadTestBlueprint(t)
	first := blueprintDigest(source)
	second := loadTestBlueprint(t)

	assert.Equal(t, first, blueprintDigest(second))
	second.raw = append(second.raw, '\n')
	assert.NotEqual(t, first, blueprintDigest(second))
}

func TestLoadBlueprintRetrievesExactKey(t *testing.T) {
	source := loadTestBlueprint(t)
	repository := &memoryBlueprintRepository{
		blueprints: map[string]*WorkbenchBlueprint{"sample-payment": source},
	}

	loaded, err := LoadBlueprint(context.Background(), repository, "sample-payment")
	require.NoError(t, err)
	assert.Same(t, source, loaded)
	assert.Equal(t, "sample-payment", repository.retrievedKey)
}

func TestLoadBlueprintDoesNotResolvePrefixes(t *testing.T) {
	repository := &memoryBlueprintRepository{
		blueprints: map[string]*WorkbenchBlueprint{"sample-payment": loadTestBlueprint(t)},
	}

	_, err := LoadBlueprint(context.Background(), repository, "sample-pay")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `retrieving blueprint "sample-pay"`)
	assert.Equal(t, "sample-pay", repository.retrievedKey)
}

func TestLoadBlueprintWrapsRepositoryErrors(t *testing.T) {
	repository := &memoryBlueprintRepository{retrieveErr: errors.New("permission denied")}
	_, err := LoadBlueprint(context.Background(), repository, "sample-payment")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieving blueprint")
}

func TestLoadBlueprintRejectsEmptyResponse(t *testing.T) {
	repository := &memoryBlueprintRepository{
		blueprints: map[string]*WorkbenchBlueprint{"sample-payment": nil},
	}
	_, err := LoadBlueprint(context.Background(), repository, "sample-payment")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestNewSessionPinsEffectiveWorkbenchDefinition(t *testing.T) {
	source := loadTestBlueprint(t)
	session, err := NewSessionFromBlueprint(source, "coop_pin", map[string]string{"language": "go"}, nil)
	require.NoError(t, err)

	source.BlueprintVersion = 99
	source.TemplateVersion = 88
	source.Steps[0].StepVersion = 77
	source.Steps[0].Nodes[0].APIRequestDetails.Fixture.Path = "/v1/changed_upstream"
	source.raw = nil
	changed, err := NewSessionFromBlueprint(source, "coop_changed", nil, nil)
	require.NoError(t, err)

	require.NotNil(t, session.BlueprintDefinition)
	assert.Equal(t, "Sample payment", session.BlueprintDefinition.Title.DefaultMessage)
	require.NotNil(t, session.BlueprintPin)
	assert.Equal(t, 7, session.BlueprintPin.BlueprintVersion)
	assert.Equal(t, 4, session.BlueprintPin.TemplateVersion)
	assert.Equal(t, 3, session.BlueprintPin.Steps[0].StepVersion)
	assert.NotEqual(t, changed.BlueprintPin.Digest, session.BlueprintPin.Digest)
	assert.Equal(t, "/v1/payment_intents", session.Steps[1].Nodes[0].Request().Path)
	assert.Equal(t, "${node.create-payment:id}", session.Steps[1].Outputs[0].Source)
	assert.Equal(t, "success", session.Settings["simulation"])
	assert.Equal(t, "US", session.Settings["country"])
	assert.Equal(t, "go", session.Settings["language"])
	assert.Equal(t, "2000", session.Params["amount"])

	encoded, err := json.Marshal(session)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"blueprint_definition"`)
	assert.Contains(t, string(encoded), `"api_request_details"`)
	assert.NotContains(t, string(encoded), `"configured_details"`)
	assert.Contains(t, string(encoded), `"blueprint_version":7`)
	assert.True(t, strings.Contains(string(encoded), `"digest":"sha256:`))
}
