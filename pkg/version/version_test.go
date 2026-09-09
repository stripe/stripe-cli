package version

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/installmethod"
)

func TestUpgradeNotice(t *testing.T) {
	// ansi styling is on by default even when stdout is not a terminal, so turn it
	// off to assert on the text itself.
	ansi.DisableColors = true
	t.Cleanup(func() { ansi.DisableColors = false })

	t.Run("names the command for a known install method", func(t *testing.T) {
		notice := upgradeNotice("v1.2.3", installmethod.UpgradeAdvice(installmethod.Homebrew, "darwin"))

		require.Contains(t, notice, "A newer version of the Stripe CLI is available, please update to: v1.2.3")
		require.Contains(t, notice, "Run brew upgrade stripe to upgrade.")
	})

	t.Run("reports the version alone when no command can be named", func(t *testing.T) {
		notice := upgradeNotice("v1.2.3", installmethod.UpgradeAdvice(installmethod.Unknown, "linux"))

		require.Contains(t, notice, "please update to: v1.2.3")
		require.NotContains(t, notice, "Run")
	})

	t.Run("prints nothing for a self-updating install method", func(t *testing.T) {
		require.Empty(t, upgradeNotice("v1.2.3", installmethod.UpgradeAdvice(installmethod.NPX, "darwin")))
	})
}

func TestNeedsToUpgrade(t *testing.T) {
	require.False(t, needsToUpgrade("4.2.4.2", "v4.2.4.2"))
	require.False(t, needsToUpgrade("4.2.4.2", "4.2.4.2"))
	require.True(t, needsToUpgrade("4.2.4.2", "4.2.4.3"))
	require.True(t, needsToUpgrade("4.2.4.2", "v4.2.4.3"))
	require.True(t, needsToUpgrade("v4.2.4.2", "v4.2.4.3"))
}
