// Package agentsetup contains helpers for detecting and configuring AI coding
// agent integrations.
package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ClientClaudeCode      = "claude-code"
	ClaudeBinaryName      = "claude"
	TargetClaudePlugin    = "stripe@claude-plugins-official"
	LocalClaudePlugin     = "stripe@stripe"
	ClaudeMarketplace     = "claude-plugins-official"
	ClaudePluginStatePath = ".claude/plugins/installed_plugins.json"

	StatusNotDetected = "not_detected"
	StatusInstalled   = "installed"
	StatusMissing     = "missing"
	StatusError       = "error"
)

// LookPathFunc matches exec.LookPath and exists to make detection testable.
type LookPathFunc func(string) (string, error)

// ReadFileFunc matches os.ReadFile and exists to make status parsing testable.
type ReadFileFunc func(string) ([]byte, error)

// HomeDirFunc matches os.UserHomeDir and exists to make status parsing testable.
type HomeDirFunc func() (string, error)

// WorkDirFunc matches os.Getwd and exists to make local plugin scope testable.
type WorkDirFunc func() (string, error)

// RunCommandFunc runs a command. The production implementation streams stdio.
type RunCommandFunc func(context.Context, string, ...string) error

// Scanner scans local agent installations without mutating them.
type Scanner struct {
	LookPath LookPathFunc
	ReadFile ReadFileFunc
	HomeDir  HomeDirFunc
	WorkDir  WorkDirFunc
}

// ClaudeStatus is the read-only setup status for Claude Code.
type ClaudeStatus struct {
	Client          string `json:"client"`
	Detected        bool   `json:"detected"`
	ExecutablePath  string `json:"executable_path,omitempty"`
	PluginInstalled bool   `json:"plugin_installed"`
	PluginID        string `json:"plugin_id,omitempty"`
	PluginVersion   string `json:"plugin_version,omitempty"`
	PluginScope     string `json:"plugin_scope,omitempty"`
	PluginProject   string `json:"plugin_project_path,omitempty"`
	PluginStatePath string `json:"plugin_state_path,omitempty"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}

type installedPluginState struct {
	Version int                             `json:"version"`
	Plugins map[string][]installedPluginRef `json:"plugins"`
}

type installedPluginRef struct {
	Scope       string `json:"scope"`
	Version     string `json:"version"`
	Installed   string `json:"installPath"`
	ProjectPath string `json:"projectPath"`
}

// DefaultScanner returns a Scanner backed by the real OS.
func DefaultScanner() Scanner {
	return Scanner{
		LookPath: exec.LookPath,
		ReadFile: os.ReadFile,
		HomeDir:  os.UserHomeDir,
		WorkDir:  os.Getwd,
	}
}

// ScanClaude returns Claude Code installation and Stripe plugin status.
func (s Scanner) ScanClaude() ClaudeStatus {
	s = s.withDefaults()

	status := ClaudeStatus{
		Client: ClientClaudeCode,
		Status: StatusNotDetected,
	}

	claudePath, err := s.LookPath(ClaudeBinaryName)
	if err != nil {
		return status
	}
	status.Detected = true
	status.ExecutablePath = claudePath
	status.Status = StatusMissing

	home, err := s.HomeDir()
	if err != nil {
		status.Status = StatusError
		status.Error = fmt.Sprintf("locating home directory: %v", err)
		return status
	}
	status.PluginStatePath = filepath.Join(home, ClaudePluginStatePath)

	body, err := s.ReadFile(status.PluginStatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status
		}
		status.Status = StatusError
		status.Error = fmt.Sprintf("reading Claude plugin state: %v", err)
		return status
	}

	var pluginState installedPluginState
	if err := json.Unmarshal(body, &pluginState); err != nil {
		status.Status = StatusError
		status.Error = fmt.Sprintf("parsing Claude plugin state: %v", err)
		return status
	}

	workDir, err := s.WorkDir()
	if err != nil {
		status.Status = StatusError
		status.Error = fmt.Sprintf("locating working directory: %v", err)
		return status
	}

	if id, ref, ok := findStripePlugin(pluginState.Plugins, workDir); ok {
		status.PluginInstalled = true
		status.PluginID = id
		status.PluginVersion = ref.Version
		status.PluginScope = ref.Scope
		status.PluginProject = ref.ProjectPath
		status.Status = StatusInstalled
	}

	return status
}

// ClaudeInstallCommand returns the command used to install the Stripe Claude
// Code plugin.
func ClaudeInstallCommand() (string, []string) {
	return ClaudeBinaryName, []string{"plugin", "install", TargetClaudePlugin}
}

// ClaudeMarketplaceUpdateCommand returns the command used to refresh Claude's
// official plugin marketplace metadata.
func ClaudeMarketplaceUpdateCommand() (string, []string) {
	return ClaudeBinaryName, []string{"plugin", "marketplace", "update", ClaudeMarketplace}
}

// RunCommand streams a command through the current process stdio.
func RunCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s Scanner) withDefaults() Scanner {
	defaults := DefaultScanner()
	if s.LookPath == nil {
		s.LookPath = defaults.LookPath
	}
	if s.ReadFile == nil {
		s.ReadFile = defaults.ReadFile
	}
	if s.HomeDir == nil {
		s.HomeDir = defaults.HomeDir
	}
	if s.WorkDir == nil {
		s.WorkDir = defaults.WorkDir
	}
	return s
}

func findStripePlugin(plugins map[string][]installedPluginRef, workDir string) (string, installedPluginRef, bool) {
	for _, id := range []string{TargetClaudePlugin, LocalClaudePlugin} {
		for _, ref := range plugins[id] {
			if pluginVisibleInWorkDir(ref, workDir) {
				return id, ref, true
			}
		}
	}
	return "", installedPluginRef{}, false
}

func pluginVisibleInWorkDir(ref installedPluginRef, workDir string) bool {
	if ref.Scope != "local" || ref.ProjectPath == "" {
		return true
	}

	rel, err := filepath.Rel(ref.ProjectPath, workDir)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel))
}
