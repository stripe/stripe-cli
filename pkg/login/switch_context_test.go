package login

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
)

func rowKeys(rows []switchRow) []string {
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		switch {
		case r.isSeparator:
			keys = append(keys, "<sep>")
		case r.isSectionLabel:
			keys = append(keys, "<label:"+r.label+">")
		default:
			keys = append(keys, r.id+"/"+r.mode)
		}
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
		"<sep>",
		"<label:Other sandboxes>",
		"acct_orphan/test",
		"acct_orphan2/test",
	}, rowKeys(rows))

	for _, r := range rows {
		if r.isSeparator || r.isSectionLabel {
			assert.False(t, r.selectable())
			continue
		}
		assert.True(t, r.selectable(), "account rows, including nested ones, must still be selectable")
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

func TestBuildSwitchRows_SoleUnknownParentSandboxShownAsRegularAccount(t *testing.T) {
	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_orphan", Name: "Mystery Sandbox", Modes: []string{"test"}},
	}

	rows, _, _ := buildSwitchRows(accounts, nil, "", "test")

	assert.Equal(t, []string{
		"acct_live1/live",
		"acct_orphan/test",
	}, rowKeys(rows), "a single sandbox with no known parent should appear as a regular account, not under an Other sandboxes heading")
	for _, r := range rows {
		assert.False(t, r.nested)
	}
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

func TestBuildSwitchRows_NoSandboxesOmitsOtherSection(t *testing.T) {
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
