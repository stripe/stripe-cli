package workflow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
)

func workflowNode(key, title string, state coop.NodeState) coop.SessionNode {
	return coop.SessionNode{
		BlueprintNode: coop.BlueprintNode{
			Key:   key,
			Title: coop.MessageDescriptor{DefaultMessage: title},
		},
		State: state,
	}
}

func workflowStep(key, title string, nodes ...coop.SessionNode) coop.SessionStep {
	return coop.SessionStep{
		BlueprintStepDefinition: coop.BlueprintStepDefinition{
			Key:   key,
			Title: coop.MessageDescriptor{DefaultMessage: title},
		},
		Nodes: nodes,
	}
}

func TestStartWorkTransitionsNodeAndReturnsTypedNextCommand(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		return "", nil
	}))

	resp, err := service.StartWork(session.ID, 1, "Scanning")
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "active", resp.State)
	assert.Empty(t, resp.Next)
	assert.Contains(t, resp.NextTemplate, "stripe coop agent report-work")
	require.NotEmpty(t, resp.RequiredInputs)

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Equal(t, "Scanning", node.Activity)
}

func TestStartWorkReturnsOnlyCurrentNodeContext(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		current := &session.Steps[0].Nodes[0]
		current.NodeType = coop.NodeTestHelper
		current.Description = coop.MessageDescriptor{DefaultMessage: "Advance the test clock and confirm an invoice is created."}
		current.ReviewPrompt = "Confirm the new invoice appears."
		current.ReviewCommand = "stripe invoices list --limit=1"
		current.TestHelperDetails = &coop.BlueprintTestHelperDetails{
			Requests: []coop.BlueprintRequestFixture{{
				Key:    "advance-clock",
				Path:   "/v1/test_helpers/test_clocks/clock_123/advance",
				Method: "post",
			}},
		}
		current.AsyncHandlerDetails = &coop.BlueprintAsyncHandlerDetails{
			Events: []coop.AsyncEvent{{EventType: "invoice.created"}},
		}
		session.Steps[0].Nodes[1].Description = coop.MessageDescriptor{DefaultMessage: "FUTURE NODE DETAILS MUST NOT LEAK"}
		return nil
	})
	require.NoError(t, err)

	resp, err := NewService(store).StartWork(session.ID, 1, "Starting")
	require.NoError(t, err)
	require.True(t, resp.OK)

	assert.Contains(t, resp.AgentPrompt, "Current node 1 of 2")
	assert.Contains(t, resp.AgentPrompt, "Advance the test clock")
	assert.Contains(t, resp.AgentPrompt, "Confirm the new invoice appears")
	assert.Contains(t, resp.AgentPrompt, `stripe invoices list --limit=1`)
	assert.Contains(t, resp.AgentPrompt, `stripe coop agent report-check --session=workflow_test --step=1 --check="<what you verified>" --passed`)
	assert.Contains(t, resp.AgentPrompt, `stripe coop agent skip --session=workflow_test --step=1 --note="<reason>"`)
	assert.NotContains(t, resp.AgentPrompt, "FUTURE NODE DETAILS MUST NOT LEAK")
	require.Len(t, resp.TestRequests, 1)
	assert.Equal(t, "advance-clock", resp.TestRequests[0].Key)
	require.Len(t, resp.Events, 1)
	assert.Equal(t, "invoice.created", resp.Events[0].EventType)
}

