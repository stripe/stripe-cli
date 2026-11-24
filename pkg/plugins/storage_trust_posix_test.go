//go:build unix

package plugins

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

func protectedPluginConfig(t *testing.T) *CustomTestConfig {
	t.Helper()
	t.Setenv("STRIPE_PLUGINS_PATH", "")
	previous := PluginsPath
	PluginsPath = ""
	t.Cleanup(func() { PluginsPath = previous })
	return &CustomTestConfig{customConfigPath: filepath.Join(t.TempDir(), "config")}
}

func requireUnsafeStorage(t *testing.T, err error, path string) {
	t.Helper()
	require.Error(t, err)
	require.True(t, IsUnsafePluginStorage(err), "%v", err)
	require.Contains(t, err.Error(), path)
	require.Contains(t, err.Error(), "reinstall")
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	require.Equal(t, errorcategory.Filesystem, category)
}

func TestPluginStorageRejectsWritablePaths(t *testing.T) {
	for _, mode := range []os.FileMode{0777, 0775} {
		for _, kind := range []string{"config", "config-parent", "store", "store-parent", "metadata-dir", "metadata", "cache", "binary", "version-dir", "sticky-store"} {
			t.Run(fmt.Sprintf("%s/%o", kind, mode), func(t *testing.T) {
				cfg := protectedPluginConfig(t)
				fs := afero.NewOsFs()
				plugin := Plugin{Shortname: "sample", Binary: "sample"}
				require.NoError(t, writeLocalPluginMetadata(cfg, fs, plugin))
				binary := filepath.Join(getPluginsDir(cfg), "sample", "1.0.0", "sample")
				require.NoError(t, os.MkdirAll(filepath.Dir(binary), 0755))
				require.NoError(t, os.WriteFile(binary, []byte("unused"), 0755))
				cache := getCachedPluginManifestPath(cfg)
				require.NoError(t, os.WriteFile(cache, []byte("unused"), 0644))
				metadata, err := getLocalPluginMetadataPath(cfg, "sample")
				require.NoError(t, err)
				paths := map[string]string{
					"config": cfg.customConfigPath, "config-parent": filepath.Dir(cfg.customConfigPath),
					"store": getPluginsDir(cfg), "store-parent": filepath.Join(getPluginsDir(cfg), "sample"),
					"metadata-dir": getLocalPluginMetadataDir(cfg), "metadata": metadata,
					"cache": cache, "binary": binary, "version-dir": filepath.Dir(binary),
					"sticky-store": getPluginsDir(cfg),
				}
				unsafe := paths[kind]
				original, err := os.Stat(unsafe)
				require.NoError(t, err)
				t.Cleanup(func() { _ = os.Chmod(unsafe, original.Mode()) })
				if kind == "sticky-store" {
					mode |= os.ModeSticky
				}
				require.NoError(t, os.Chmod(unsafe, mode))
				requireUnsafeStorage(t, checkPluginStorage(cfg, fs, metadata, cache, binary), unsafe)
			})
		}
	}
}

func TestPluginStorageSafeFirstInstallAndOverrides(t *testing.T) {
	cfg := protectedPluginConfig(t)
	fs := afero.NewOsFs()
	plugin := Plugin{Shortname: "sample", Binary: "sample"}
	require.NoError(t, plugin.verifychecksumAndSavePlugin([]byte("development"), cfg, fs, localDevelopmentVersion))
	require.NoError(t, PersistInstalledPluginState(cfg, fs, plugin))
	_, err := LookUpPlugin(context.Background(), cfg, fs, "sample")
	require.NoError(t, err)

	for _, compiled := range []bool{false, true} {
		t.Run(fmt.Sprintf("compiled=%v", compiled), func(t *testing.T) {
			override := filepath.Join(t.TempDir(), "plugins")
			require.NoError(t, os.MkdirAll(override, 0755))
			if compiled {
				PluginsPath = override
				defer func() { PluginsPath = "" }()
			} else {
				t.Setenv("STRIPE_PLUGINS_PATH", override)
			}
			marker := filepath.Join(t.TempDir(), "executed")
			payload := []byte(fmt.Sprintf("#!/bin/sh\nprintf unused > %q\n", marker))
			require.NoError(t, plugin.verifychecksumAndSavePlugin(payload, cfg, fs, localDevelopmentVersion))
			_, client, err := plugin.dispensePluginInterface(cfg, fs, localDevelopmentVersion, "", log.NewEntry(log.New()))
			require.Error(t, err) // The marker executable does not implement the plugin protocol.
			require.NotNil(t, client)
			client.Kill()
			_, err = os.Stat(marker)
			require.NoError(t, err)
			require.NoError(t, os.Chmod(override, 0777))
			requireUnsafeStorage(t, plugin.verifychecksumAndSavePlugin([]byte("unused"), cfg, fs, localDevelopmentVersion), override)
		})
	}
}

