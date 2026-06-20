package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestStartWorkTransitionsNodeAndReturnsTypedNextCommand(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		return "", nil
	}))

	resp, err := service.StartWork(session.ID, 1, "Scanning")
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "active", resp.State)
	assert.Contains(t, resp.Next, "stripe coop agent report-work")
	require.NotNil(t, resp.BlueprintStep)
	assert.Equal(t, "one", resp.BlueprintStep.Key)
	assert.Contains(t, resp.AgentGuidance, "Follow blueprint step 1")

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Equal(t, "Scanning", node.Activity)
}

func TestStartWorkReturnsWebhookExampleForAsyncHandler(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		session.Settings = map[string]string{"language": "node"}
		session.Steps[0].Nodes[0].Type = coop.NodeAsyncHandler
		session.Steps[0].Nodes[0].Events = []string{"invoice.paid", "customer.subscription.created"}
		return nil
	})
	require.NoError(t, err)
	service := NewService(store)

	resp, err := service.StartWork(session.ID, 1, "Implementing webhook")
	require.NoError(t, err)

	require.True(t, resp.OK)
	assert.Empty(t, resp.SDKExample)
	assert.Contains(t, resp.WebhookExample, `case "invoice.paid"`)
	assert.Contains(t, resp.WebhookExample, `case "customer.subscription.created"`)
	assert.Contains(t, resp.WebhookExample, "stripe.v2.core.events.retrieve")
	assert.Contains(t, resp.WebhookExample, "v1.<event>")
	assert.Contains(t, resp.AgentGuidance, "signed webhook/event handler")
	require.NotNil(t, resp.BlueprintStep)
	assert.Equal(t, []string{"invoice.paid", "customer.subscription.created"}, resp.BlueprintStep.Events)
}

func TestStartWorkReturnsSDKExampleForBlueprintParams(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		session.Settings = map[string]string{"language": "node"}
		session.Steps[0].Nodes[0].Type = coop.NodeAPIRequest
		session.Steps[0].Nodes[0].Request = &coop.APIRequest{
			Path:   "/v1/checkout/sessions",
			Method: "post",
			Params: map[string]interface{}{
				"mode": "payment",
			},
		}
		return nil
	})
	require.NoError(t, err)
	called := false
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		called = true
		assert.Equal(t, "/v1/checkout/sessions", path)
		assert.Equal(t, "post", method)
		return "await stripe.checkout.sessions.create({ mode: 'payment' })", nil
	}))

	resp, err := service.StartWork(session.ID, 1, "Create checkout")
	require.NoError(t, err)

	require.True(t, called)
	assert.Contains(t, resp.SDKExample, "mode: 'payment'")
	assert.Contains(t, resp.AgentGuidance, "blueprint_step.api_request.params are canonical")
	require.NotNil(t, resp.BlueprintStep)
	require.NotNil(t, resp.BlueprintStep.APIRequest)
	assert.Equal(t, "/v1/checkout/sessions", resp.BlueprintStep.APIRequest.Path)
}

func TestStartWorkReturnsStructuredSemantics(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		session.Steps[0].Nodes[0].Type = coop.NodeAPIRequest
		session.Steps[0].Nodes[0].Request = &coop.APIRequest{
			Path:   "/v1/checkout/sessions",
			Method: "post",
		}
		session.Steps[0].Nodes[0].Semantics = &coop.BlueprintSemantics{
			SourceOfTruth: &coop.SourceOfTruthSemantics{Amount: "app_domain"},
			PaymentLifecycle: &coop.PaymentLifecycleSemantics{
				CompletionEvent:                  "checkout.session.completed",
				FulfillmentRequiresSignedWebhook: true,
			},
		}
		return nil
	})
	require.NoError(t, err)
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		return "", nil
	}))

	resp, err := service.StartWork(session.ID, 1, "Create checkout")
	require.NoError(t, err)

	require.NotNil(t, resp.BlueprintStep)
	require.NotNil(t, resp.BlueprintStep.Semantics)
	assert.Equal(t, "app_domain", resp.BlueprintStep.Semantics.SourceOfTruth.Amount)
	assert.Contains(t, resp.AgentGuidance, "Blueprint source-of-truth semantics are canonical")
	assert.Contains(t, resp.AgentGuidance, "completion_event=checkout.session.completed")
}

