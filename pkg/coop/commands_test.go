package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExactCoopCommands(t *testing.T) {
	assert.Equal(t, "stripe coop status", StatusCommand(""))
	assert.Equal(t, "stripe coop status --session=coop_123", StatusCommand("coop_123"))
	assert.Equal(t, `stripe coop run "one-time-payment"`, RunCommand("one-time-payment"))
	assert.Equal(t, "stripe coop stop --session=coop_123", StopCommand("coop_123"))
	assert.Equal(t, `stripe coop agent start-work --session=coop_123 --step=2 --note="Beginning: Product"`, StartWorkCommand("coop_123", 2, "Beginning: Product"))
	assert.Equal(t, "stripe coop agent await-review --session=coop_123 --step=2", AwaitReviewCommand("coop_123", 2))
	assert.Equal(t, "stripe coop agent next-action --session=coop_123", NextActionCommand("coop_123", ""))
	assert.Equal(t, "stripe coop agent next-action --session=coop_123 --completed=deploy", NextActionCommand("coop_123", "deploy"))
	assert.Equal(t, `stripe coop agent start-followup --session="coop_123" --action="deploy" --target="Vercel"`, StartFollowupCommand("coop_123", "deploy", "Vercel"))
}
