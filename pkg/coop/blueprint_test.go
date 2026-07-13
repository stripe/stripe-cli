package coop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBlueprint(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)
	assert.Equal(t, "one-time-payment", bp.ID)
	assert.Equal(t, "Accept a one-time payment", bp.Title)
	assert.Contains(t, bp.Products, "Payments")
	assert.Len(t, bp.Steps, 3)
	assert.Equal(t, "setup-chapter", bp.Steps[0].Key)
	assert.Equal(t, NodeAPIRequest, bp.Steps[0].Nodes[0].Type)
}

func TestAllEmbeddedBlueprintsHaveQualityMetadata(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)
	require.NotEmpty(t, ids)

	weakPhrases := []string{
		"do the thing",
		"verify it works",
		"todo",
		"tbd",
		"placeholder",
		"lorem ipsum",
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			bp, err := LoadBlueprint(id)
			require.NoError(t, err)

			assertQualityText(t, "blueprint title", bp.Title, 4, weakPhrases)
			if bp.Description != "" {
				assertQualityText(t, "blueprint description", bp.Description, 20, weakPhrases)
			}
			assert.True(t, bp.Description != "" || len(bp.Products) > 0, "blueprint should include description or product metadata")

			for _, ch := range bp.Steps {
				assertQualityText(t, "step title "+ch.Key, ch.Title, 4, weakPhrases)
				for _, n := range ch.Nodes {
					assertQualityText(t, "node title "+n.Key, n.Title, 4, weakPhrases)
					assert.NotEqual(t, "api request", strings.ToLower(strings.TrimSpace(n.Title)), "node %q should have a product-specific title", n.Key)
					if n.Description != "" {
						assertQualityText(t, "node description "+n.Key, n.Description, 20, weakPhrases)
					}
					if !n.AutoConfirm {
						assertQualityText(t, "node review prompt "+n.Key, n.ReviewPrompt, 20, weakPhrases)
						assertObservableGuidance(t, n.Key, n.ReviewPrompt)
					}

					switch n.Type {
					case NodeAPIRequest:
						require.NotNil(t, n.Request, "apiRequest node %q should have request metadata", n.Key)
					case NodeAsyncHandler:
						assert.NotEmpty(t, n.Events, "asyncHandler node %q should name webhook events to verify", n.Key)
					case NodeCLICommand, NodeTestHelper, NodeSetUpWebhooks:
						if n.Description != "" {
							assertObservableGuidance(t, n.Key, n.Description)
						}
					}
				}
			}
		})
	}
}

func assertQualityText(t *testing.T, label, value string, minLen int, weakPhrases []string) {
	t.Helper()
	trimmed := strings.TrimSpace(value)
	require.NotEmpty(t, trimmed, "%s should not be empty", label)
	assert.GreaterOrEqual(t, len(trimmed), minLen, "%s should be specific enough", label)

	lower := strings.ToLower(trimmed)
	for _, phrase := range weakPhrases {
		assert.NotContains(t, lower, phrase, "%s contains weak placeholder text", label)
	}
}

func assertObservableGuidance(t *testing.T, key, description string) {
	t.Helper()
	lower := strings.ToLower(description)
	observableTerms := []string{"verify", "confirm", "report", "check", "run", "summarize", "ask", "open"}
	for _, term := range observableTerms {
		if strings.Contains(lower, term) {
			return
		}
	}
	assert.Failf(t, "weak verification guidance", "node %q should name an observable check or reported outcome", key)
}

func TestLoadBlueprintNotFound(t *testing.T) {
	_, err := LoadBlueprint("nonexistent-blueprint")
	assert.Error(t, err)
}

