package plugins

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestParseCompletionOutput tests parsing of Cobra's __complete protocol output.
//
// @see https://github.com/spf13/cobra/blob/main/completions.go
func TestParseCompletionOutput(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		wantCompletions []string
		wantDirective   cobra.ShellCompDirective
	}{
		{
			name:            "valid output with descriptions",
			output:          "create\tCreate a new app\nstart\tStart the dev server\nupload\tUpload your app\n:4\n",
			wantCompletions: []string{"create\tCreate a new app", "start\tStart the dev server", "upload\tUpload your app"},
			wantDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:            "valid output without descriptions",
			output:          "create\nstart\nupload\n:4\n",
			wantCompletions: []string{"create", "start", "upload"},
			wantDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:            "valid output with default directive",
			output:          "create\n:0\n",
			wantCompletions: []string{"create"},
			wantDirective:   cobra.ShellCompDirectiveDefault,
		},
		{
			name:            "directive only, no completions",
			output:          ":4\n",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:            "empty output",
			output:          "",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveError,
		},
		{
			name:            "missing directive line",
			output:          "create\nstart\n",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveError,
		},
		{
			name:            "non-numeric directive",
			output:          "create\n:abc\n",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveError,
		},
		{
			name:            "output with no trailing newline",
			output:          "create\n:4",
			wantCompletions: []string{"create"},
			wantDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:            "combined directives",
			output:          "flag-value\n:6\n",
			wantCompletions: []string{"flag-value"},
			wantDirective:   cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:            "whitespace-only output",
			output:          "   \n",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveError,
		},
		{
			name:            "empty directive value after colon",
			output:          "create\n:\n",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := parseCompletionOutput(tt.output)
			assert.Equal(t, tt.wantCompletions, completions)
			assert.Equal(t, tt.wantDirective, directive)
		})
	}
}
