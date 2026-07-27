package agentsetup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorFromOutput(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{
			name: "prefers failed line",
			raw:  `✘ Failed to install plugin "stripe@claude-plugins-official"`,
			want: `Failed to install plugin "stripe@claude-plugins-official"`,
		},
		{
			name: "collapses carriage returns",
			raw:  "Installing...\r✘ failed to write plugin manifest",
			want: "failed to write plugin manifest",
		},
		{
			name: "falls back to last line",
			raw:  "starting up\n\nall done here",
			want: "all done here",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, errorFromOutput([]byte(tc.raw)))
		})
	}
}