func TestValidateBlueprintReferences(t *testing.T) {
	newBlueprint := func(reference string) *Blueprint {
		return &Blueprint{
			ID: "test",
			Steps: []BlueprintStep{{
				StepDefinition: StepDefinition{Key: "setup", Title: "Setup"},
				Nodes: []NodeDefinition{
					{Key: "create-product", Request: &APIRequest{Path: "/v1/products", Method: "post"}},
					{Key: "create-clock", TestRequests: []TestHelperRequest{{
						Key: "create-clock-request",
						APIRequest: APIRequest{
							Path:   "/v1/test_helpers/test_clocks",
							Method: "post",
						},
					}}},
					{Key: "wait-for-invoice"},
					{Key: "use-reference", Request: &APIRequest{Path: reference, Method: "get"}},
				},
			}},
		}
	}

	tests := []struct {
		name      string
		reference string
		wantError string
	}{
		{name: "direct node", reference: "${node.setup.create-product:default_price}"},
		{name: "named request", reference: "${node.setup.create-clock.create-clock-request:id}"},
		{name: "numeric result", reference: "${node.setup.wait-for-invoice.0:id}"},
		{name: "nested field path", reference: "${node.setup.create-product:data[0].price.id}"},
		{name: "non-node interpolation", reference: "${env:randomName}"},
		{name: "different placeholder with node prefix", reference: "${nodeVersion}"},
		{name: "missing field delimiter", reference: "${node.setup.create-product}", wantError: "malformed node reference"},
		{name: "missing closing brace", reference: "${node.setup.create-product:id", wantError: "malformed node reference"},
		{name: "missing namespace dot", reference: "${node:setup.create-product:id}", wantError: "malformed node reference"},
		{name: "empty reference", reference: "${node.:id}", wantError: "malformed node reference"},
		{name: "empty field", reference: "${node.setup.create-product:}", wantError: "malformed node reference"},
		{name: "unclosed reference before another", reference: "${node.setup.create-product:id/${node.setup.create-product:id}", wantError: "malformed node reference"},
		{name: "unknown node", reference: "${node.old.create-product:id}", wantError: "unknown node reference"},
		{name: "unknown named request", reference: "${node.setup.create-clock.old-request:id}", wantError: "unknown node reference"},
		{name: "unknown second reference", reference: "${node.setup.create-product:id}/${node.old.create-product:id}", wantError: "unknown node reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBlueprintReferences(newBlueprint(tt.reference))
			if tt.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestLoadBlueprintPrefixMatch(t *testing.T) {
	bp, err := LoadBlueprint("setup-future")
	require.NoError(t, err)
	assert.Equal(t, "setup-future-payments", bp.ID)
}

func TestLoadBlueprintPrefixMatchUnique(t *testing.T) {
	bp, err := LoadBlueprint("one-time")
	require.NoError(t, err)
	assert.Equal(t, "one-time-payment", bp.ID)
}

func TestLoadBlueprintPrefixMatchAmbiguous(t *testing.T) {
	// "flat" matches both "flat-fee-and-overages" and "flat-subscription-with-entitlements"
	_, err := LoadBlueprint("flat")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
}

func TestListBlueprints(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)
	assert.Contains(t, ids, "one-time-payment")
	assert.Contains(t, ids, "setup-future-payments")
}

func TestNewSessionFromBlueprint(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	session := NewSessionFromBlueprint(
		bp,
		"coop_test123",
		map[string]string{"language": "node"},
		map[string]string{"account_name": "Jenny Rosen"},
	)

	assert.Equal(t, "coop_test123", session.ID)
	assert.Equal(t, "one-time-payment", session.Blueprint)
	assert.Equal(t, SessionActive, session.Status)
	assert.Equal(t, "node", session.Settings["language"])
	assert.Equal(t, "Jenny Rosen", session.Params["account_name"])
	// 3 blueprint steps + 1 prepended context step
	assert.Len(t, session.Steps, 4)

	// First step is always the context-gathering step
	assert.Equal(t, "context-step", session.Steps[0].Key)
	assert.Equal(t, "Understand the project", session.Steps[0].Nodes[0].Title)

	// All nodes should be pending
	for _, ch := range session.Steps {
		for _, n := range ch.Nodes {
			assert.Equal(t, NodePending, n.State)
		}
	}

	// Total nodes = blueprint nodes (4) + context node (1)
	assert.Equal(t, 5, session.TotalNodes())

	assert.NotEmpty(t, session.Steps[1].Nodes[0].ReviewPrompt)
	assert.Equal(t, "stripe trigger checkout.session.completed", session.Steps[3].Nodes[0].ReviewCommand)
}

func TestListBlueprintsWithMetadata(t *testing.T) {
	bps, err := ListBlueprintsWithMetadata()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(bps), 2)

	found := false
	for _, bp := range bps {
		if bp.ID == "setup-future-payments" {
			found = true
			assert.Equal(t, "Save a card for future payments", bp.Title)
		}
	}
	assert.True(t, found, "expected to find setup-future-payments")
}

func TestLoadBlueprintStepStructure(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	// Verify step keys are unique
	keys := make(map[string]bool)
	for _, ch := range bp.Steps {
		assert.False(t, keys[ch.Key], "duplicate step key: %s", ch.Key)
		keys[ch.Key] = true
		assert.NotEmpty(t, ch.Title)
		assert.NotEmpty(t, ch.Nodes)

		// Verify node keys are unique within step
		nodeKeys := make(map[string]bool)
		for _, n := range ch.Nodes {
			assert.False(t, nodeKeys[n.Key], "duplicate node key: %s", n.Key)
			nodeKeys[n.Key] = true
			assert.NotEmpty(t, n.Title)
			assert.NotEmpty(t, n.Type)
		}
	}
}

func TestLoadBlueprintNodeTypes(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	typesSeen := make(map[NodeType]bool)
	for _, ch := range bp.Steps {
		for _, n := range ch.Nodes {
			typesSeen[n.Type] = true
		}
	}

	assert.True(t, typesSeen[NodeAPIRequest], "expected apiRequest nodes")
	assert.True(t, typesSeen[NodeUIComponent], "expected uiComponent nodes")
	assert.True(t, typesSeen[NodeAsyncHandler], "expected asyncHandler nodes")
}

func TestLoadBlueprintAPIRequestHasRequest(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	for _, ch := range bp.Steps {
		for _, n := range ch.Nodes {
			if n.Type == NodeAPIRequest {
				assert.NotNil(t, n.Request, "apiRequest node %q should have request field", n.Key)
				assert.NotEmpty(t, n.Request.Path)
				assert.NotEmpty(t, n.Request.Method)
			}
		}
	}
}

func TestLoadBlueprintAsyncHandlerHasEvents(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	for _, ch := range bp.Steps {
		for _, n := range ch.Nodes {
			if n.Type == NodeAsyncHandler {
				assert.NotEmpty(t, n.Events, "asyncHandler node %q should have events", n.Key)
			}
		}
	}
}

func TestNewSessionFromBlueprintPreservesRequest(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	session := NewSessionFromBlueprint(bp, "test_123", nil, nil)

	// First blueprint node (after context step) is apiRequest — should preserve the request
	firstBlueprintNode := session.Steps[1].Nodes[0]
	assert.Equal(t, NodeAPIRequest, firstBlueprintNode.Type)
	assert.NotNil(t, firstBlueprintNode.Request)
	assert.Equal(t, "/v1/products", firstBlueprintNode.Request.Path)
	assert.Equal(t, "post", firstBlueprintNode.Request.Method)
}

func TestNewSessionFromBlueprintPreservesEvents(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	session := NewSessionFromBlueprint(bp, "test_123", nil, nil)

	// Find the asyncHandler node
	for _, ch := range session.Steps {
		for _, n := range ch.Nodes {
			if n.Type == NodeAsyncHandler {
				assert.Contains(t, n.Events, "checkout.session.completed")
				return
			}
		}
	}
	t.Fatal("expected to find asyncHandler node")
}

func TestEmbeddedBlueprintsUseCanonicalJSON(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)
	require.NotEmpty(t, ids)

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			raw, err := blueprintFS.ReadFile("blueprints/" + id + ".json")
			require.NoError(t, err)

			var bp Blueprint
			require.NoError(t, json.Unmarshal(raw, &bp))

			normalized, err := json.MarshalIndent(bp, "", "  ")
			require.NoError(t, err)
			normalized = append(normalized, '\n')

			assert.Equal(t, string(normalized), string(raw), "blueprint JSON should be normalized through the Blueprint schema")
		})
	}
}