func TestPluginStorageRejectsUnsafeXDGParentBeforeInstall(t *testing.T) {
	protectedPluginConfig(t)
	xdg := filepath.Join(t.TempDir(), "xdg")
	require.NoError(t, os.MkdirAll(xdg, 0755))
	require.NoError(t, os.Chmod(xdg, 0777))
	t.Setenv("XDG_CONFIG_HOME", xdg)
	cfg := &config.Config{}
	fs := afero.NewOsFs()
	_, err := ResolvePluginForInstall(context.Background(), cfg, fs, "sample", "1.0.0", "http://127.0.0.1:1", "http://127.0.0.1:1")
	requireUnsafeStorage(t, err, xdg)
	_, err = os.Stat(filepath.Join(xdg, "stripe"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestPluginStorageSymlinkTargetsAndAncestors(t *testing.T) {
	for _, kind := range []string{"file-target", "target-parent", "link-parent", "safe-directory", "safe-file", "parent-traversal", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			cfg := protectedPluginConfig(t)
			fs := afero.NewOsFs()
			require.NoError(t, os.MkdirAll(cfg.customConfigPath, 0755))
			targetDir := filepath.Join(t.TempDir(), "target")
			require.NoError(t, os.MkdirAll(targetDir, 0755))
			target := filepath.Join(targetDir, "sample")
			require.NoError(t, os.WriteFile(target, []byte("unused"), 0755))
			link := filepath.Join(cfg.customConfigPath, "link")
			check := link
			unsafe := ""
			switch kind {
			case "file-target":
				require.NoError(t, os.Chmod(target, 0777))
				unsafe = target
			case "target-parent":
				require.NoError(t, os.Chmod(targetDir, 0777))
				unsafe = targetDir
			case "link-parent":
				require.NoError(t, os.Chmod(cfg.customConfigPath, 0777))
				unsafe = cfg.customConfigPath
			case "safe-directory":
				target = targetDir
				check = filepath.Join(link, "sample")
			case "parent-traversal":
				target = filepath.Join(targetDir, "child") + "/../sample"
				unsafe = link
			case "cycle":
				target = link
				unsafe = link
			}
			require.NoError(t, os.Symlink(target, link))
			err := checkPluginStorage(cfg, fs, check)
			if unsafe == "" {
				require.NoError(t, err)
			} else {
				requireUnsafeStorage(t, err, unsafe)
			}
		})
	}
}

type pluginOwnershipInfo struct {
	os.FileInfo
	metadata interface{}
}

func (i pluginOwnershipInfo) Sys() interface{} { return i.metadata }

type pluginInspectionFs struct {
	afero.Fs
	path string
	info os.FileInfo
	err  error
}

func (fs pluginInspectionFs) LstatIfPossible(path string) (os.FileInfo, bool, error) {
	if path == fs.path {
		return fs.info, true, fs.err
	}
	return fs.Fs.(afero.Lstater).LstatIfPossible(path)
}

func (fs pluginInspectionFs) ReadlinkIfPossible(path string) (string, error) {
	return fs.Fs.(afero.LinkReader).ReadlinkIfPossible(path)
}

func TestPluginStorageSystemAlias(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "private", "var")
	require.NoError(t, os.MkdirAll(filepath.Join(target, "user", "config"), 0755))
	alias := filepath.Join(root, "var")
	require.NoError(t, os.Symlink("private/var", alias))
	info, err := os.Lstat(alias)
	require.NoError(t, err)
	fs := pluginInspectionFs{Fs: afero.NewOsFs(), path: alias, info: pluginOwnershipInfo{FileInfo: info, metadata: &syscall.Stat_t{Uid: 0}}}
	store := filepath.Join(alias, "user", "config")
	require.NoError(t, checkPluginPath(fs, filepath.Join(store, "metadata.toml"), store))

	link := filepath.Join(store, "link")
	require.NoError(t, os.Symlink(link, link))
	err = checkPluginPath(fs, link, store)
	requireUnsafeStorage(t, err, link)
	require.Contains(t, err.Error(), "symbolic link cycle")
}

func TestPluginStorageSystemAncestors(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/system/private/var/user/config", 0755))
	// Model a protected root-owned ancestor independently of the test runner UID.
	info, err := fs.Stat("/system")
	require.NoError(t, err)
	rootFs := pluginInspectionFs{Fs: fs, path: "/system", info: pluginOwnershipInfo{FileInfo: info, metadata: &syscall.Stat_t{Uid: 0}}}
	require.NoError(t, checkPluginPath(rootFs, "/system/private/var/user/config", "/system/private/var/user/config"))
	stickyInfo := pluginOwnershipInfo{FileInfo: stickyPluginInfo{FileInfo: info}, metadata: &syscall.Stat_t{Uid: 0}}
	rootFs.info = stickyInfo
	require.NoError(t, checkPluginPath(rootFs, "/system/private/var/user/config", "/system/private/var/user/config"))
	requireUnsafeStorage(t, checkPluginPath(rootFs, "/system", "/system"), "/system")
}

type stickyPluginInfo struct{ os.FileInfo }

func (i stickyPluginInfo) Mode() os.FileMode { return os.ModeDir | os.ModeSticky | 0777 }

