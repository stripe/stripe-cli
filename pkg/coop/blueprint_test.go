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
	list         []WorkbenchBlueprintSummary
	blueprints   map[string]*WorkbenchBlueprint
	listErr      error
	retrieveErr  error
	retrievedKey string
}

func (r *memoryBlueprintRepository) List(context.Context) ([]WorkbenchBlueprintSummary, error) {
	return r.list, r.listErr
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

func loadTestSummaries(t *testing.T) []WorkbenchBlueprintSummary {
	t.Helper()
	raw, err := os.ReadFile("testdata/blueprints-list.json")
	require.NoError(t, err)
	var response struct {
		Data []WorkbenchBlueprintSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &response))
	return response.Data
}

func TestCompileBlueprintNormalizesAPIDetails(t *testing.T) {
	blueprint, err := CompileBlueprint(loadTestBlueprint(t), map[string]string{"country": "GB"})
	require.NoError(t, err)

	assert.Equal(t, "sample-payment", blueprint.ID)
	assert.Equal(t, "Sample payment", blueprint.Title)
	assert.Equal(t, "Build and verify a representative payment flow.", blueprint.Description)
	assert.Equal(t, []string{"Payments"}, blueprint.Products)
	require.Len(t, blueprint.Steps, 1)
	assert.Equal(t, "setup", blueprint.Steps[0].Key)
	require.Len(t, blueprint.Steps[0].Outputs, 1)
	assert.Equal(t, "${node.create-payment:id}", blueprint.Steps[0].Outputs[0].Source)
	require.Len(t, blueprint.Steps[0].Nodes, 4)

	apiNode := blueprint.Steps[0].Nodes[0]
	require.NotNil(t, apiNode.Request)
	assert.Equal(t, "POST", apiNode.Request.Method)
	assert.Equal(t, "ctx_test", apiNode.Request.Headers["Stripe-Context"])
	params := apiNode.Request.Params.(map[string]any)
	assert.Equal(t, float64(2000), params["amount"])
	assert.Equal(t, "usd", params["currency"])
	assert.Equal(t, "pm_card_visa", params["payment_method"])
	assert.Equal(t, map[string]any{"country": "GB", "controller": "application"}, params["account"])
	assert.Equal(t, map[string]any{"source": "base", "baseline": "yes"}, params["metadata"])
	assert.Equal(t, "client_secret", apiNode.Request.ProcessingDetails.OutputField)
	assert.Equal(t, "Confirm the implementation calls the intended Stripe API and reuses any IDs needed by later steps.", apiNode.ReviewPrompt)
	assert.Empty(t, apiNode.ReviewCommand)

	asyncNode := blueprint.Steps[0].Nodes[1]
	require.Len(t, asyncNode.Events, 1)
	assert.Equal(t, "payment_intent.succeeded", asyncNode.Events[0].EventType)
	assert.Equal(t, "${node.create-payment:id}", asyncNode.Events[0].ObjectID)
	assert.Equal(t, "succeeded", asyncNode.Events[0].EventData["additional_properties"].(map[string]any)["status"])
	assert.Equal(t, "create-payment", asyncNode.Events[0].OnNodeComplete.NodeKey)
	assert.Equal(t, "stripe trigger payment_intent.succeeded", asyncNode.ReviewCommand)

	testNode := blueprint.Steps[0].Nodes[2]
	require.Len(t, testNode.TestRequests, 1)
	assert.Equal(t, "0", testNode.TestRequests[0].Key)
	assert.Equal(t, "acct_test", testNode.TestRequests[0].Headers["Stripe-Account"])
	assert.Contains(t, testNode.TestRequests[0].Path, "${node.create-payment:id}")
	assert.Equal(t, "GB", testNode.TestRequests[0].Params.(map[string]any)["expected_country"])

	uiNode := blueprint.Steps[0].Nodes[3]
	require.NotNil(t, uiNode.UIComponent)
	assert.Equal(t, "inline", uiNode.UIComponent.Display)
	assert.Equal(t, "ui_component.payment", uiNode.UIComponent.DisplayComponentRef.ID)
	assert.Equal(t, "${node.create-payment:client_secret}", uiNode.UIComponent.StripeElementRef["params"].(map[string]any)["client_secret"])
	require.Len(t, uiNode.UIComponent.Options, 2)
	assert.Equal(t, "Open the hosted payment", uiNode.UIComponent.Options[0].Title)
	assert.Contains(t, uiNode.UIComponent.Options[0].Link, "${node.create-payment:id}")
	require.Len(t, uiNode.UIComponent.Options[1].Requests, 2)
	assert.Contains(t, uiNode.UIComponent.Options[1].Requests[1].Path, "${node.show-payment.0:id}")

	assert.Equal(t, 7, blueprint.Pin.BlueprintVersion)
	assert.Equal(t, 4, blueprint.Pin.TemplateVersion)
	require.Len(t, blueprint.Pin.Steps, 1)
	assert.Equal(t, 3, blueprint.Pin.Steps[0].StepVersion)
	assert.Equal(t, 2, blueprint.Pin.Steps[0].TemplateVersion)
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, blueprint.Pin.Digest)
}