func TestStartWorkReturnsBlueprintAppRoles(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		session.Steps[0].Nodes[0].AppRoles = []coop.AppRole{
			{
				ID:          "payable_record",
				Kind:        "domain_record",
				Required:    true,
				Description: "The local record being paid for",
			},
		}
		return nil
	})
	require.NoError(t, err)
	service := NewService(store)

	resp, err := service.StartWork(session.ID, 1, "Bind roles")
	require.NoError(t, err)

	require.NotNil(t, resp.BlueprintStep)
	require.Len(t, resp.BlueprintStep.AppRoles, 1)
	assert.Equal(t, "payable_record", resp.BlueprintStep.AppRoles[0].ID)
	assert.Equal(t, "domain_record", resp.BlueprintStep.AppRoles[0].Kind)
	assert.True(t, resp.BlueprintStep.AppRoles[0].Required)
}

func TestStartWorkAvoidsEmptySDKExampleForEndpointOnlyMutatingRequest(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		session.Settings = map[string]string{"language": "node"}
		session.Steps[0].Nodes[0].Type = coop.NodeAPIRequest
		session.Steps[0].Nodes[0].Request = &coop.APIRequest{
			Path:   "/v1/checkout/sessions",
			Method: "post",
		}
		return nil
	})
	require.NoError(t, err)
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		t.Fatal("endpoint-only mutating requests should not fetch docs snippets")
		return "", nil
	}))

	resp, err := service.StartWork(session.ID, 1, "Create checkout")
	require.NoError(t, err)

	assert.Contains(t, resp.SDKExample, "does not include canonical request params")
	assert.Contains(t, resp.AgentGuidance, "endpoint and method only")
}

func TestStartWorkReturnsGuidanceForUIComponent(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		session.Steps[0].Nodes[0].Type = coop.NodeUIComponent
		session.Steps[0].Nodes[0].Description = "Add a Checkout button that calls the app checkout endpoint."
		session.Steps[0].Nodes[0].ReviewPrompt = "Open the app and confirm the button redirects to Checkout."
		return nil
	})
	require.NoError(t, err)
	service := NewService(store)

	resp, err := service.StartWork(session.ID, 1, "Building UI")
	require.NoError(t, err)

	require.NotNil(t, resp.BlueprintStep)
	assert.Equal(t, coop.NodeUIComponent, resp.BlueprintStep.Type)
	assert.Contains(t, resp.AgentGuidance, "Add a Checkout button")
	assert.Contains(t, resp.AgentGuidance, "Open the app and confirm")
	assert.Contains(t, resp.AgentGuidance, "user-facing app behavior")
}

func TestStartWorkIsIdempotentForActiveNode(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First note")
	require.NoError(t, err)
	resp, err := service.StartWork(session.ID, 1, "Updated note")
	require.NoError(t, err)

	require.True(t, resp.OK)
	assert.Equal(t, "active", resp.State)
	assert.Contains(t, resp.Next, "stripe coop agent report-work")

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Equal(t, "Updated note", node.Activity)
}

func TestReportWorkContinuesStepBeforeReview(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Done"}, false)
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "review", resp.State)
	assert.Contains(t, resp.Message, "Continue the step")
	assert.Contains(t, resp.Next, "--step=2")
}

func TestReportWorkRoutesToAwaitReviewWhenStepReady(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)
	_, err = service.StartWork(session.ID, 2, "Second")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 2, ReportWorkInput{File: "client.go"}, false)
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Contains(t, resp.Message, "Step ready for review")
	assert.Contains(t, resp.Next, "stripe coop agent await-review")
}

func TestConfirmAndRequestChangesUseCentralWorkflow(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)

	updated, err := service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)
	node, err := updated.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeDone, node.State)
}

