package agentsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	ClientCursor      = "cursor"
	CursorBinaryName  = "cursor"
	CursorPluginName  = "stripe"          // plugin directory name under a marketplace
	CursorPluginsDir  = ".cursor/plugins" // relative to the home directory
	CursorDisplayName = "Cursor"
)

// CursorProvider detects the Stripe plugin for Cursor.
//
// Unlike Claude Code, Cursor has no `cursor plugin` CLI and no JSON registry:
// the `cursor` binary is only the editor launcher, and installed plugins are
// recorded as a directory tree under ~/.cursor/plugins/cache/<marketplace>/
// <plugin>/<hash>/ (a `.cache-complete` marker plus a .cursor-plugin/plugin.json
// metadata file). Detection therefore walks the filesystem.
//
// Cursor plugins are installed from inside Cursor via the `/add-plugin stripe`
// slash command; there is no shell CLI installer. This provider therefore
// detects the plugin from disk and, when it is missing, points the user at the
// in-app command rather than shelling out.
type CursorProvider struct {
	Scanner Scanner
}

// NewCursorProvider returns a Cursor setup provider. The RunCommandFunc argument
// is accepted for signature parity with the other providers but is unused,
// because Cursor has no CLI installer.
func NewCursorProvider(scanner Scanner, _ RunCommandFunc) Provider {
	return CursorProvider{Scanner: scanner}
}

func (p CursorProvider) ID() string { return ClientCursor }

func (p CursorProvider) Detect() Status { return p.Scanner.ScanCursor() }

func (p CursorProvider) Plan(status Status, _ bool) Plan {
	if status.Detected && !status.Plugin.Installed && status.Status != StatusError {
		return Plan{Action: ActionManual, Manual: "run /add-plugin stripe inside Cursor Agent to install the Stripe plugin"}
	}
	return Plan{Action: ActionNone}
}

// Apply is a no-op for Cursor: the plugin is installed from inside Cursor, so the
// setup flow surfaces the ActionManual instruction rather than calling Apply.
func (p CursorProvider) Apply(_ context.Context, _ io.Writer, _ Plan) error {
	return nil
}

type cursorPluginRef struct {
	version string
	scope   string
	path    string
}

// ScanCursor returns Cursor installation and Stripe plugin status by walking the
// on-disk plugin cache.
func (s Scanner) ScanCursor() Status {
	s = s.withDefaults()

	status := Status{
		Client:      ClientCursor,
		DisplayName: CursorDisplayName,
		Status:      StatusNotDetected,
	}

	binPath, err := s.LookPath(CursorBinaryName)
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

	pluginsRoot := filepath.Join(home, CursorPluginsDir)
	status.Plugin.StatePath = pluginsRoot

	if id, ref, ok := findCursorStripePlugin(s, pluginsRoot); ok {
		status.Plugin.Installed = true
		status.Plugin.ID = id
		status.Plugin.Version = ref.version
		status.Plugin.Scope = ref.scope
		status.Plugin.StatePath = ref.path
		status.Status = StatusInstalled
	}

	return status
}

// findCursorStripePlugin looks for a completed Stripe plugin install under
// ~/.cursor/plugins/cache/<marketplace>/stripe/<hash>/. A `.cache-complete`
// marker file indicates the install finished.
func findCursorStripePlugin(s Scanner, pluginsRoot string) (string, cursorPluginRef, bool) {
	cacheRoot := filepath.Join(pluginsRoot, "cache")
	marketplaces, err := os.ReadDir(cacheRoot)
	if err != nil {
		return "", cursorPluginRef{}, false
	}

	for _, marketplace := range marketplaces {
		if !marketplace.IsDir() {
			continue
		}
		stripeDir := filepath.Join(cacheRoot, marketplace.Name(), CursorPluginName)
		hashes, err := os.ReadDir(stripeDir)
		if err != nil {
			continue
		}
		for _, hash := range hashes {
			if !hash.IsDir() {
				continue
			}
			hashPath := filepath.Join(stripeDir, hash.Name())
			if _, err := os.Stat(filepath.Join(hashPath, ".cache-complete")); err != nil {
				continue
			}

			ref := cursorPluginRef{scope: "user", path: hashPath}
			if body, err := s.ReadFile(filepath.Join(hashPath, ".cursor-plugin", "plugin.json")); err == nil {
				var meta struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}
				if json.Unmarshal(body, &meta) == nil {
					ref.version = meta.Version
				}
			}
			return CursorPluginName + "@" + marketplace.Name(), ref, true
		}
	}

	return "", cursorPluginRef{}, false
}