func TestPluginStorageOwnershipAndInspectionFailures(t *testing.T) {
	cfg := protectedPluginConfig(t)
	require.NoError(t, os.MkdirAll(cfg.customConfigPath, 0755))
	info, err := os.Stat(cfg.customConfigPath)
	require.NoError(t, err)
	for _, metadata := range []interface{}{nil, &syscall.Stat_t{Uid: uint32(os.Geteuid() + 1)}} {
		fs := pluginInspectionFs{Fs: afero.NewOsFs(), path: cfg.customConfigPath, info: pluginOwnershipInfo{FileInfo: info, metadata: metadata}}
		requireUnsafeStorage(t, checkPluginStorage(cfg, fs), cfg.customConfigPath)
	}
	fs := pluginInspectionFs{Fs: afero.NewOsFs(), path: cfg.customConfigPath, err: os.ErrPermission}
	requireUnsafeStorage(t, checkPluginStorage(cfg, fs), cfg.customConfigPath)
}

func TestPluginLaunchRejectsWritableBinaryBeforeExecution(t *testing.T) {
	for _, version := range []string{"1.0.0", localDevelopmentVersion} {
		t.Run(version, func(t *testing.T) {
			cfg := protectedPluginConfig(t)
			fs := afero.NewOsFs()
			marker := filepath.Join(t.TempDir(), "executed")
			payload := []byte(fmt.Sprintf("#!/bin/sh\nprintf unused > %q\n", marker))
			sum := sha256.Sum256(payload)
			plugin := Plugin{Shortname: "sample", Binary: "sample", Releases: []Release{{Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Sum: fmt.Sprintf("%x", sum)}}}
			require.NoError(t, plugin.verifychecksumAndSavePlugin(payload, cfg, fs, version))
			binary := filepath.Join(getPluginsDir(cfg), "sample", version, "sample")
			require.NoError(t, os.Chmod(binary, 0777))
			memory := afero.NewMemMapFs()
			require.NoError(t, memory.MkdirAll(filepath.Dir(binary), 0755))
			require.NoError(t, afero.WriteFile(memory, binary, payload, 0755))
			for _, launchFS := range []afero.Fs{fs, memory} {
				_, client, err := plugin.dispensePluginInterface(cfg, launchFS, version, "", log.NewEntry(log.New()))
				require.Nil(t, client)
				requireUnsafeStorage(t, err, binary)
			}
			_, err := os.Stat(marker)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestPluginLaunchAnchorsRelativeOverrideBeforeChangingDirectory(t *testing.T) {
	for _, version := range []string{"1.0.0", localDevelopmentVersion} {
		t.Run(version, func(t *testing.T) {
			cfg := protectedPluginConfig(t)
			t.Chdir(t.TempDir())
			t.Setenv("STRIPE_PLUGINS_PATH", "plugins")
			fs := afero.NewOsFs()
			trustedMarker := filepath.Join(t.TempDir(), "trusted")
			payload := []byte(fmt.Sprintf("#!/bin/sh\nprintf unused > %q\n", trustedMarker))
			sum := sha256.Sum256(payload)
			plugin := Plugin{Shortname: "sample", Binary: "sample", Releases: []Release{{Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Sum: fmt.Sprintf("%x", sum)}}}
			require.NoError(t, plugin.verifychecksumAndSavePlugin(payload, cfg, fs, version))

			cwd := t.TempDir()
			untrustedMarker := filepath.Join(t.TempDir(), "untrusted")
			lookalike := filepath.Join(cwd, "plugins", "sample", version, "sample")
			require.NoError(t, os.MkdirAll(filepath.Dir(lookalike), 0755))
			require.NoError(t, os.WriteFile(lookalike, []byte(fmt.Sprintf("#!/bin/sh\nprintf unused > %q\n", untrustedMarker)), 0755))
			require.NoError(t, os.Chmod(lookalike, 0777))
			_, client, err := plugin.dispensePluginInterface(cfg, fs, version, cwd, log.NewEntry(log.New()))
			require.Error(t, err) // The trusted marker does not implement the plugin protocol.
			require.NotNil(t, client)
			client.Kill()
			_, err = os.Stat(trustedMarker)
			require.NoError(t, err)
			_, err = os.Stat(untrustedMarker)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestPluginLaunchAndHooksRejectUnsafeStore(t *testing.T) {
	cfg := protectedPluginConfig(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(cfg.customConfigPath))
	actualCfg := &config.Config{}
	store := getPluginsDir(actualCfg)
	require.NoError(t, os.MkdirAll(store, 0777))
	require.NoError(t, os.Chmod(store, 0777))
	plugin := Plugin{Shortname: "sample", Binary: "sample"}
	fs := afero.NewOsFs()
	for _, version := range []string{"", "1.0.0"} {
		requireUnsafeStorage(t, plugin.RunWithoutAutoUpgrade(context.Background(), actualCfg, fs, nil, "", version, "", "", ""), store)
	}
	requireUnsafeStorage(t, plugin.PostInstall(context.Background(), actualCfg, fs, localDevelopmentVersion, "", "", "", ""), store)
	helper := NewCoreCLIHelper(context.Background(), actualCfg, fs, "", "", "")
	requireUnsafeStorage(t, helper.RunPeerPlugin("sample", nil, ""), store)
}
