// Package fixtures provides tests for trigger fixture consistency.
//
// These tests ensure that the Events map in triggers.go and the fixture JSON files
// in the triggers/ directory stay in sync. They prevent entire classes of errors:
// - Registering an event without creating its fixture file
// - Creating a fixture file without registering it in the Events map
// - Typos in file paths
// - Invalid JSON in fixture files
//
// These tests are critical infrastructure for the trigger expansion work that will
// add 67 new events. Manual review doesn't scale - these tests catch mistakes instantly.
package fixtures

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventsMapHasCorrespondingFixtureFiles ensures that every event in the Events map
// has a corresponding fixture JSON file. This catches cases where you add an Events map
// entry but forget to create the fixture file (or delete the file but forget to remove
// the Events map entry).
func TestEventsMapHasCorrespondingFixtureFiles(t *testing.T) {
	for eventName, fixturePath := range Events {
		t.Run(eventName, func(t *testing.T) {
			// Check if the fixture file exists
			content, err := triggers.ReadFile(fixturePath)
			require.NoError(t, err, "Event %q maps to %q but file does not exist", eventName, fixturePath)
			require.NotEmpty(t, content, "Event %q maps to %q but file is empty", eventName, fixturePath)
		})
	}
}

// TestFixtureFilesHaveEventsMapEntry ensures that every fixture JSON file in the triggers/
// directory has a corresponding entry in the Events map. This prevents orphaned fixture files
// that cannot be triggered via the CLI.
func TestFixtureFilesHaveEventsMapEntry(t *testing.T) {
	// Read all fixture files from the embedded FS
	entries, err := triggers.ReadDir("triggers")
	require.NoError(t, err, "Failed to read triggers directory")

	// Build reverse map: filepath -> event name
	reverseMap := make(map[string]string)
	for eventName, fixturePath := range Events {
		reverseMap[fixturePath] = eventName
	}

	var orphanedFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		// Skip non-JSON files
		if !strings.HasSuffix(filename, ".json") {
			continue
		}

		fixturePath := "triggers/" + filename
		if _, exists := reverseMap[fixturePath]; !exists {
			orphanedFiles = append(orphanedFiles, fixturePath)
		}
	}

	assert.Empty(t, orphanedFiles, "Found fixture files without Events map entries (orphaned files): %v", orphanedFiles)
}

// TestNoMissingFixtureFiles ensures that if an event name is in the Events map,
// the corresponding fixture file actually exists. This is a safety check to catch
// typos or missing files.
func TestNoMissingFixtureFiles(t *testing.T) {
	var missingFiles []string

	for eventName, fixturePath := range Events {
		_, err := triggers.ReadFile(fixturePath)
		if err != nil {
			missingFiles = append(missingFiles, fixturePath+" (for event: "+eventName+")")
		}
	}

	assert.Empty(t, missingFiles, "Events map references non-existent fixture files: %v", missingFiles)
}

// TestNoDuplicateEventClaims ensures no two fixture files claim the same event name,
// either via basename or via _meta.aliases. Duplicates would cause non-deterministic
// behavior since buildEventsMap iterates directory entries in filesystem order.
func TestNoDuplicateEventClaims(t *testing.T) {
	entries, err := triggers.ReadDir("triggers")
	require.NoError(t, err, "Failed to read triggers directory")

	// claimedBy maps event name → first file that claimed it.
	claimedBy := make(map[string]string)
	var conflicts []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := "triggers/" + entry.Name()
		basename := strings.TrimSuffix(entry.Name(), ".json")

		claim := func(event string) {
			if prior, exists := claimedBy[event]; exists {
				conflicts = append(conflicts, event+": claimed by both "+prior+" and "+path)
			} else {
				claimedBy[event] = path
			}
		}

		claim(basename)

		content, readErr := triggers.ReadFile(path)
		require.NoError(t, readErr)
		var data FixtureData
		if json.Unmarshal(content, &data) == nil {
			for _, alias := range data.Meta.Aliases {
				claim(alias)
			}
		}
	}

	assert.Empty(t, conflicts, "Multiple fixture files claim the same event name: %v", conflicts)
}

// TestFixtureFilesAreValidJSON validates all fixture files against the FixtureData schema.
// This catches JSON syntax errors, structural issues, and missing required fields.
func TestFixtureFilesAreValidJSON(t *testing.T) {
	entries, err := triggers.ReadDir("triggers")
	require.NoError(t, err, "Failed to read triggers directory")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			fixturePath := "triggers/" + entry.Name()
			content, err := triggers.ReadFile(fixturePath)
			require.NoError(t, err, "Failed to read %s", fixturePath)

			// Validate against the FixtureData struct schema
			var fixtureData FixtureData
			err = json.Unmarshal(content, &fixtureData)
			require.NoError(t, err, "Fixture file %s does not match FixtureData schema", fixturePath)

			// Validate the fixtures array is not empty
			require.NotEmpty(t, fixtureData.Requests, "Fixture file %s has empty fixtures array", fixturePath)

			// Validate each request has required fields
			for i, req := range fixtureData.Requests {
				assert.NotEmpty(t, req.Name, "Fixture %s request %d missing 'name' field", fixturePath, i)
				assert.NotEmpty(t, req.Path, "Fixture %s request %d missing 'path' field", fixturePath, i)
				assert.NotEmpty(t, req.Method, "Fixture %s request %d missing 'method' field", fixturePath, i)
			}
		})
	}
}

// TestEventNamesFollowConvention ensures event names follow Stripe's naming convention:
// resource.action (e.g., "customer.created") or resource.sub_resource.action
func TestEventNamesFollowConvention(t *testing.T) {
	for eventName := range Events {
		t.Run(eventName, func(t *testing.T) {
			// v2 events can have brackets, e.g., v2.core.account[configuration.customer].updated
			if strings.HasPrefix(eventName, "v2.") {
				// v2 events are more complex, just ensure they have dots
				assert.Contains(t, eventName, ".", "v2 event name %q should contain dots", eventName)
				return
			}

			// v1 events should follow resource.action or resource.sub_resource.action pattern
			parts := strings.Split(eventName, ".")
			assert.GreaterOrEqual(t, len(parts), 2, "Event name %q should have at least 2 parts (resource.action)", eventName)

			// Check that parts don't contain underscores at the start/end (common typo)
			for i, part := range parts {
				assert.False(t, strings.HasPrefix(part, "_"), "Event name %q part %d starts with underscore", eventName, i)
				assert.False(t, strings.HasSuffix(part, "_"), "Event name %q part %d ends with underscore", eventName, i)
			}
		})
	}
}
