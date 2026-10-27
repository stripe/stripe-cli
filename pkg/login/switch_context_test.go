package login

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
)

func rowKeys(rows []switchRow) []string {
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.id+"/"+r.mode)
	}
	return keys
}

func TestBuildSwitchRows_GroupsSandboxesUnderParent(t *testing.T) {
	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
		{ID: "acct_sandbox2", Name: "Acme Staging", Modes: []string{"test"}},
		{ID: "acct_orphan", Name: "Mystery Sandbox", Modes: []string{"test"}},
		{ID: "acct_orphan2", Name: "Second Mystery", Modes: []string{"test"}},
	}
	parentAccountIDs := map[string]string{
		"acct_sandbox1": "acct_live1",
		"acct_sandbox2": "acct_live1",
	}

	rows, nameW, idW := buildSwitchRows(accounts, parentAccountIDs, "", "test")

	assert.Equal(t, []string{
		"acct_live1/live",
		"acct_sandbox1/test",
		"acct_sandbox2/test",
		"acct_orphan/test",
		"acct_orphan2/test",
	}, rowKeys(rows), "standalone sandboxes should appear as their own accounts, not grouped under a heading")

	for _, r := range rows {
		if r.id == "acct_sandbox1" || r.id == "acct_sandbox2" {
			assert.True(t, r.nested, "sandboxes with a known parent should be nested")
		}
		if r.id == "acct_live1" || r.id == "acct_orphan" || r.id == "acct_orphan2" {
			assert.False(t, r.nested)
		}
	}

	assert.Equal(t, len("Mystery Sandbox"), nameW)
	assert.Equal(t, len("acct_sandbox1"), idW)
}

func TestBuildSwitchRows_LegacyCombinedAccountNestsTestMode(t *testing.T) {
	accounts := []config.AuthorizedAccount{
		{ID: "acct_legacy", Name: "Acme Inc", Modes: []string{"live", "test"}},
	}

	rows, _, _ := buildSwitchRows(accounts, nil, "", "test")

	assert.Len(t, rows, 2)
	assert.Equal(t, "Acme Inc", rows[0].name)
	assert.False(t, rows[0].nested)
	assert.Equal(t, "Test mode", rows[1].name)
	assert.True(t, rows[1].nested)
}

func TestBuildSwitchRows_LiveModeRendersBeforeNestedTestModeRegardlessOfAPIOrder(t *testing.T) {
	accounts := []config.AuthorizedAccount{
		{ID: "acct_legacy", Name: "Acme Inc", Modes: []string{"test", "live"}},
	}

	rows, _, _ := buildSwitchRows(accounts, nil, "", "test")

	assert.Len(t, rows, 2)
	assert.Equal(t, "live", rows[0].mode, "the live row should come first even when the API returns test before live")
	assert.False(t, rows[0].nested)
	assert.Equal(t, "Test mode", rows[1].name)
	assert.True(t, rows[1].nested)
}

func TestBuildSwitchRows_NoSandboxes(t *testing.T) {
	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
	}

	rows, _, _ := buildSwitchRows(accounts, nil, "", "test")

	assert.Equal(t, []string{"acct_live1/live"}, rowKeys(rows))
}

func TestBuildSwitchRows_MarksActiveRow(t *testing.T) {
	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live", "test"}},
	}

	rows, _, _ := buildSwitchRows(accounts, nil, "acct_live1", "test")

	assert.False(t, rows[0].active)
	assert.True(t, rows[1].active)
}