func TestEmbeddedBlueprintsDoNotCarryAPIRequestKeys(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)
	require.NotEmpty(t, ids)

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			raw, err := blueprintFS.ReadFile("blueprints/" + id + ".json")
			require.NoError(t, err)

			var document any
			require.NoError(t, json.Unmarshal(raw, &document))
			assertNoRequestKey(t, document)

			bp, err := LoadBlueprint(id)
			require.NoError(t, err)
			assertNoAPIRequestNodeKeyInterpolation(t, bp, string(raw))
		})
	}
}

func assertNoRequestKey(t *testing.T, value any) {
	t.Helper()

	switch v := value.(type) {
	case map[string]any:
		if request, ok := v["request"].(map[string]any); ok {
			assert.NotContains(t, request, "key", "apiRequest nodes only carry one request, so request.key is redundant")
		}
		for _, child := range v {
			assertNoRequestKey(t, child)
		}
	case []any:
		for _, child := range v {
			assertNoRequestKey(t, child)
		}
	}
}

func assertNoAPIRequestNodeKeyInterpolation(t *testing.T, bp *Blueprint, raw string) {
	t.Helper()

	for _, step := range bp.Steps {
		for _, node := range step.Nodes {
			if node.Type != NodeAPIRequest {
				continue
			}
			assert.NotContains(
				t,
				raw,
				"${node."+step.Key+"."+node.Key+".",
				"apiRequest node interpolation should not include request keys",
			)
		}
	}
}

