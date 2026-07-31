package skill

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundledSkillHasCompleteLayout(t *testing.T) {
	files, err := Files()
	require.NoError(t, err)

	for _, path := range []string{
		"SKILL.md",
		"references/command-api.md",
		"references/node-contracts.md",
		"references/recovery.md",
	} {
		assert.Contains(t, files, path)
		assert.NotEmpty(t, files[path], "%s must not be empty", path)
	}

	// The skill stays lean: no README/changelog/install docs duplicating the
	// CLI-managed install flow.
	for path := range files {
		lower := strings.ToLower(path)
		assert.NotContains(t, lower, "readme")
		assert.NotContains(t, lower, "changelog")
		assert.NotContains(t, lower, "install")
	}
}

func TestBundledSkillFrontmatterIsValid(t *testing.T) {
	files, err := Files()
	require.NoError(t, err)
	frontmatter := skillFrontmatter(t, string(files["SKILL.md"]))

	name, ok := frontmatter["name"]
	require.True(t, ok, "frontmatter must declare name")
	assert.Equal(t, Name, name, "frontmatter name must match the skill directory name")
	assert.LessOrEqual(t, len(name), 64)
	assert.Regexp(t, `^[a-z0-9]+(-[a-z0-9]+)*$`, name)

	description, ok := frontmatter["description"]
	require.True(t, ok, "frontmatter must declare description")
	assert.NotEmpty(t, description)
	assert.LessOrEqual(t, len(description), 1024, "description exceeds the agentskills spec limit")

	// The description gates activation: it must scope the skill to Co-op
	// sessions and away from ordinary Stripe development.
	assert.Contains(t, description, "stripe coop")
	assert.Contains(t, description, "coop_")
	assert.Contains(t, description, "Do not use for ordinary Stripe coding outside Co-op")
}

func TestBundledSkillReferenceIntegrity(t *testing.T) {
	files, err := Files()
	require.NoError(t, err)

	body := string(files["SKILL.md"])
	for _, reference := range []string{
		"references/command-api.md",
		"references/node-contracts.md",
		"references/recovery.md",
	} {
		assert.Contains(t, body, reference, "SKILL.md must point at %s", reference)
	}

	// Every references/... mention across the skill resolves to a bundled file.
	for path, contents := range files {
		for _, line := range strings.Split(string(contents), "\n") {
			for _, token := range strings.Fields(line) {
				token = strings.Trim(token, "`()[].,:;")
				if !strings.HasPrefix(token, "references/") {
					continue
				}
				assert.Contains(t, files, token, "%s mentions missing reference %q", path, token)
			}
		}
	}
}

func TestBundledSkillCoversLifecycleContract(t *testing.T) {
	files, err := Files()
	require.NoError(t, err)
	skillBody := string(files["SKILL.md"])
	allContent := skillBody + string(files["references/command-api.md"]) +
		string(files["references/node-contracts.md"]) + string(files["references/recovery.md"])

	for _, command := range []string{
		"start-work", "report-check", "report-work", "skip",
		"await-review", "resume", "next-action", "start-followup",
	} {
		assert.Contains(t, skillBody, command, "SKILL.md must name control command %s", command)
	}
	for _, nodeType := range []string{
		"apiRequest", "asyncHandler", "uiComponent", "testHelper",
		"cliCommand", "dashboard", "setUpWebhooks",
	} {
		assert.Contains(t, allContent, nodeType)
	}
	for _, contract := range []string{
		"${node",                           // reference resolution
		"pm_card_visa",                     // card-number safety
		"raw request body",                 // webhook signature verification
		"one node at a time",               // pacing
		"stripe sandbox create --from-git", // auth bootstrap
		"heartbeat",                        // never touch internal state
	} {
		assert.Contains(t, allContent, contract)
	}
}

func TestContentHashIsStableAndContentSensitive(t *testing.T) {
	first, err := ContentHash()
	require.NoError(t, err)
	second, err := ContentHash()
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Len(t, first, 64)

	files, err := Files()
	require.NoError(t, err)
	baseline := hashFiles(files)
	files["SKILL.md"] = append(files["SKILL.md"], '\n')
	assert.NotEqual(t, baseline, hashFiles(files))
}

// skillFrontmatter parses the simple key: value YAML frontmatter block.
func skillFrontmatter(t *testing.T, body string) map[string]string {
	t.Helper()
	require.True(t, strings.HasPrefix(body, "---\n"), "SKILL.md must start with frontmatter")
	rest := body[len("---\n"):]
	end := strings.Index(rest, "\n---")
	require.Greater(t, end, 0, "frontmatter must be terminated")

	frontmatter := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		frontmatter[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return frontmatter
}
