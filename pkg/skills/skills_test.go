package skills

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchIndex_success(t *testing.T) {
	expected := &SkillsIndex{
		Skills: []SkillEntry{
			{Name: "stripe", URL: "https://example.com/stripe.md", Description: "Stripe skill"},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer ts.Close()

	got, err := fetchIndexFromURL(ts.URL)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, len(got.Skills))
	assert.Equal(t, "stripe", got.Skills[0].Name)
	assert.Equal(t, "https://example.com/stripe.md", got.Skills[0].URL)
}

func TestFetchIndex_httpError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := fetchIndexFromURL(ts.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status fetching skills index")
}

func TestInstall_createsSkillsDir(t *testing.T) {
	skillContent := "# Stripe skill content"

	var skillServerURL string
	skillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, skillContent)
	}))
	defer skillServer.Close()
	skillServerURL = skillServer.URL + "/stripe.md"

	indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := SkillsIndex{
			Skills: []SkillEntry{
				{Name: "stripe", URL: skillServerURL},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer indexServer.Close()

	destDir := t.TempDir()
	installed, err := installFromURL(destDir, indexServer.URL)
	require.NoError(t, err)

	skillsDir := filepath.Join(destDir, ".skills")
	_, statErr := os.Stat(skillsDir)
	assert.NoError(t, statErr, ".skills directory should exist")
	assert.Equal(t, 1, len(installed))
}

func TestInstall_downloadsAllSkills(t *testing.T) {
	skillContents := map[string]string{
		"/skill-a.md": "# Skill A",
		"/skill-b.md": "# Skill B",
	}

	skillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := skillContents[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, content)
	}))
	defer skillServer.Close()

	indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := SkillsIndex{
			Skills: []SkillEntry{
				{Name: "skill-a", URL: skillServer.URL + "/skill-a.md"},
				{Name: "skill-b", URL: skillServer.URL + "/skill-b.md"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer indexServer.Close()

	destDir := t.TempDir()
	installed, err := installFromURL(destDir, indexServer.URL)
	require.NoError(t, err)
	assert.Equal(t, 2, len(installed))

	for _, path := range installed {
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, "installed file should exist: %s", path)
	}

	contentA, err := os.ReadFile(filepath.Join(destDir, ".skills", "skill-a.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Skill A", string(contentA))

	contentB, err := os.ReadFile(filepath.Join(destDir, ".skills", "skill-b.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Skill B", string(contentB))
}

func TestInstall_emptyIndex(t *testing.T) {
	indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := SkillsIndex{Skills: []SkillEntry{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer indexServer.Close()

	destDir := t.TempDir()
	installed, err := installFromURL(destDir, indexServer.URL)
	require.NoError(t, err)
	assert.Nil(t, installed)

	skillsDir := filepath.Join(destDir, ".skills")
	_, statErr := os.Stat(skillsDir)
	assert.True(t, os.IsNotExist(statErr), ".skills directory should not be created for empty index")
}

func TestInstall_skillFetchError(t *testing.T) {
	skillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/good-skill.md" {
			fmt.Fprint(w, "# Good skill")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer skillServer.Close()

	indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := SkillsIndex{
			Skills: []SkillEntry{
				{Name: "good-skill", URL: skillServer.URL + "/good-skill.md"},
				{Name: "bad-skill", URL: skillServer.URL + "/bad-skill.md"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	}))
	defer indexServer.Close()

	destDir := t.TempDir()
	installed, err := installFromURL(destDir, indexServer.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-skill")

	// The first skill should have been installed before the error
	assert.Equal(t, 1, len(installed))
}
