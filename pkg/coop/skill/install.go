package skill

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/stripe/stripe-cli/pkg/version"
)

// ManifestFileName marks a skill directory as CLI-managed. A same-name skill
// directory without it is treated as user-authored and never touched.
const ManifestFileName = ".stripe-cli-skill.json"

const manifestManagedBy = "stripe-cli"

// ErrUnmanagedSkill reports that the install target already holds a skill this
// CLI does not manage. Callers must not overwrite it.
var ErrUnmanagedSkill = errors.New("a skill not managed by the Stripe CLI already exists at the target path")

// Manifest records how an installed skill copy was produced so later runs can
// distinguish "ours, current", "ours, stale", and "not ours".
type Manifest struct {
	ManagedBy   string `json:"managed_by"`
	Skill       string `json:"skill"`
	CLIVersion  string `json:"cli_version"`
	ContentHash string `json:"content_hash"`
}

// InstallResult describes what Install did.
type InstallResult string

const (
	ResultInstalled InstallResult = "installed"
	ResultUpgraded  InstallResult = "upgraded"
	ResultCurrent   InstallResult = "current"
)

// Install writes the bundled skill to targetDir (the full skill directory,
// e.g. ~/.agents/skills/stripe-coop). It is idempotent and atomic:
//   - an existing CLI-managed copy with the same content hash is left as is;
//   - a stale CLI-managed copy is replaced via a staged directory and renames,
//     so a crash never leaves a half-written skill in place;
//   - anything else at the path returns ErrUnmanagedSkill untouched.
func Install(targetDir string) (InstallResult, error) {
	files, err := Files()
	if err != nil {
		return "", err
	}
	contentHash := hashFiles(files)

	existing, err := readManifest(targetDir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// Fresh install below.
	case err != nil:
		return "", err
	case existing.ManagedBy != manifestManagedBy || existing.Skill != Name:
		return "", fmt.Errorf("%w: %s", ErrUnmanagedSkill, targetDir)
	case existing.ContentHash == contentHash:
		return ResultCurrent, nil
	}
	upgrading := err == nil

	parent := filepath.Dir(targetDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("creating skills directory: %w", err)
	}
	sweepInstallDebris(parent)

	staging, err := stageSkill(parent, files, contentHash)
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := swapIntoPlace(staging, targetDir, upgrading); err != nil {
		return "", err
	}
	if upgrading {
		return ResultUpgraded, nil
	}
	return ResultInstalled, nil
}

// installDebrisMaxAge guards the sweep against deleting a concurrent
// install's live staging directory: only debris comfortably older than any
// in-flight install is removed.
const installDebrisMaxAge = time.Hour

// sweepInstallDebris removes staging/retired directories abandoned by earlier
// interrupted runs (hard kills between staging and the rename swap, or a
// retired copy whose removal failed). Best effort: a directory still held open
// elsewhere is retried on the next install.
func sweepInstallDebris(parent string) {
	for _, pattern := range []string{
		fmt.Sprintf(".%s.staging-*", Name),
		fmt.Sprintf(".%s.retired-*", Name),
	} {
		matches, err := filepath.Glob(filepath.Join(parent, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Lstat(match)
			if err != nil || time.Since(info.ModTime()) < installDebrisMaxAge {
				continue
			}
			_ = os.RemoveAll(match)
		}
	}
}

// readManifest returns the manifest at dir. os.ErrNotExist means the target
// directory itself does not exist; a directory without a readable manifest is
// reported as unmanaged.
func readManifest(dir string) (Manifest, error) {
	if _, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, os.ErrNotExist
		}
		return Manifest{}, fmt.Errorf("checking existing skill at %s: %w", dir, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ManifestFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, fmt.Errorf("%w: %s", ErrUnmanagedSkill, dir)
		}
		return Manifest{}, fmt.Errorf("reading skill manifest at %s: %w", dir, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: %s has an unreadable manifest", ErrUnmanagedSkill, dir)
	}
	return manifest, nil
}

func stageSkill(parent string, files map[string][]byte, contentHash string) (string, error) {
	staging := filepath.Join(parent, fmt.Sprintf(".%s.staging-%s", Name, uuid.New().String()[:8]))
	for path, contents := range files {
		destination := filepath.Join(staging, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			_ = os.RemoveAll(staging)
			return "", fmt.Errorf("staging skill directory: %w", err)
		}
		if err := os.WriteFile(destination, contents, 0o644); err != nil {
			_ = os.RemoveAll(staging)
			return "", fmt.Errorf("staging skill file %s: %w", path, err)
		}
	}
	manifest, err := json.MarshalIndent(Manifest{
		ManagedBy:   manifestManagedBy,
		Skill:       Name,
		CLIVersion:  version.Version,
		ContentHash: contentHash,
	}, "", "  ")
	if err != nil {
		_ = os.RemoveAll(staging)
		return "", fmt.Errorf("encoding skill manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(staging, ManifestFileName), append(manifest, '\n'), 0o644); err != nil {
		_ = os.RemoveAll(staging)
		return "", fmt.Errorf("staging skill manifest: %w", err)
	}
	return staging, nil
}

// swapIntoPlace renames staging to target. When replacing an existing managed
// copy the old directory is moved aside first (rename onto an existing
// directory fails on some platforms) and restored if the final rename fails.
func swapIntoPlace(staging, target string, replaceExisting bool) error {
	if !replaceExisting {
		if err := os.Rename(staging, target); err != nil {
			return fmt.Errorf("installing skill at %s: %w", target, err)
		}
		return nil
	}

	retired := filepath.Join(filepath.Dir(target), fmt.Sprintf(".%s.retired-%s", Name, uuid.New().String()[:8]))
	if err := os.Rename(target, retired); err != nil {
		return fmt.Errorf("retiring stale skill at %s: %w", target, err)
	}
	if err := os.Rename(staging, target); err != nil {
		if restoreErr := os.Rename(retired, target); restoreErr != nil {
			return errors.Join(
				fmt.Errorf("installing skill at %s: %w", target, err),
				fmt.Errorf("restoring previous skill: %w", restoreErr),
			)
		}
		return fmt.Errorf("installing skill at %s: %w", target, err)
	}
	_ = os.RemoveAll(retired)
	return nil
}
