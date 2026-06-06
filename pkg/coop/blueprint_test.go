package coop

import (
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
	assert.Len(t, bp.Chapters, 4)
	assert.Equal(t, "setup-chapter", bp.Chapters[0].Key)
	assert.Equal(t, NodeAPIRequest, bp.Chapters[0].Nodes[0].Type)
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

			for _, ch := range bp.Chapters {
				assertQualityText(t, "chapter title "+ch.Key, ch.Title, 4, weakPhrases)
				for _, n := range ch.Nodes {
					assertQualityText(t, "node title "+n.Key, n.Title, 4, weakPhrases)
					assert.NotEqual(t, "api request", strings.ToLower(strings.TrimSpace(n.Title)), "node %q should have a product-specific title", n.Key)
					if n.Description != "" {
						assertQualityText(t, "node description "+n.Key, n.Description, 20, weakPhrases)
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
	observableTerms := []string{"verify", "confirm", "report", "check", "run", "summarize", "ask"}
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

func TestLoadBlueprintPrefixMatch(t *testing.T) {
	bp, err := LoadBlueprint("deploy")
	require.NoError(t, err)
	assert.Equal(t, "deploy-stripe-projects", bp.ID)
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

	session := NewSessionFromBlueprint(bp, "coop_test123", map[string]string{"language": "node"})

	assert.Equal(t, "coop_test123", session.ID)
	assert.Equal(t, "one-time-payment", session.Blueprint)
	assert.Equal(t, SessionActive, session.Status)
	assert.Equal(t, "node", session.Settings["language"])
	// 4 blueprint chapters + 1 prepended context chapter
	assert.Len(t, session.Chapters, 5)

	// First chapter is always the context-gathering step
	assert.Equal(t, "context-chapter", session.Chapters[0].Key)
	assert.Equal(t, "Understand the project", session.Chapters[0].Nodes[0].Title)

	// All nodes should be pending
	for _, ch := range session.Chapters {
		for _, n := range ch.Nodes {
			assert.Equal(t, StepPending, n.State)
		}
	}

	// Total steps = blueprint steps (6) + context step (1)
	assert.Equal(t, 7, session.TotalSteps())
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

func TestLoadBlueprintChapterStructure(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	// Verify chapter keys are unique
	keys := make(map[string]bool)
	for _, ch := range bp.Chapters {
		assert.False(t, keys[ch.Key], "duplicate chapter key: %s", ch.Key)
		keys[ch.Key] = true
		assert.NotEmpty(t, ch.Title)
		assert.NotEmpty(t, ch.Nodes)

		// Verify node keys are unique within chapter
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
	for _, ch := range bp.Chapters {
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

	for _, ch := range bp.Chapters {
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

	for _, ch := range bp.Chapters {
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

	session := NewSessionFromBlueprint(bp, "test_123", nil)

	// First blueprint node (after context chapter) is apiRequest — should preserve the request
	firstBlueprintNode := session.Chapters[1].Nodes[0]
	assert.Equal(t, NodeAPIRequest, firstBlueprintNode.Type)
	assert.NotNil(t, firstBlueprintNode.Request)
	assert.Equal(t, "/v1/products", firstBlueprintNode.Request.Path)
	assert.Equal(t, "post", firstBlueprintNode.Request.Method)
}

func TestNewSessionFromBlueprintPreservesEvents(t *testing.T) {
	bp, err := LoadBlueprint("one-time-payment")
	require.NoError(t, err)

	session := NewSessionFromBlueprint(bp, "test_123", nil)

	// Find the asyncHandler node
	for _, ch := range session.Chapters {
		for _, n := range ch.Nodes {
			if n.Type == NodeAsyncHandler {
				assert.Contains(t, n.Events, "checkout.session.completed")
				return
			}
		}
	}
	t.Fatal("expected to find asyncHandler node")
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
			require.NotEmpty(t, bp.Chapters)

			chapterKeys := make(map[string]bool)
			for _, ch := range bp.Chapters {
				assert.NotEmpty(t, ch.Key)
				assert.False(t, chapterKeys[ch.Key], "duplicate chapter key: %s", ch.Key)
				chapterKeys[ch.Key] = true
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
				}
			}

			session := NewSessionFromBlueprint(bp, "test_"+id, map[string]string{"language": "node"})
			assert.Equal(t, id, session.Blueprint)
			require.NotEmpty(t, session.Chapters)
			require.NotEmpty(t, session.Chapters[0].Nodes)
			assert.Equal(t, "Understand the project", session.Chapters[0].Nodes[0].Title)
			assert.Equal(t, len(bp.Chapters)+1, len(session.Chapters))
		})
	}
}