func TestEmbeddedBlueprintTopologyMatchesSourceDefinitions(t *testing.T) {
	// These topologies are copied from pay-server Workbench blueprint definitions.
	// Do not add CLI-only steps or nodes here; update the source blueprint and then
	// refresh this subset from pay-server.
	expected := map[string][][]string{
		"flat-fee-and-overages": {
			{"create-customer-chapter", "createCustomer"},
			{"create-pricing-plan-chapter", "createEmptyPricingPlan", "createMeter"},
			{"create-rate-card-chapter", "createRateCard", "createMeteredItem", "addGraduatedRateToRateCard", "attachRateCardToPricingPlan"},
			{"create-licensed-fee-chapter", "createLicensedItem", "createLicenseFee", "attachLicenseFeeToPricingPlan"},
			{"subscribe-customer-chapter", "setLiveVersion", "createCheckoutSession", "completeCheckout", "waitForServicingActivated"},
		},
		"flat-subscription-with-entitlements": {
			{"create-products-chapter", "create-basic-product", "create-basic-feature", "attach-feature-to-product"},
			{"setup-chapter", "create-test-clock", "create-customer"},
			{"subscribe-chapter", "create-checkout-session", "complete-checkout", "track-subscription-creation", "check-entitlements"},
			{"next-billing-cycle-chapter", "advance-time", "wait-for-invoice-created", "view-invoice"},
			{"cleanup-chapter", "test-clock-advanced", "delete-test-clock"},
		},
		"one-time-payment": {
			{"setup-chapter", "create-product"},
			{"checkout-chapter", "create-checkout-session", "complete-checkout"},
			{"webhook-chapter", "handle-checkout-completed"},
		},
		"setup-future-payments": {
			{"create-new-customer-chapter", "create-new-customer"},
			{"create-checkout-session-chapter", "create-checkout-session", "complete-checkout", "wait-for-checkout-completed"},
			{"charge-payment-method-later-chapter", "retrieve-setup-intent", "charge-payment-method-later", "wait-for-payment-intent-succeeded"},
		},
	}

	for id, expectedSteps := range expected {
		t.Run(id, func(t *testing.T) {
			bp, err := LoadBlueprint(id)
			require.NoError(t, err)
			require.Len(t, bp.Steps, len(expectedSteps))

			for i, expectedStep := range expectedSteps {
				require.NotEmpty(t, expectedStep)
				assert.Equal(t, expectedStep[0], bp.Steps[i].Key)
				require.Len(t, bp.Steps[i].Nodes, len(expectedStep)-1)
				for j, expectedNode := range expectedStep[1:] {
					assert.Equal(t, expectedNode, bp.Steps[i].Nodes[j].Key)
				}
			}
		})
	}
}

func TestAllEmbeddedBlueprintsAreStructurallyValid(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)
	require.NotEmpty(t, ids)

	allowedTypes := map[NodeType]bool{
		NodeAPIRequest:    true,
		NodeAsyncHandler:  true,
		NodeUIComponent:   true,
		NodeTestHelper:    true,
		NodeCLICommand:    true,
		NodeDashboard:     true,
		NodeSetUpWebhooks: true,
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			bp, err := LoadBlueprint(id)
			require.NoError(t, err)
			assert.Equal(t, id, bp.ID)
			assert.NotEmpty(t, bp.Title)
			require.NotEmpty(t, bp.Steps)

			stepKeys := make(map[string]bool)
			for _, ch := range bp.Steps {
				assert.NotEmpty(t, ch.Key)
				assert.False(t, stepKeys[ch.Key], "duplicate step key: %s", ch.Key)
				stepKeys[ch.Key] = true
				assert.NotEmpty(t, ch.Title)
				require.NotEmpty(t, ch.Nodes)

				nodeKeys := make(map[string]bool)
				for _, n := range ch.Nodes {
					assert.NotEmpty(t, n.Key)
					assert.False(t, nodeKeys[n.Key], "duplicate node key: %s", n.Key)
					nodeKeys[n.Key] = true
					assert.NotEmpty(t, n.Title)
					assert.True(t, allowedTypes[n.Type], "unsupported node type: %s", n.Type)

					if n.Type == NodeAPIRequest {
						require.NotNil(t, n.Request, "apiRequest node %q should have request field", n.Key)
						assert.NotEmpty(t, n.Request.Path)
						assert.NotEmpty(t, n.Request.Method)
					}
					if n.Type == NodeTestHelper && len(n.TestRequests) > 0 {
						for _, req := range n.TestRequests {
							assert.NotEmpty(t, req.Key, "testHelper node %q request should have key", n.Key)
							assert.NotEmpty(t, req.Path, "testHelper node %q request %q should have path", n.Key, req.Key)
							assert.NotEmpty(t, req.Method, "testHelper node %q request %q should have method", n.Key, req.Key)
						}
					}
				}
			}

			session := NewSessionFromBlueprint(bp, "test_"+id, map[string]string{"language": "node"}, nil)
			assert.Equal(t, id, session.Blueprint)
			require.NotEmpty(t, session.Steps)
			require.NotEmpty(t, session.Steps[0].Nodes)
			assert.Equal(t, "Understand the project", session.Steps[0].Nodes[0].Title)
			assert.Equal(t, len(bp.Steps)+1, len(session.Steps))
		})
	}
}
