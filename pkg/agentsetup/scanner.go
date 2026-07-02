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

// LookPathFunc matches exec.LookPath and exists to make detection testable.
type LookPathFunc func(string) (string, error)

// ReadFileFunc matches os.ReadFile and exists to make status parsing testable.
type ReadFileFunc func(string) ([]byte, error)

// HomeDirFunc matches os.UserHomeDir and exists to make status parsing testable.
type HomeDirFunc func() (string, error)

// WorkDirFunc matches os.Getwd and exists to make local plugin scope testable.
type WorkDirFunc func() (string, error)

// ReadDirFunc matches os.ReadDir and exists to make directory listing testable.
type ReadDirFunc func(string) ([]os.DirEntry, error)

// StatFunc matches os.Stat and exists to make file existence checks testable.
type StatFunc func(string) (os.FileInfo, error)

// RunCommandFunc runs a command. The production implementation streams stdio.
type RunCommandFunc func(context.Context, string, ...string) error

// Scanner scans local agent installations without mutating them.
type Scanner struct {
	LookPath LookPathFunc
	ReadFile ReadFileFunc
	HomeDir  HomeDirFunc
	WorkDir  WorkDirFunc
	ReadDir  ReadDirFunc
	Stat     StatFunc
}

// DefaultScanner returns a Scanner backed by the real OS.
func DefaultScanner() Scanner {
	return Scanner{
		LookPath: exec.LookPath,
		ReadFile: os.ReadFile,
		HomeDir:  os.UserHomeDir,
		WorkDir:  os.Getwd,
		ReadDir:  os.ReadDir,
		Stat:     os.Stat,
	}
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
	if s.ReadDir == nil {
		s.ReadDir = defaults.ReadDir
	}
	if s.Stat == nil {
		s.Stat = defaults.Stat
	}
	return s
}

// RunCommand streams a command through the current process stdio.
func RunCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pluginClient describes a client whose Stripe plugin state is a Claude-style
// installed_plugins.json registry. Claude Code uses this; Cursor and Codex
// record installs differently and have their own scanners.
type pluginClient struct {
	id           string
	displayName  string
	shortName    string // used in error messages, e.g. "Claude"
	binaryName   string
	targetPlugin string
	localPlugin  string // optional local marketplace id; "" when none
	pluginState  string // path under the home dir to installed_plugins.json
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

// scanPlugin returns installation and Stripe plugin status for a plugin-based
// client without mutating anything on disk.
func (s Scanner) scanPlugin(client pluginClient) Status {
	s = s.withDefaults()

	status := Status{
		Client:      client.id,
		DisplayName: client.displayName,
		Status:      StatusNotDetected,
	}

	binPath, err := s.LookPath(client.binaryName)
	if err != nil {
		return status
	}
	status.Detected = true
	status.ExecutablePath = binPath
	status.Status = StatusMissing

	home, err := s.HomeDir()
	if err != nil {
		status.Status = StatusError
		status.Error = fmt.Sprintf("locating home directory: %v", err)
		return status
	}
	status.Plugin.StatePath = filepath.Join(home, client.pluginState)

	body, err := s.ReadFile(status.Plugin.StatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status
		}
		status.Status = StatusError
		status.Error = fmt.Sprintf("reading %s plugin state: %v", client.shortName, err)
		return status
	}

	var pluginState installedPluginState
	if err := json.Unmarshal(body, &pluginState); err != nil {
		status.Status = StatusError
		status.Error = fmt.Sprintf("parsing %s plugin state: %v", client.shortName, err)
		return status
	}

	workDir, err := s.WorkDir()
	if err != nil {
		status.Status = StatusError
		status.Error = fmt.Sprintf("locating working directory: %v", err)
		return status
	}

	if id, ref, ok := findStripePlugin(pluginState.Plugins, client, workDir); ok {
		status.Plugin.Installed = true
		status.Plugin.ID = id
		status.Plugin.Version = ref.Version
		status.Plugin.Scope = ref.Scope
		status.Plugin.Project = ref.ProjectPath
		status.Status = StatusInstalled
	}

	return status
}

func findStripePlugin(plugins map[string][]installedPluginRef, client pluginClient, workDir string) (string, installedPluginRef, bool) {
	ids := []string{client.targetPlugin}
	if client.localPlugin != "" {
		ids = append(ids, client.localPlugin)
	}
	for _, id := range ids {
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
