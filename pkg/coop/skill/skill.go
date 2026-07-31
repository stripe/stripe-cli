// Package skill bundles the stripe-coop Agent Skill inside the CLI binary and
// installs it into coding-harness skill directories. The skill teaches an
// agent how to operate the Co-op command protocol, so it is version-locked to
// this binary: it is embedded at build time and never downloaded at launch.
package skill

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Name is the skill directory and frontmatter name.
const Name = "stripe-coop"

//go:embed all:stripe-coop
var embedded embed.FS

// Files returns the bundled skill files keyed by path relative to the skill
// directory (e.g. "SKILL.md", "references/command-api.md").
func Files() (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(embedded, Name, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := embedded.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded skill file %s: %w", path, err)
		}
		files[strings.TrimPrefix(path, Name+"/")] = contents
		return nil
	})
	if err != nil {
		return nil, err
	}
	if _, ok := files["SKILL.md"]; !ok {
		return nil, fmt.Errorf("embedded skill is missing SKILL.md")
	}
	return files, nil
}

// ContentHash returns a stable hash of every bundled file path and content.
// Two binaries bundling identical skill content produce identical hashes, so
// installs are idempotent across CLI versions that did not change the skill.
func ContentHash() (string, error) {
	files, err := Files()
	if err != nil {
		return "", err
	}
	return hashFiles(files), nil
}

func hashFiles(files map[string][]byte) string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	digest := sha256.New()
	for _, path := range paths {
		digest.Write([]byte(path))
		digest.Write([]byte{0})
		digest.Write(files[path])
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}
