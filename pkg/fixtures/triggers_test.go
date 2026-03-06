package fixtures

import (
	"encoding/json"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventsMapFilesExist verifies every Events map entry points to an
// embedded fixture file that can be opened and read.
func TestEventsMapFilesExist(t *testing.T) {
	for event, filepath := range Events {
		t.Run(event, func(t *testing.T) {
			f, err := triggers.Open(filepath)
			require.NoError(t, err, "event %q references missing file: %s", event, filepath)
			defer f.Close()

			data, err := io.ReadAll(f)
			require.NoError(t, err, "failed to read file for event %q: %s", event, filepath)
			require.NotEmpty(t, data, "file is empty for event %q: %s", event, filepath)
		})
	}
}

// TestAllFixtureFilesAreReferenced verifies no orphaned JSON files exist in
// the triggers/ directory without a corresponding Events map entry.
func TestAllFixtureFilesAreReferenced(t *testing.T) {
	referenced := make(map[string]bool)
	for _, filepath := range Events {
		referenced[filepath] = true
	}

	err := fs.WalkDir(triggers, "triggers", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		assert.True(t, referenced[path], "orphaned fixture file not in Events map: %s", path)
		return nil
	})
	require.NoError(t, err)
}

// TestFixtureFilesHaveValidStructure verifies every embedded fixture file
// is valid JSON with the required _meta and fixtures fields.
func TestFixtureFilesHaveValidStructure(t *testing.T) {
	err := fs.WalkDir(triggers, "triggers", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}

		t.Run(path, func(t *testing.T) {
			f, err := triggers.Open(path)
			require.NoError(t, err)
			defer f.Close()

			data, err := io.ReadAll(f)
			require.NoError(t, err)

			var fixtureData FixtureData
			err = json.Unmarshal(data, &fixtureData)
			require.NoError(t, err, "invalid JSON in %s", path)

			assert.NotEmpty(t, fixtureData.Requests, "empty fixtures array in %s", path)

			for i, req := range fixtureData.Requests {
				assert.NotEmpty(t, req.Name, "fixture %d missing name in %s", i, path)
				assert.NotEmpty(t, req.Path, "fixture %d missing path in %s", i, path)
				assert.NotEmpty(t, req.Method, "fixture %d missing method in %s", i, path)
			}
		})

		return nil
	})
	require.NoError(t, err)
}

// TestEventsMapAliasesAreDocumented verifies that when multiple event names
// share the same fixture file, we are aware of them. Update this list when
// adding intentional aliases.
func TestEventsMapAliasesAreDocumented(t *testing.T) {
	knownAliases := map[string][]string{
		"triggers/topup.created.json": {"topup.created", "topup.succeeded"},
		"triggers/invoice.paid.json":  {"invoice.paid", "invoice_payment.paid"},
	}

	fileToEvents := make(map[string][]string)
	for event, file := range Events {
		fileToEvents[file] = append(fileToEvents[file], event)
	}

	for file, events := range fileToEvents {
		if len(events) <= 1 {
			continue
		}
		expected, isKnown := knownAliases[file]
		if !isKnown {
			t.Errorf("unexpected alias: file %s mapped by multiple events %v — add to knownAliases if intentional", file, events)
			continue
		}
		assert.ElementsMatch(t, expected, events, "alias set changed for %s", file)
	}
}
