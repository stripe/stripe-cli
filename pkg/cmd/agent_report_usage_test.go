package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

type agentReportUsageTelemetryEvent struct {
	name  string
	value string
}

type agentReportUsageTelemetryClient struct {
	events []agentReportUsageTelemetryEvent
}

func (c *agentReportUsageTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *agentReportUsageTelemetryClient) SendEvent(_ context.Context, name, value string) {
	c.events = append(c.events, agentReportUsageTelemetryEvent{name: name, value: value})
}

func TestAgentReportUsageEmitsSkillUsedEventAndJSONResponse(t *testing.T) {
	client := &agentReportUsageTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	stdout, stderr, err := executeAgentReportUsage(ctx, "--skill", "arbitrary skill / name")

	require.NoError(t, err)
	require.Empty(t, stderr)
	require.Equal(t, "{\"message\":\"Thank you for reporting. If anything about the tool was surprisingly good or bad, Stripe welcomes feedback from agents via `stripe feedback --help`\"}\n", stdout)
	require.Equal(t, agentReportUsageResponse{Message: agentReportUsageMessage}, decodeAgentReportUsageResponse(t, stdout))
	require.Equal(t, []agentReportUsageTelemetryEvent{{
		name:  agentReportUsageEventName,
		value: `{"skillName":"arbitrary skill / name"}`,
	}}, client.events)
}

func TestAgentReportUsageRequiresSkill(t *testing.T) {
	_, _, err := executeAgentReportUsage(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "required flag(s) \"skill\" not set")
}

func TestAgentReportUsageRejectsEmptySkill(t *testing.T) {
	_, _, err := executeAgentReportUsage(context.Background(), "--skill=")

	require.EqualError(t, err, "--skill must not be empty")
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	require.Equal(t, errorcategory.UserInput, category)
}

func TestAgentReportUsageDryRunLogsWithoutEmitting(t *testing.T) {
	client := &agentReportUsageTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	stdout, stderr, err := executeAgentReportUsage(ctx, "--skill", "any value", "--dry-run")

	require.NoError(t, err)
	require.Equal(t, agentReportUsageResponse{Message: agentReportUsageMessage}, decodeAgentReportUsageResponse(t, stdout))
	require.Contains(t, stderr, `Dry run: would emit telemetry event "skill_used" with additional context {"skillName":"any value"}`)
	require.Empty(t, client.events)
}

func TestAgentReportUsageDebugLogsArgumentsAndCalls(t *testing.T) {
	client := &agentReportUsageTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	_, stderr, err := executeAgentReportUsage(ctx, "--skill", "debug value", "--debug")

	require.NoError(t, err)
	require.Contains(t, stderr, `Debug: report_usage arguments: skill="debug value" dry-run=false`)
	require.Contains(t, stderr, `Debug: emitting telemetry event "skill_used" with additional context {"skillName":"debug value"}`)
	require.Contains(t, stderr, "Debug: writing JSON response")
	require.Len(t, client.events, 1)
}

func TestAgentReportUsageWithoutTelemetryStillReturnsJSON(t *testing.T) {
	stdout, stderr, err := executeAgentReportUsage(context.Background(), "--skill", "anything", "--debug")

	require.NoError(t, err)
	require.Equal(t, agentReportUsageResponse{Message: agentReportUsageMessage}, decodeAgentReportUsageResponse(t, stdout))
	require.Contains(t, stderr, "Debug: telemetry client unavailable; event was not emitted")
}

func TestAgentReportUsageRejectsPositionalArguments(t *testing.T) {
	_, _, err := executeAgentReportUsage(context.Background(), "--skill", "anything", "unexpected")

	require.Error(t, err)
}

func TestAgentReportUsageIsHiddenFromHelpAndCommandMap(t *testing.T) {
	agent := newAgentCmd()
	reportUsage, _, err := agent.cmd.Find([]string{"report_usage"})
	require.NoError(t, err)
	require.True(t, reportUsage.Hidden)

	help, err := executeCommand(agent.cmd, "--help")
	require.NoError(t, err)
	require.NotContains(t, help, "report_usage")

	var commandMap bytes.Buffer
	printCommandMap(&commandMap, agent.cmd, mapModeTree)
	require.NotContains(t, commandMap.String(), "report_usage")
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

func decodeAgentReportUsageResponse(t *testing.T, output string) agentReportUsageResponse {
	t.Helper()

	require.True(t, strings.HasSuffix(output, "\n"))
	var response agentReportUsageResponse
	require.NoError(t, json.Unmarshal([]byte(output), &response))
	return response
}
