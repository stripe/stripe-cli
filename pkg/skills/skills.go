// Package skills manages downloading Stripe AI skills.
package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const indexURL = "https://docs.stripe.com/.well-known/skills/index.json"

// SkillEntry represents a single entry in the skills index.
type SkillEntry struct {
	// Name is a human-readable identifier for the skill (e.g. "stripe").
	Name string `json:"name"`
	// URL is the absolute URL of the skill file to download.
	URL string `json:"url"`
	// Description is an optional human-readable description.
	Description string `json:"description,omitempty"`
}

// SkillsIndex is the top-level structure of the index.json document.
type SkillsIndex struct {
	Skills []SkillEntry `json:"skills"`
}

// FetchIndex downloads and parses the skills index from docs.stripe.com.
func FetchIndex() (*SkillsIndex, error) {
	return fetchIndexFromURL(indexURL)
}

func fetchIndexFromURL(url string) (*SkillsIndex, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching skills index: %s", resp.Status)
	}

	var index SkillsIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to parse skills index: %w", err)
	}
	return &index, nil
}

// Install downloads all skills from the index into the `.skills/` directory
// inside destDir (typically the user's current working directory).
// It returns a list of installed file paths and an error if any step failed.
func Install(destDir string) ([]string, error) {
	return installFromURL(destDir, indexURL)
}

// installFromURL is the internal implementation that accepts a custom index URL,
// making it easy to test with a mock HTTP server.
func installFromURL(destDir, url string) ([]string, error) {
	index, err := fetchIndexFromURL(url)
	if err != nil {
		return nil, err
	}

	if len(index.Skills) == 0 {
		return nil, nil
	}

	skillsDir := filepath.Join(destDir, ".skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .skills directory: %w", err)
	}

	var installed []string
	for _, skill := range index.Skills {
		path, err := downloadSkill(skillsDir, skill)
		if err != nil {
			return installed, err
		}
		installed = append(installed, path)
	}
	return installed, nil
}

// downloadSkill fetches a single skill file and writes it into skillsDir.
// The filename is derived from the URL's base name.
func downloadSkill(skillsDir string, skill SkillEntry) (string, error) {
	resp, err := http.Get(skill.URL) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("failed to fetch skill %q: %w", skill.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status fetching skill %q: %s", skill.Name, resp.Status)
	}

	filename := filepath.Base(skill.URL)
	if filename == "." || filename == "/" {
		filename = skill.Name + ".md"
	}
	destPath := filepath.Join(skillsDir, filename)

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write skill %q: %w", skill.Name, err)
	}

	return destPath, nil
}