func TestCompileBlueprintEvaluatesConditionalStepsAndNodes(t *testing.T) {
	source := loadTestBlueprint(t)
	source.BlueprintSettings = []WorkbenchSettingGroup{{
		Key: "environment",
		Settings: []WorkbenchField{{
			Name:   "env_livemode",
			Schema: WorkbenchFieldSchema{DefaultValue: false},
		}},
	}}
	testNode := source.Steps[0].Nodes[0]
	testNode.Key = "test-node"
	testNode.IsIncluded = map[string]any{"==": []any{"${params:env_livemode}", "false"}}
	liveNode := source.Steps[0].Nodes[0]
	liveNode.Key = "live-node"
	liveNode.IsIncluded = map[string]any{"==": []any{"${params:env_livemode}", "true"}}
	source.Steps[0].Nodes = []WorkbenchBlueprintNode{testNode, liveNode}
	source.Steps[0].Outputs = nil

	liveStep := source.Steps[0]
	liveStep.Key = source.Key + "--live-only"
	liveStep.Title = MessageDescriptor{DefaultMessage: "Live only"}
	liveStep.IsIncluded = map[string]any{"==": []any{"${params:env_livemode}", "true"}}
	liveStep.Nodes = []WorkbenchBlueprintNode{liveNode}
	source.Steps = append(source.Steps, liveStep)

	testMode, err := CompileBlueprint(source, nil)
	require.NoError(t, err)
	require.Len(t, testMode.Steps, 1)
	require.Len(t, testMode.Steps[0].Nodes, 1)
	assert.Equal(t, "test-node", testMode.Steps[0].Nodes[0].Key)
	require.Len(t, testMode.Pin.Steps, 2, "the pin covers the complete retrieved snapshot")

	liveMode, err := CompileBlueprint(source, map[string]string{"env_livemode": "true"})
	require.NoError(t, err)
	require.Len(t, liveMode.Steps, 2)
	require.Len(t, liveMode.Steps[0].Nodes, 1)
	assert.Equal(t, "live-node", liveMode.Steps[0].Nodes[0].Key)
}

