package agentsetup

import (
	"context"
	"io"
	"sort"
	"strings"
)

const (
	StatusNotDetected = "not_detected"
	StatusInstalled   = "installed"
	StatusMissing     = "missing"
	StatusError       = "error"

	ActionNone      = "none"
	ActionInstall   = "install"
	ActionReinstall = "reinstall"
	// ActionManual means setup cannot be automated and the user must perform a
	// step themselves (e.g. Cursor plugins are installed from inside Cursor).
	ActionManual = "manual"
)

// Provider detects and configures Stripe tooling for one AI coding client.
type Provider interface {
	ID() string
	Detect() Status
	Plan(Status, bool) Plan
	Apply(context.Context, io.Writer, Plan) error
}

// DefaultProviders returns production setup providers keyed by client id.
func DefaultProviders() map[string]Provider {
	scanner := DefaultScanner()
	claude := NewClaudeProvider(scanner, RunCommand)
	cursor := NewCursorProvider(scanner, RunCommand)
	codex := NewCodexProvider(scanner, RunCommand)
	return map[string]Provider{
		claude.ID(): claude,
		cursor.ID(): cursor,
		codex.ID():  codex,
	}
}

func SupportedProviderIDs(providers map[string]Provider) string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// Status is the read-only setup status for one AI coding client.
type Status struct {
	Client         string       `json:"client"`
	DisplayName    string       `json:"display_name"`
	Detected       bool         `json:"detected"`
	ExecutablePath string       `json:"executable_path,omitempty"`
	Plugin         PluginStatus `json:"plugin"`
	Status         string       `json:"status"`
	Error          string       `json:"error,omitempty"`
}

// PluginStatus is the plugin-specific part of a client setup status.
type PluginStatus struct {
	Installed bool   `json:"installed"`
	ID        string `json:"id,omitempty"`
	Version   string `json:"version,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Project   string `json:"project_path,omitempty"`
	StatePath string `json:"state_path,omitempty"`
}

// Plan describes the next setup action for a provider.
type Plan struct {
	Action  string   `json:"action"`
	Command []string `json:"command,omitempty"`
	// Manual holds the instruction shown for ActionManual plans.
	Manual string `json:"manual,omitempty"`
}
