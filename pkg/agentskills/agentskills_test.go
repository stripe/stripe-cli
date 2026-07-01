package agentskills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// startSkillsServer serves the index at /index.json and each file at
// /<skill>/<path>. content maps "<skill>/<file>" -> body. Missing routes 404.
// It points IndexURL at the test server for the test's duration.
func startSkillsServer(t *testing.T, index Index, content map[string]string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(index))
	})
	for path, body := range content {
		body := body
		mux.HandleFunc("/"+path, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, body)
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	original := IndexURL
	IndexURL = server.URL + "/index.json"
	t.Cleanup(func() { IndexURL = original })

	return server
}

func TestInstall_WritesNestedFiles(t *testing.T) {
	index := Index{Skills: []Skill{
		{Name: "stripe-best-practices", Files: []string{"SKILL.md", "references/billing.md"}},
		{Name: "upgrade-stripe", Files: []string{"SKILL.md"}},
	}}
	server := startSkillsServer(t, index, map[string]string{
		"stripe-best-practices/SKILL.md":              "# best practices",
		"stripe-best-practices/references/billing.md": "# billing",
		"upgrade-stripe/SKILL.md":                     "# upgrade",
	})

	dest := t.TempDir()
	installed, err := Install(context.Background(), server.Client(), dest)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"stripe-best-practices", "upgrade-stripe"}, installed)

	body, err := os.ReadFile(filepath.Join(dest, "stripe-best-practices", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "# best practices", string(body))

	body, err = os.ReadFile(filepath.Join(dest, "stripe-best-practices", "references", "billing.md"))
	require.NoError(t, err)
	require.Equal(t, "# billing", string(body))

	body, err = os.ReadFile(filepath.Join(dest, "upgrade-stripe", "SKILL.md"))
	require.NoError(t, err)
	require.Equal(t, "# upgrade", string(body))
}

func TestInstall_SkipsMissingFilesButKeepsSkill(t *testing.T) {
	index := Index{Skills: []Skill{
		{Name: "stripe-best-practices", Files: []string{"SKILL.md", "references/missing.md"}},
	}}
	server := startSkillsServer(t, index, map[string]string{
		"stripe-best-practices/SKILL.md": "# best practices",
		// references/missing.md has no route -> 404
	})

	dest := t.TempDir()
	installed, err := Install(context.Background(), server.Client(), dest)

	require.NoError(t, err)
	require.Equal(t, []string{"stripe-best-practices"}, installed)
	require.FileExists(t, filepath.Join(dest, "stripe-best-practices", "SKILL.md"))
	require.NoFileExists(t, filepath.Join(dest, "stripe-best-practices", "references", "missing.md"))
}

func TestInstall_SkillWithNoRetrievableFilesIsOmitted(t *testing.T) {
	index := Index{Skills: []Skill{
		{Name: "ok", Files: []string{"SKILL.md"}},
		{Name: "broken", Files: []string{"SKILL.md"}}, // no route
	}}
	server := startSkillsServer(t, index, map[string]string{
		"ok/SKILL.md": "# ok",
	})

	dest := t.TempDir()
	installed, err := Install(context.Background(), server.Client(), dest)

	require.NoError(t, err)
	require.Equal(t, []string{"ok"}, installed)
	require.NoDirExists(t, filepath.Join(dest, "broken"))
}

func TestInstall_IndexError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	original := IndexURL
	IndexURL = server.URL + "/index.json"
	t.Cleanup(func() { IndexURL = original })

	installed, err := Install(context.Background(), server.Client(), t.TempDir())

	require.Error(t, err)
	require.Nil(t, installed)
	require.Contains(t, err.Error(), "fetching skills index")
}
