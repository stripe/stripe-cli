package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

type unsafePluginPathError struct {
	path   string
	reason string
}

func (e *unsafePluginPathError) Error() string {
	return fmt.Sprintf("unsafe plugin storage path %q: %s; use a new directory owned by the current user and not writable by group or others, then reinstall plugins from a trusted source (do not reuse contents from the unsafe store)", e.path, e.reason)
}

func unsafePluginPath(path, reason string) error {
	return errorcategory.With(&unsafePluginPathError{path: path, reason: reason}, errorcategory.Filesystem)
}

// IsUnsafePluginStorage reports a failed local plugin storage trust check.
func IsUnsafePluginStorage(err error) bool {
	var unsafePath *unsafePluginPathError
	return errors.As(err, &unsafePath)
}

// checkPluginStorage checks the config store even when the binaries live in an
// override directory. Missing paths are allowed so a safe first install works.
func checkPluginStorage(cfg config.IConfig, fs afero.Fs, paths ...string) error {
	configDir := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginsDir := getPluginsDir(cfg)
	for _, root := range []string{configDir, pluginsDir, getLocalPluginMetadataDir(cfg)} {
		if err := checkPluginPath(fs, root, root); err != nil {
			return err
		}
	}
	if _, ok := cfg.(*config.Config); ok {
		if path := viper.ConfigFileUsed(); path != "" {
			if err := checkPluginPath(fs, path, path); err != nil {
				return err
			}
		}
	}
	for _, path := range paths {
		root := path
		for _, store := range []string{configDir, pluginsDir} {
			if pluginPathWithin(path, store) {
				root = store
				break
			}
		}
		if err := checkPluginPath(fs, path, root); err != nil {
			return err
		}
	}
	return nil
}

// State-only helpers have no filesystem argument. Custom config implementations
// own their persistence; the standard config writer always uses the OS filesystem.
func checkPluginStateWrite(cfg config.IConfig) error {
	if _, ok := cfg.(*config.Config); ok {
		return checkPluginStorage(cfg, afero.NewOsFs())
	}
	return nil
}

func pluginPathWithin(path, root string) bool {
	path, pathErr := filepath.Abs(path)
	root, rootErr := filepath.Abs(root)
	if pathErr != nil || rootErr != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
