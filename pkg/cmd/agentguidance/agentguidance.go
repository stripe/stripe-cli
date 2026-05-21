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
━━ Stripe CLI Agent Guidance ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Before running a command, consider which API surface is best for
  the task:

  • First-class commands (e.g. stripe customers list) and
    stripe get/post cover the public Stripe API.
  • stripe spec covers dynamic APIs for account configuration,
    settings, and other tasks not in the public API.

    stripe spec search <query>    find dynamic API methods by keyword
    stripe spec details <method>  full description and parameters
    stripe spec execute <method>  call the method with your auth

  Run ` + "`stripe spec --help`" + ` for more information.

  To snooze this message for the rest of today:
    stripe agent-guidance snooze
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`