func TestReportWorkPersistsOutputsAndStartWorkResolvesLaterRequest(t *testing.T) {
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "output_workflow",
		Status:        coop.SessionActive,
		Settings:      map[string]string{"language": "node"},
		Steps: []coop.SessionStep{{
			BlueprintStepDefinition: coop.BlueprintStepDefinition{Key: "setup", Title: coop.MessageDescriptor{DefaultMessage: "Setup"}},
			Nodes: []coop.SessionNode{
				{
					BlueprintNode: coop.BlueprintNode{
						Key:      "create-product",
						Title:    coop.MessageDescriptor{DefaultMessage: "Create product"},
						NodeType: coop.NodeAPIRequest,
						APIRequestDetails: &coop.BlueprintAPIRequestDetails{Fixture: coop.BlueprintRequestFixture{
							Path:   "/v1/products",
							Method: "post",
						}},
					},
					State: coop.NodePending,
				},
				{
					BlueprintNode: coop.BlueprintNode{
						Key:      "use-product",
						Title:    coop.MessageDescriptor{DefaultMessage: "Use product"},
						NodeType: coop.NodeAPIRequest,
						APIRequestDetails: &coop.BlueprintAPIRequestDetails{Fixture: coop.BlueprintRequestFixture{
							Path:   "/v1/products/${node.setup.create-product:id}",
							Method: "get",
							Params: map[string]any{
								"version": "${node.setup.create-product:latest_version}",
							},
						}},
					},
					State: coop.NodePending,
				},
			},
		}},
	}
	require.NoError(t, store.Write(session))
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		return "", nil
	}))

	start, err := service.StartWork(session.ID, 1, "Creating product")
	require.NoError(t, err)
	require.True(t, start.OK)
	assert.Equal(t, []coop.RequiredOutput{
		{Field: "id"},
		{Field: "latest_version"},
	}, start.RequiredOutputs)
	assert.Empty(t, start.Next)
	assert.Contains(t, start.NextTemplate, `--output="id=<id>"`)

	missingEverything, err := service.ReportWork(session.ID, 1, ReportWorkInput{}, false)
	require.NoError(t, err)
	assert.False(t, missingEverything.OK)
	assert.Contains(t, missingEverything.Error, "--note flag is required")
	require.NotNil(t, missingEverything.Recovery)
	assert.Contains(t, missingEverything.Recovery.NextTemplate, `--note="<what you did>"`)
	assert.Contains(t, missingEverything.Recovery.NextTemplate, `--output="id=<id>"`)
	assert.Contains(t, missingEverything.Recovery.NextTemplate, `--output="latest_version=<latest_version>"`)
	require.Len(t, missingEverything.Recovery.RequiredInputs, 3)

	missing, err := service.ReportWork(session.ID, 1, ReportWorkInput{Note: "Created product"}, false)
	require.NoError(t, err)
	assert.False(t, missing.OK)
	assert.Contains(t, missing.Error, "missing required --output values")
	require.NotNil(t, missing.Recovery)
	assert.Contains(t, missing.Recovery.NextTemplate, "--output")

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	first, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, first.State)
	assert.Empty(t, first.Outputs)

	reported, err := service.ReportWork(session.ID, 1, ReportWorkInput{
		Note: "Created product",
		Outputs: coop.NodeOutputs{
			coop.DefaultOutputSource: {
				"id":             json.RawMessage(`"prod_123"`),
				"latest_version": json.RawMessage(`7`),
			},
		},
	}, false)
	require.NoError(t, err)
	require.True(t, reported.OK)

	next, err := service.StartWork(session.ID, 2, "Using product")
	require.NoError(t, err)
	require.True(t, next.OK)
	require.NotNil(t, next.APIRequest)
	assert.Equal(t, "/v1/products/prod_123", next.APIRequest.Path)
	assert.Equal(t, float64(7), next.APIRequest.Params["version"])
}

func TestSkipCascadesToTransitiveOutputDependents(t *testing.T) {
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "skip_dependencies",
		Status:        coop.SessionActive,
		Steps: []coop.SessionStep{{
			BlueprintStepDefinition: coop.BlueprintStepDefinition{Key: "setup", Title: coop.MessageDescriptor{DefaultMessage: "Setup"}},
			Nodes: []coop.SessionNode{
				{BlueprintNode: coop.BlueprintNode{Key: "source", Title: coop.MessageDescriptor{DefaultMessage: "Source"}}, State: coop.NodePending},
				{
					BlueprintNode: coop.BlueprintNode{
						Key: "direct", Title: coop.MessageDescriptor{DefaultMessage: "Direct dependent"},
						APIRequestDetails: &coop.BlueprintAPIRequestDetails{Fixture: coop.BlueprintRequestFixture{Path: "/v1/direct/${node.setup.source:id}", Method: "get"}},
					},
					State: coop.NodePending,
				},
				{
					BlueprintNode: coop.BlueprintNode{
						Key: "transitive", Title: coop.MessageDescriptor{DefaultMessage: "Transitive dependent"},
						APIRequestDetails: &coop.BlueprintAPIRequestDetails{Fixture: coop.BlueprintRequestFixture{Path: "/v1/transitive/${node.setup.direct:id}", Method: "get"}},
					},
					State: coop.NodePending,
				},
				{BlueprintNode: coop.BlueprintNode{Key: "independent", Title: coop.MessageDescriptor{DefaultMessage: "Independent"}}, State: coop.NodePending},
			},
		}},
	}
	require.NoError(t, store.Write(session))

	resp, err := NewService(store).Skip(session.ID, 1, "Does not apply")
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Contains(t, resp.Message, "dependent nodes: 2, 3")
	assert.Contains(t, resp.Next, "--step=4")

	updated, err := store.Read(session.ID)
	require.NoError(t, err)
	for _, nodeNumber := range []int{1, 2, 3} {
		node, err := updated.NodeByNumber(nodeNumber)
		require.NoError(t, err)
		assert.Equal(t, coop.NodeSkipped, node.State)
	}
	direct, err := updated.NodeByNumber(2)
	require.NoError(t, err)
	assert.Contains(t, direct.Activity, "depends on skipped node 1")
	independent, err := updated.NodeByNumber(4)
	require.NoError(t, err)
	assert.Equal(t, coop.NodePending, independent.State)
}

