package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

type agentReportUsageTelemetryEvent struct {
	name  string
	value string
}

type agentReportUsageTelemetryClient struct {
	events chan agentReportUsageTelemetryEvent
}

func newAgentReportUsageTelemetryClient() *agentReportUsageTelemetryClient {
	return &agentReportUsageTelemetryClient{events: make(chan agentReportUsageTelemetryEvent, 2)}
}

func (c *agentReportUsageTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *agentReportUsageTelemetryClient) SendEvent(_ context.Context, name, value string) {
	c.events <- agentReportUsageTelemetryEvent{name: name, value: value}
}

func TestAgentReportUsageEmitsSkillUsage(t *testing.T) {
	client := newAgentReportUsageTelemetryClient()
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	stdout, stderr, err := executeAgentReportUsage(ctx, "--skill", "stripe-best-practices")

	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(agentReportUsageAcknowledgement, "stripe-best-practices"), stdout)
	require.Empty(t, stderr)
	event := waitForAgentReportUsageEvent(t, client)
	require.Equal(t, agentReportUsageEventName, event.name)

	var payload agentReportUsageEvent
	require.NoError(t, json.Unmarshal([]byte(event.value), &payload))
	require.Equal(t, agentReportUsageEvent{Type: "skill", Name: "stripe-best-practices"}, payload)
	assertNoAgentReportUsageEvent(t, client)
}

func TestAgentReportUsageDoesNotValidateSkillName(t *testing.T) {
	client := newAgentReportUsageTelemetryClient()
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	stdout, stderr, err := executeAgentReportUsage(ctx, "--skill", "arbitrary skill name")

	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(agentReportUsageAcknowledgement, "arbitrary skill name"), stdout)
	require.Empty(t, stderr)
	event := waitForAgentReportUsageEvent(t, client)

	var payload agentReportUsageEvent
	require.NoError(t, json.Unmarshal([]byte(event.value), &payload))
	require.Equal(t, "arbitrary skill name", payload.Name)
	assertNoAgentReportUsageEvent(t, client)
}

func TestAgentReportUsageMissingOrEmptySkillIsSilentNoOp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing"},
		{name: "empty", args: []string{"--skill="}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newAgentReportUsageTelemetryClient()
			ctx := stripe.WithTelemetryClient(context.Background(), client)

			stdout, stderr, err := executeAgentReportUsage(ctx, tt.args...)

			require.NoError(t, err)
			require.Empty(t, stdout)
			require.Empty(t, stderr)
			assertNoAgentReportUsageEvent(t, client)
		})
	}
}

func TestAgentReportUsageWithoutTelemetryClientStillAcknowledges(t *testing.T) {
	stdout, stderr, err := executeAgentReportUsage(context.Background(), "--skill", "stripe-best-practices")

	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(agentReportUsageAcknowledgement, "stripe-best-practices"), stdout)
	require.Empty(t, stderr)
}

func TestAgentReportUsageRejectsPositionalArguments(t *testing.T) {
	_, _, err := executeAgentReportUsage(context.Background(), "unexpected")

	require.Error(t, err)
}

func TestAgentCommandRegistersReportUsage(t *testing.T) {
	agent := newAgentCmd()

	reportUsage, _, err := agent.cmd.Find([]string{"report_usage"})

	require.NoError(t, err)
	require.Equal(t, "report_usage", reportUsage.Name())

	for _, child := range agent.cmd.Commands() {
		require.NotEqual(t, "ping", child.Name())
	}
}

func executeAgentReportUsage(ctx context.Context, args ...string) (string, string, error) {
	reportUsage := newAgentReportUsageCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	reportUsage.cmd.SetOut(stdout)
	reportUsage.cmd.SetErr(stderr)
	reportUsage.cmd.SetArgs(args)

	err := reportUsage.cmd.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

func waitForAgentReportUsageEvent(t *testing.T, client *agentReportUsageTelemetryClient) agentReportUsageTelemetryEvent {
	t.Helper()

	select {
	case event := <-client.events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for agent report usage telemetry")
		return agentReportUsageTelemetryEvent{}
	}
}

func assertNoAgentReportUsageEvent(t *testing.T, client *agentReportUsageTelemetryClient) {
	t.Helper()

	select {
	case event := <-client.events:
		t.Fatalf("unexpected agent report usage telemetry: %#v", event)
	case <-time.After(25 * time.Millisecond):
	}
}
