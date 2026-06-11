//go:build !darwin

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWSLFromVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "microsoft keyword",
			content: "Linux version 5.15.0-microsoft-standard-WSL2",
			want:    true,
		},
		{
			name:    "Microsoft capitalised",
			content: "Linux version 5.15.0-Microsoft-standard",
			want:    true,
		},
		{
			name:    "wsl keyword",
			content: "Linux version 5.15.0 (wsl@build)",
			want:    true,
		},
		{
			name:    "WSL uppercase",
			content: "Linux version 5.15.0 (WSL2)",
			want:    true,
		},
		{
			name:    "plain linux",
			content: "Linux version 6.1.0-28-amd64 (debian-kernel@lists.debian.org)",
			want:    false,
		},
		{
			name:    "empty",
			content: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isWSLFromVersion(tt.content))
		})
	}
}

func TestIsWSL_UnreadableProcVersion(t *testing.T) {
	require.False(t, isWSLFromVersion(""))
}