func TestStartWorkAPIRequestGuidanceDefersPlacementToProjectContext(t *testing.T) {
	store, session := workflowTestStore(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		current := &session.Steps[0].Nodes[0]
		current.NodeType = coop.NodeAPIRequest
		current.Description = coop.MessageDescriptor{DefaultMessage: "Create a product during integration setup."}
		current.APIRequestDetails = &coop.BlueprintAPIRequestDetails{
			Fixture: coop.BlueprintRequestFixture{Path: "/v1/products", Method: "post"},
		}
		return nil
	})
	require.NoError(t, err)

	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		return "stripe.products.create(...)", nil
	}))
	resp, err := service.StartWork(session.ID, 1, "Starting")
	require.NoError(t, err)
	require.True(t, resp.OK)

	require.NotNil(t, resp.APIRequest)
	assert.Equal(t, "/v1/products", resp.APIRequest.Path)
	assert.Equal(t, "stripe.products.create(...)", resp.SDKExample)
	assert.Contains(t, resp.AgentPrompt, "Based on the task and existing project")
	assert.Contains(t, resp.AgentPrompt, "runtime application code")
	assert.Contains(t, resp.AgentPrompt, "setup or seed script")
	assert.Contains(t, resp.AgentPrompt, "one-time provisioning")
	assert.NotContains(t, resp.AgentPrompt, "Implement the Stripe call in api_request in the application")
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
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Implemented server work"}, false)
	require.NoError(t, err)
	_, err = service.StartWork(session.ID, 2, "Second")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 2, ReportWorkInput{File: "client.go", Note: "Implemented client work"}, false)
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Contains(t, resp.Message, "Step ready for review")
	assert.Contains(t, resp.Next, "stripe coop agent await-review")
	assert.Equal(t, int(AwaitTimeout.Seconds()), resp.WaitTimeoutSeconds)
}

func TestAwaitTimeoutContractLeavesHarnessHeadroom(t *testing.T) {
	// The interval has to sit under an agent harness's *default* command
	// timeout, not just under the one Co-op advertises: Claude Code's Bash tool
	// defaults to 120s, so anything longer is killed whenever a model does not
	// set a timeout itself.
	assert.Less(t, AwaitTimeout, 120*time.Second)
	assert.Greater(t, AwaitHarnessTimeout, AwaitTimeout)

	resp := waitingResponse("session_123", 4, AwaitTimeout, 90*time.Second, "")
	require.NoError(t, resp.Validate())
	assert.Equal(t, int(AwaitTimeout.Seconds()), resp.WaitTimeoutSeconds)
}

func TestConfiguredAwaitTimeoutIsAdvertised(t *testing.T) {
	store, session := workflowTestStore(t)
	timeout := 30 * time.Second
	service := NewService(store, WithAwaitTimeout(timeout))

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{Note: "First done"}, false)
	require.NoError(t, err)
	_, err = service.StartWork(session.ID, 2, "Second")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 2, ReportWorkInput{Note: "Second done"}, false)
	require.NoError(t, err)

	require.True(t, resp.OK)
	assert.Contains(t, resp.Next, "stripe coop agent await-review")
	assert.Equal(t, int(timeout.Seconds()), resp.WaitTimeoutSeconds)

	waiting := waitingResponse(session.ID, 2, timeout, time.Minute, "")
	assert.Equal(t, int(timeout.Seconds()), waiting.WaitTimeoutSeconds)
}

func TestReportWorkRequiresNoteWithTemplateRecovery(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 1, ReportWorkInput{}, false)
	require.NoError(t, err)

	assert.False(t, resp.OK)
	assert.Empty(t, resp.Next)
	require.NotNil(t, resp.Recovery)
	assert.Contains(t, resp.Recovery.NextTemplate, `--note="<what you did>"`)
	require.Len(t, resp.Recovery.RequiredInputs, 1)
	require.NoError(t, resp.Validate())
}

func TestConfirmAndRequestChangesUseCentralWorkflow(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Implemented server work"}, false)
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
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Implemented server work"}, false)
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
				return service.ReportWork(sessionID, 1, ReportWorkInput{File: "server.go", Note: "Implemented server work"}, false)
			},
		},
		{
			name: "report check",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.ReportCheck(sessionID, 1, "Manual checkout passed", "", true)
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
			workflowStep("add-integration", "Add integration",
				workflowNode("add-integration", "Add integration", coop.NodeActive),
			),
		},
	}
	require.NoError(t, store.Write(child))
	service := NewService(store)

	resp, err := service.ReportWork(child.ID, 1, ReportWorkInput{File: "server.go", Note: "Added another integration"}, true)

	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "stripe coop agent next-action --session=parent_session --completed=add-integration", resp.Next)
	assert.Equal(t, int(helpers.NextActionInterval.Seconds()), resp.WaitTimeoutSeconds)
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
			workflowStep("step", "Step",
				workflowNode("one", "One", coop.NodePending),
				workflowNode("two", "Two", coop.NodePending),
			),
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}