func TestCompileBlueprintRejectsUnknownInclusionExpressions(t *testing.T) {
	source := loadTestBlueprint(t)
	source.Steps[0].IsIncluded = map[string]any{"unknown": []any{true}}

	_, err := CompileBlueprint(source, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported inclusion operator "unknown"`)
}

func TestCompileBlueprintEvaluatesLivemodeAsTestMode(t *testing.T) {
	source := loadTestBlueprint(t)
	source.BlueprintSettings = nil
	source.Steps[0].Config.Params = map[string]string{"env_livemode": "${env:livemode}"}
	testNode := source.Steps[0].Nodes[0]
	testNode.Key = "test-node"
	testNode.IsIncluded = map[string]any{"==": []any{"${params:env_livemode}", "false"}}
	liveNode := source.Steps[0].Nodes[0]
	liveNode.Key = "live-node"
	liveNode.IsIncluded = map[string]any{"==": []any{"${params:env_livemode}", "true"}}
	source.Steps[0].Nodes = []WorkbenchBlueprintNode{testNode, liveNode}
	source.Steps[0].Outputs = nil

	compiled, err := CompileBlueprint(source, map[string]string{"env_livemode": "true"})
	require.NoError(t, err)
	require.Len(t, compiled.Steps[0].Nodes, 1)
	assert.Equal(t, "test-node", compiled.Steps[0].Nodes[0].Key)
	assert.Equal(t, "false", compiled.ResolvedSettings["env_livemode"])
}

func TestDeriveReviewCommandUsesSingleEventType(t *testing.T) {
	assert.Equal(t, "stripe trigger checkout.session.completed", deriveReviewCommand(NodeDefinition{
		Type:   NodeAsyncHandler,
		Events: []AsyncEvent{{EventType: "checkout.session.completed"}},
	}))
	assert.Empty(t, deriveReviewCommand(NodeDefinition{
		Type:   NodeAsyncHandler,
		Events: []AsyncEvent{{EventType: "first.event"}, {EventType: "second.event"}},
	}))
	assert.Empty(t, deriveReviewCommand(NodeDefinition{
		Type:   NodeAsyncHandler,
		Events: []AsyncEvent{{EventType: "event; echo unsafe"}},
	}))
	assert.Empty(t, deriveReviewCommand(NodeDefinition{Type: NodeAPIRequest}))
}

func TestCompileBlueprintMergesAllMatchingConfigurationVariants(t *testing.T) {
	defaults, err := CompileBlueprint(loadTestBlueprint(t), nil)
	require.NoError(t, err)
	defaultParams := defaults.Steps[0].Nodes[0].Request.Params.(map[string]any)
	assert.Equal(t, "pm_card_visa", defaultParams["payment_method"])
	assert.Equal(t, map[string]any{"country": "US", "controller": "application"}, defaultParams["account"])
	assert.Empty(t, defaults.Steps[0].Nodes[0].Request.ExpectedErrorType)

	selected, err := CompileBlueprint(loadTestBlueprint(t), map[string]string{
		"simulation": "declined",
		"country":    "GB",
	})
	require.NoError(t, err)
	selectedRequest := selected.Steps[0].Nodes[0].Request
	selectedParams := selectedRequest.Params.(map[string]any)
	assert.Equal(t, "pm_card_chargeDeclined", selectedParams["payment_method"])
	assert.Equal(t, map[string]any{"country": "GB", "controller": "application"}, selectedParams["account"])
	assert.Equal(t, "card_error", selectedRequest.ExpectedErrorType)
	selectedUI := selected.Steps[0].Nodes[3].UIComponent
	assert.Equal(t, "modal", selectedUI.Display)
	require.Len(t, selectedUI.Options, 1)
	assert.Equal(t, "Review the declined payment", selectedUI.Options[0].Title)
}

func TestResolveBlueprintSettingsUsesExplicitStepMapping(t *testing.T) {
	source := loadTestBlueprint(t)
	source.Steps[0].Config.Settings = map[string]string{
		"merchant_country": "${blueprint_settings.payment:country}",
	}

	defaults := resolveBlueprintSettings(source, nil)
	assert.Equal(t, "US", defaults["merchant_country"])

	selected := resolveBlueprintSettings(source, map[string]string{"country": "GB"})
	assert.Equal(t, "GB", selected["merchant_country"])

	selected = resolveBlueprintSettings(source, map[string]string{"merchant_country": "CA"})
	assert.Equal(t, "CA", selected["merchant_country"])
}

func TestBlueprintDigestPinsRetrievedSnapshot(t *testing.T) {
	source := loadTestBlueprint(t)
	first := blueprintDigest(source)
	second := loadTestBlueprint(t)

	assert.Equal(t, first, blueprintDigest(second))

	second.raw = append(second.raw, '\n')
	assert.NotEqual(t, first, blueprintDigest(second))
}

func TestResolveBlueprintKey(t *testing.T) {
	available := []WorkbenchBlueprintSummary{
		{Key: "flat-subscription"},
		{Key: "flat-fee"},
		{Key: "one-time-payment"},
	}

	exact, err := ResolveBlueprintKey(available, "flat-fee")
	require.NoError(t, err)
	assert.Equal(t, "flat-fee", exact)

	prefix, err := ResolveBlueprintKey(available, "one-time")
	require.NoError(t, err)
	assert.Equal(t, "one-time-payment", prefix)

	_, err = ResolveBlueprintKey(available, "flat")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "flat-fee, flat-subscription")

	_, err = ResolveBlueprintKey(available, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadBlueprintUsesCanonicalPrefixMatch(t *testing.T) {
	source := loadTestBlueprint(t)
	repository := &memoryBlueprintRepository{
		list:       loadTestSummaries(t),
		blueprints: map[string]*WorkbenchBlueprint{"sample-payment": source},
	}

	compiled, err := LoadBlueprint(context.Background(), repository, "sample-pay", nil)
	require.NoError(t, err)
	assert.Equal(t, "sample-payment", compiled.ID)
	assert.Equal(t, "sample-payment", repository.retrievedKey)
}

func TestLoadBlueprintWrapsRepositoryErrors(t *testing.T) {
	repository := &memoryBlueprintRepository{listErr: errors.New("network unavailable")}
	_, err := LoadBlueprint(context.Background(), repository, "sample", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing blueprints")

	repository = &memoryBlueprintRepository{
		list:        loadTestSummaries(t),
		retrieveErr: errors.New("permission denied"),
	}
	_, err = LoadBlueprint(context.Background(), repository, "sample-payment", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieving blueprint")
}

func TestValidateBlueprintReferences(t *testing.T) {
	newBlueprint := func(reference string) *Blueprint {
		return &Blueprint{
			ID: "test",
			Steps: []BlueprintStep{{
				StepDefinition: StepDefinition{Key: "setup", Title: "Setup"},
				Nodes: []NodeDefinition{
					{Key: "create-product", Request: &APIRequest{Path: "/v1/products", Method: "POST"}},
					{Key: "complete", UIComponent: &UIComponentDetails{Options: []UIComponentOption{{Requests: []APIRequest{{Path: "/first"}, {Path: "/second"}}}}}},
					{Key: "use-reference", Request: &APIRequest{Path: reference, Method: "GET"}},
				},
			}},
		}
	}

	valid := []string{
		"${node.create-product:id}",
		"${node.setup.create-product:id}",
		"${node.complete.1:id}",
		"${env:randomName}",
	}
	for _, reference := range valid {
		require.NoError(t, validateBlueprintReferences(newBlueprint(reference)), reference)
	}

	invalid := []string{
		"${node.create-product}",
		"${node.create-product:id",
		"${node:setup.create-product:id}",
		"${node.missing:id}",
	}
	for _, reference := range invalid {
		require.Error(t, validateBlueprintReferences(newBlueprint(reference)), reference)
	}
}

func TestNewSessionPinsCompiledBlueprint(t *testing.T) {
	source := loadTestBlueprint(t)
	compiled, err := CompileBlueprint(source, nil)
	require.NoError(t, err)
	session := NewSessionFromBlueprint(compiled, "coop_pin", map[string]string{"language": "go"}, nil)

	source.BlueprintVersion = 99
	source.TemplateVersion = 88
	source.Steps[0].StepVersion = 77
	source.Steps[0].Nodes[0].APIRequestDetails.Fixture.Path = "/v1/changed_upstream"
	source.raw = nil
	changed, err := CompileBlueprint(source, nil)
	require.NoError(t, err)

	require.NotNil(t, session.BlueprintPin)
	assert.Equal(t, 7, session.BlueprintPin.BlueprintVersion)
	assert.Equal(t, 4, session.BlueprintPin.TemplateVersion)
	assert.Equal(t, 3, session.BlueprintPin.Steps[0].StepVersion)
	assert.NotEqual(t, changed.Pin.Digest, session.BlueprintPin.Digest)
	assert.Equal(t, "/v1/payment_intents", session.Steps[1].Nodes[0].Request.Path)
	assert.Equal(t, "${node.create-payment:id}", session.Steps[1].Outputs[0].Source)
	assert.Equal(t, "success", session.Settings["simulation"])
	assert.Equal(t, "US", session.Settings["country"])
	assert.Equal(t, "go", session.Settings["language"])

	encoded, err := json.Marshal(session)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"blueprint_version":7`)
	assert.True(t, strings.Contains(string(encoded), `"digest":"sha256:`))
}
