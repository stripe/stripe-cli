package agentguidance

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

// MaybeEmit writes the agent guidance message to w when all gates pass:
//   - an AI agent is detected via env vars
//   - the first positional in args (after any leading flags) is not on
//     the suppression list
//   - snoozedUntil does not equal today's local ISO date
//
// Any failed gate is a no-op. The function performs no I/O beyond the
// single Fprint and never returns an error: emission is best-effort and
// must never block the user's command.
func MaybeEmit(getEnv func(string) string, w io.Writer, snoozedUntil string, today time.Time, args []string) {
	if useragent.DetectAIAgent(getEnv) == "" {
		return
	}
	if isSuppressedCommand(args) {
		return
	}
	if isSnoozedToday(snoozedUntil, today) {
		return
	}
	fmt.Fprint(w, message)
}

// SnoozeDate returns today's local date as an ISO string ("2026-05-21")
// for storage in config.
func SnoozeDate(today time.Time) string {
	return today.Format("2006-01-02")
}

func isSnoozedToday(stored string, today time.Time) bool {
	if stored == "" {
		return false
	}
	return stored == today.Format("2006-01-02")
}

func isSuppressedCommand(args []string) bool {
	// Skip leading flag tokens (-foo, --foo, --foo=bar, -foo bar) to find
	// the first positional. Cobra persistent flags can appear before the
	// subcommand, e.g. `stripe -p prod agent-guidance snooze`. Without
	// this, args[0] would be the flag, not the subcommand, and the
	// suppression list would miss.
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		hasInlineValue := strings.Contains(args[i], "=")
		i++
		// If the flag does NOT use --foo=bar form AND the next token is
		// not itself a flag, treat it as the flag's value and skip it.
		// We can't perfectly distinguish boolean flags without the cobra
		// flag set, but this heuristic handles the common cases
		// (-p prod, --color off, --api-key sk_test_...).
		if !hasInlineValue && i < len(args) && !strings.HasPrefix(args[i], "-") {
			i++
		}
	}
	if i >= len(args) {
		return true
	}
	switch args[i] {
	case "agent-guidance", "spec", "completion", "version",
		"--version", "-v", "help", "--help", "-h":
		return true
	}
	return false
}

const message = `
━━ Stripe CLI Agent Guidance — read this before responding ━━━━━━━

  This command will run and print its output below. Don't return a
  final answer to the user based only on this output. ALSO run:

    stripe spec search <task-keyword>

  Why: ` + "`stripe spec`" + ` covers Stripe functionality not always reachable
  via the public API — both Dashboard surfaces and capabilities that
  have no public-API equivalent at all. Treat the output below as a
  partial answer until spec confirms otherwise.

  Example. User asks: "update my branding color"
    ✗ stripe get /v1/account            → returns some fields, but
                                          may be a partial projection
    ✓ stripe spec search branding       → finds UpdateBrandingSettings
    ✓ stripe spec execute UpdateBrandingSettings -d primary_color=...

  ` + "`stripe spec search`" + ` is cheap. When in doubt, run it.

  To snooze for the rest of today: stripe agent-guidance snooze
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`
