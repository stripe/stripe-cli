package agentsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	CursorMarketplace            = "cursor-public"
	TargetCursorPlugin           = "stripe@cursor-public"
	CursorPluginName             = "stripe"
	CursorPluginsDir             = ".cursor/plugins"
	cursorStripeMarketplaceID    = "408" // numeric ID for stripe@cursor-public in Cursor's marketplace
	cursorVscdbQueryTimeout      = 5 * time.Second
	cursorManualInstallHint      = "run /add-plugin stripe inside Cursor agent"
	sqlite3BinaryName            = "sqlite3"
)

// cursorInstalledPlugin is an entry in Cursor's state.vscdb installedIds JSON array.
type cursorInstalledPlugin struct {
	ID      string   `json:"id"`
	Sources []string `json:"sources"`
}

// cursorStripePluginStatus is the result of querying Cursor's plugin registry.
type cursorStripePluginStatus struct {
	installed         bool
	supportsDetection bool
	scope             string
	projectPath       string
}

// cursorStateDBPath returns the path to Cursor's global state.vscdb for the
// current platform.
func cursorStateDBPath(home, goos string) (string, error) {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb"), nil
	case "linux":
		return filepath.Join(home, ".config", "Cursor", "User", "globalStorage", "state.vscdb"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb"), nil
	default:
		return "", fmt.Errorf("unsupported OS %q", goos)
	}
}

// cursorWorkspaceURI returns the file:// URI Cursor uses for a local workspace folder.
func cursorWorkspaceURI(absPath string) string {
	slashPath := strings.ReplaceAll(absPath, "\\", "/")
	if strings.HasPrefix(slashPath, "/") {
		return "file://" + slashPath
	}
	return "file:///" + slashPath
}

// cursorStripePluginFromVscdb reports whether the Stripe Cursor plugin is enabled
// at user-global or current-project scope by querying state.vscdb via sqlite3.
// When sqlite3 or the database is unavailable, supportsDetection is false.
func cursorStripePluginFromVscdb(s Scanner, runOutput RunOutputFunc) (cursorStripePluginStatus, error) {
	result := cursorStripePluginStatus{}

	if _, err := s.LookPath(sqlite3BinaryName); err != nil {
		return result, nil
	}

	home, err := s.HomeDir()
	if err != nil {
		return result, err
	}

	dbPath, err := cursorStateDBPath(home, runtime.GOOS)
	if err != nil {
		return result, err
	}

	if _, err := s.Stat(dbPath); err != nil {
		return result, nil
	}

	cwd, err := s.WorkDir()
	if err != nil {
		return result, err
	}
	workspaceURI, err := filepath.Abs(cwd)
	if err != nil {
		return result, err
	}
	workspaceURI = cursorWorkspaceURI(workspaceURI)

	query := fmt.Sprintf(
		`SELECT key, value FROM ItemTable WHERE key LIKE 'cursor.plugins.installedIds.%%|no-workspace' OR key LIKE 'cursor.plugins.installedIds.%%|%s';`,
		escapeSQLiteString(workspaceURI),
	)

	ctx, cancel := context.WithTimeout(context.Background(), cursorVscdbQueryTimeout)
	defer cancel()

	out, err := runOutput(ctx, sqlite3BinaryName, dbPath, query)
	if err != nil {
		return result, nil
	}

	userInstalled, projectInstalled, projectPath := parseCursorInstalledIdsOutput(string(out), workspaceURI)
	result.supportsDetection = true

	switch {
	case userInstalled && projectInstalled:
		result.installed = true
		result.scope = "user"
	case userInstalled:
		result.installed = true
		result.scope = "user"
	case projectInstalled:
		result.installed = true
		result.scope = "project"
		result.projectPath = projectPath
	}

	return result, nil
}

func escapeSQLiteString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// parseCursorInstalledIdsOutput parses sqlite3 stdout for enabled Stripe plugin
// entries. Keys use '|' as a delimiter between the account prefix and scope URI.
func parseCursorInstalledIdsOutput(output, workspaceURI string) (userInstalled, projectInstalled bool, projectPath string) {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := splitCursorInstalledIdsRow(line)
		if !ok {
			continue
		}

		if !cursorInstalledIdsContainsStripe(value) {
			continue
		}

		scopeURI := cursorInstalledIdsScopeURI(key)
		switch scopeURI {
		case "no-workspace":
			userInstalled = true
		case workspaceURI:
			projectInstalled = true
			projectPath = workspaceURIToPath(workspaceURI)
		}
	}
	return userInstalled, projectInstalled, projectPath
}

func splitCursorInstalledIdsRow(line string) (key, value string, ok bool) {
	idx := strings.Index(line, "[")
	if idx <= 1 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx-1]), strings.TrimSpace(line[idx:]), true
}

func cursorInstalledIdsScopeURI(key string) string {
	if i := strings.LastIndex(key, "|"); i >= 0 {
		return key[i+1:]
	}
	return ""
}

func workspaceURIToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return ""
	}
	path := strings.TrimPrefix(uri, "file://")
	if strings.HasPrefix(path, "/") && len(path) > 2 && path[2] == ':' {
		// file:///C:/path on Windows
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

func cursorInstalledIdsContainsStripe(valueJSON string) bool {
	var plugins []cursorInstalledPlugin
	if err := json.Unmarshal([]byte(valueJSON), &plugins); err != nil {
		return false
	}
	for _, plugin := range plugins {
		if plugin.ID == cursorStripeMarketplaceID {
			return true
		}
	}
	return false
}

// cursorPluginVersionFromCache reads the Stripe plugin version from Cursor's
// on-disk plugin cache. The cache alone does not indicate enablement.
func cursorPluginVersionFromCache(s Scanner, home string) string {
	cacheRoot := filepath.Join(home, CursorPluginsDir, "cache", CursorMarketplace, CursorPluginName)
	hashes, err := s.ReadDir(cacheRoot)
	if err != nil {
		return ""
	}

	for _, hash := range hashes {
		if !hash.IsDir() {
			continue
		}
		body, err := s.ReadFile(filepath.Join(cacheRoot, hash.Name(), ".cursor-plugin", "plugin.json"))
		if err != nil {
			continue
		}
		var meta struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(body, &meta) == nil && meta.Version != "" {
			return meta.Version
		}
	}
	return ""
}
