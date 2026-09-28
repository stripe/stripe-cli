package reporting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestCommandBucket(t *testing.T) {
	tests := []struct {
		name              string
		commandPath       string
		generatedResource bool
		expected          string
	}{
		{name: "generated resource", commandPath: "stripe customers create", generatedResource: true, expected: "resources"},
		{name: "generated v2 resource", commandPath: "stripe v2 billing meter_events create", generatedResource: true, expected: "resources"},
		{name: "generated resource without path", generatedResource: true, expected: "resources"},
		{name: "listen", commandPath: "stripe listen", expected: "listen"},
		{name: "listen without root prefix", commandPath: "listen", expected: "listen"},
		{name: "logs tail", commandPath: "stripe logs tail", expected: "logs_tail"},
		{name: "logs tail nested", commandPath: "stripe logs tail replay", expected: "logs_tail"},
		{name: "other logs command", commandPath: "stripe logs list", expected: "logs"},
		{name: "docs root", commandPath: "stripe docs", expected: "docs"},
		{name: "docs subtree", commandPath: "stripe docs prefs list", expected: "docs"},
		{name: "nested built-in", commandPath: "stripe config set color", expected: "config"},
		{name: "direct HTTP command", commandPath: "stripe get", expected: "get"},
		{name: "extra whitespace", commandPath: "  stripe   logs   tail  ", expected: "logs_tail"},
		{name: "empty", expected: "other"},
		{name: "root only", commandPath: "stripe", expected: "other"},
		{name: "unknown", commandPath: "stripe future-command run", expected: "other"},
		{name: "plugin", commandPath: "stripe custom-plugin action", expected: "other"},
		{name: "malformed prefix", commandPath: "stripe stripe listen", expected: "other"},
		{name: "malformed path", commandPath: "/stripe/listen", expected: "other"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := &stripe.CLIAnalyticsEventMetadata{
				CommandPath:       test.commandPath,
				GeneratedResource: test.generatedResource,
			}
			require.Equal(t, test.expected, commandBucket(metadata))
		})
	}
}

func TestCommandBucketRecognizesEveryBuiltIn(t *testing.T) {
	for command := range builtInCommandBuckets {
		t.Run(command, func(t *testing.T) {
			metadata := &stripe.CLIAnalyticsEventMetadata{CommandPath: "stripe " + command + " nested"}
			require.Equal(t, command, commandBucket(metadata))
		})
	}
}

func TestCommandBucketWithoutMetadata(t *testing.T) {
	require.Equal(t, otherCommandBucket, commandBucket(nil))
}