func TestConfirmReviewTreatsSkippedNodesAsTerminal(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.Skip(session.ID, 1, "Not needed")
	require.NoError(t, err)

	updated, err := service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)
	node, err := updated.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeSkipped, node.State)
}

func TestRequestChangesMovesReviewNodeBackToActive(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)
	updated, err := service.RequestChanges(session.ID, []int{1}, "Needs tests")
	require.NoError(t, err)
	node, err := updated.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Equal(t, "Needs tests", node.RejectionNote)
	assert.Nil(t, node.Implementation)
}

func TestAgentWorkflowRejectsInactiveSessions(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Service, string) (coop.CommandResponse, error)
	}{
		{
			name: "start work",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.StartWork(sessionID, 1, "Starting")
			},
		},
		{
			name: "report work",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.ReportWork(sessionID, 1, ReportWorkInput{File: "server.go"}, false)
			},
		},
		{
			name: "report check",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.ReportCheck(sessionID, 1, "Manual checkout passed", true)
			},
		},
		{
			name: "skip",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.Skip(sessionID, 1, "Not needed")
			},
		},
		{
			name: "await review",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.AwaitReview(sessionID, 1)
			},
		},
	}

	for _, status := range []coop.SessionStatus{coop.SessionCompleted, coop.SessionAborted} {
		t.Run(string(status), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					store, session := workflowTestStore(t)
					service := NewService(store)
					_, err := store.Update(session.ID, func(session *coop.Session) error {
						session.Status = status
						return nil
					})
					require.NoError(t, err)

					resp, err := tt.run(service, session.ID)
					require.NoError(t, err)
					assert.False(t, resp.OK)
					assert.Contains(t, resp.Error, "session workflow_test is "+string(status)+" and cannot be advanced")

					loaded, err := store.Read(session.ID)
					require.NoError(t, err)
					node, err := loaded.NodeByNumber(1)
					require.NoError(t, err)
					assert.Equal(t, coop.NodePending, node.State)
				})
			}
		})
	}
}

func TestReviewWorkflowRejectsInactiveSessions(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Service, string) error
	}{
		{
			name: "confirm review",
			run: func(service *Service, sessionID string) error {
				_, err := service.ConfirmReview(sessionID, []int{1})
				return err
			},
		},
		{
			name: "request changes",
			run: func(service *Service, sessionID string) error {
				_, err := service.RequestChanges(sessionID, []int{1}, "Needs tests")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, session := workflowTestStore(t)
			service := NewService(store)
			_, err := store.Update(session.ID, func(session *coop.Session) error {
				session.Status = coop.SessionAborted
				return nil
			})
			require.NoError(t, err)

			err = tt.run(service, session.ID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "session workflow_test is aborted and cannot be advanced")
		})
	}
}

func TestCompletedParentedSessionRoutesNextActionToParent(t *testing.T) {
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Blueprint:     "one-time-payment",
		Status:        coop.SessionCompleted,
	}
	require.NoError(t, store.Write(parent))
	child := &coop.Session{
		SchemaVersion:   coop.CurrentSessionSchemaVersion,
		ID:              "child_session",
		Blueprint:       "follow-up-integration",
		Status:          coop.SessionActive,
		ParentSessionID: "parent_session",
		ParentStepID:    "add-integration",
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "add-integration", Title: "Add integration"},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "add-integration", Title: "Add integration"},
						State:          coop.NodeActive,
					},
				},
			},
		},
	}
	require.NoError(t, store.Write(child))
	service := NewService(store)

	resp, err := service.ReportWork(child.ID, 1, ReportWorkInput{File: "server.go", Note: "Added another integration"}, true)

	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "stripe coop agent next-action --session=parent_session --completed=add-integration", resp.Next)
}

func workflowTestStore(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "workflow_test",
		Blueprint:     "test",
		Status:        coop.SessionActive,
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{
					Key:   "step",
					Title: "Step",
				},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "one", Title: "One"},
						State:          coop.NodePending,
					},
					{
						NodeDefinition: coop.NodeDefinition{Key: "two", Title: "Two"},
						State:          coop.NodePending,
					},
				},
			},
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}
