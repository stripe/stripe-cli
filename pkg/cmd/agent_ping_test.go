package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

type agentPingTelemetryEvent struct {
	name  string
	value string
}

type agentPingTelemetryClient struct {
	events chan agentPingTelemetryEvent
}

func newAgentPingTelemetryClient() *agentPingTelemetryClient {
	return &agentPingTelemetryClient{events: make(chan agentPingTelemetryEvent, 2)}
}

func (c *agentPingTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *agentPingTelemetryClient) SendEvent(_ context.Context, name, value string) {
	c.events <- agentPingTelemetryEvent{name: name, value: value}
}

func TestAgentPingEmitsSkillUsage(t *testing.T) {
	client := newAgentPingTelemetryClient()
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	output, err := executeAgentPing(ctx, "--skill", "stripe-best-practices")

	require.NoError(t, err)
	require.Empty(t, output)
	event := waitForAgentPingEvent(t, client)
	require.Equal(t, agentPingEventName, event.name)

	var payload agentPingEvent
	require.NoError(t, json.Unmarshal([]byte(event.value), &payload))
	require.Equal(t, agentPingEvent{Type: "skill", Name: "stripe-best-practices"}, payload)
	assertNoAgentPingEvent(t, client)
}

func TestAgentPingDoesNotValidateSkillName(t *testing.T) {
	client := newAgentPingTelemetryClient()
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	_, err := executeAgentPing(ctx, "--skill", "arbitrary skill name")

	require.NoError(t, err)
	event := waitForAgentPingEvent(t, client)

	var payload agentPingEvent
	require.NoError(t, json.Unmarshal([]byte(event.value), &payload))
	require.Equal(t, "arbitrary skill name", payload.Name)
	assertNoAgentPingEvent(t, client)
}

func TestAgentPingMissingOrEmptySkillIsSilentNoOp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing"},
		{name: "empty", args: []string{"--skill="}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newAgentPingTelemetryClient()
			ctx := stripe.WithTelemetryClient(context.Background(), client)

			output, err := executeAgentPing(ctx, tt.args...)

			require.NoError(t, err)
			require.Empty(t, output)
			assertNoAgentPingEvent(t, client)
		})
	}
}

func TestAgentPingWithoutTelemetryClientIsSilentSuccess(t *testing.T) {
	output, err := executeAgentPing(context.Background(), "--skill", "stripe-best-practices")

	require.NoError(t, err)
	require.Empty(t, output)
}

func TestAgentPingRejectsPositionalArguments(t *testing.T) {
	_, err := executeAgentPing(context.Background(), "unexpected")

	require.Error(t, err)
}

func TestAgentCommandRegistersPing(t *testing.T) {
	agent := newAgentCmd()

	ping, _, err := agent.cmd.Find([]string{"ping"})

	require.NoError(t, err)
	require.Equal(t, "ping", ping.Name())
}

func executeAgentPing(ctx context.Context, args ...string) (string, error) {
	ping := newAgentPingCmd()
	output := new(bytes.Buffer)
	ping.cmd.SetOut(output)
	ping.cmd.SetErr(output)
	ping.cmd.SetArgs(args)

	err := ping.cmd.ExecuteContext(ctx)
	return output.String(), err
}

func waitForAgentPingEvent(t *testing.T, client *agentPingTelemetryClient) agentPingTelemetryEvent {
	t.Helper()

	select {
	case event := <-client.events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for agent ping telemetry")
		return agentPingTelemetryEvent{}
	}
}

func assertNoAgentPingEvent(t *testing.T, client *agentPingTelemetryClient) {
	t.Helper()

	select {
	case event := <-client.events:
		t.Fatalf("unexpected agent ping telemetry: %#v", event)
	case <-time.After(25 * time.Millisecond):
	}
}
