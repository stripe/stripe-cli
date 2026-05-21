# CLI Agent Guidance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a CLI-level interstitial that fires on every Stripe CLI invocation in an AI-agent context, prompting the agent to consider `stripe spec` (Dynamic APIs) alongside the public API. Snoozeable per-machine-day via `stripe agent-guidance snooze`.

**Architecture:** New `pkg/cmd/agentguidance` package with pure-function emission/snooze logic. New `pkg/cmd/agent_guidance.go` cobra command. Two-line wiring in `pkg/cmd/root.go::Execute()` that mirrors the existing `emitClaudeCodePluginHint()` prior art. Snooze date stored in `[agent_guidance].snoozed_until` of `~/.config/stripe/config.toml`.

**Tech Stack:** Go, cobra, viper, testify. Repo: `stripe-cli` (`/Users/anirudhgoyal/stripe/stripe-cli`). Branch: `agent-guidance-interstitial` (already created, design doc already committed).

**Design doc:** `docs/plans/2026-05-21-cli-agent-guidance-design.md` (read this first for full context).

**Working directory for all commands:** `/Users/anirudhgoyal/stripe/stripe-cli`

---

## Pre-flight

Before starting Task 1, run from the stripe-cli repo root:

```bash
git status                          # confirm on branch agent-guidance-interstitial, clean
go build ./...                      # confirm baseline compiles
```

If `go build` fails on a clean checkout, stop and ask for help — don't proceed.

---

### Task 1: Scaffold the `agentguidance` package

**Files:**
- Create: `pkg/cmd/agentguidance/doc.go`

**Step 1: Create the package directory and doc file**

```bash
mkdir -p pkg/cmd/agentguidance
```

Write `pkg/cmd/agentguidance/doc.go`:

```go
// Package agentguidance emits a CLI-level interstitial message in
// AI-agent contexts, guiding agents toward the right API surface
// (public API vs the dynamic-API spec plugin).
package agentguidance
```

**Step 2: Verify the package compiles**

Run: `go build ./pkg/cmd/agentguidance/...`
Expected: exits 0, no output.

**Step 3: Commit**

```bash
git add pkg/cmd/agentguidance/doc.go
git commit -m "Scaffold agentguidance package"
```

---

### Task 2: Implement `Today()` with test-date override

**Files:**
- Create: `pkg/cmd/agentguidance/today.go`
- Create: `pkg/cmd/agentguidance/today_test.go`

**Step 1: Write failing test**

Write `pkg/cmd/agentguidance/today_test.go`:

```go
package agentguidance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToday(t *testing.T) {
	tests := []struct {
		name        string
		override    string
		assertFunc  func(t *testing.T, got time.Time)
	}{
		{
			name:     "no_override",
			override: "",
			assertFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now(), got, 5*time.Second)
			},
		},
		{
			name:     "valid_override",
			override: "2026-12-25",
			assertFunc: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.December, got.Month())
				assert.Equal(t, 25, got.Day())
			},
		},
		{
			name:     "malformed_override_falls_back",
			override: "not-a-date",
			assertFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now(), got, 5*time.Second)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRIPE_AGENT_GUIDANCE_TODAY", tc.override)
			got := Today()
			tc.assertFunc(t, got)
		})
	}
}
```

**Step 2: Run the test, verify it fails**

Run: `go test ./pkg/cmd/agentguidance/... -run TestToday -v`
Expected: FAIL with `undefined: Today`.

**Step 3: Implement `Today()`**

Write `pkg/cmd/agentguidance/today.go`:

```go
package agentguidance

import (
	"os"
	"time"
)

// Today returns the current local date, or a value parsed from the
// STRIPE_AGENT_GUIDANCE_TODAY override env var when set. The override
// is for E2E testing; not advertised to users.
func Today() time.Time {
	if override := os.Getenv("STRIPE_AGENT_GUIDANCE_TODAY"); override != "" {
		if t, err := time.ParseInLocation("2006-01-02", override, time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}
```

**Step 4: Run the test, verify it passes**

Run: `go test ./pkg/cmd/agentguidance/... -run TestToday -v`
Expected: all three subtests PASS.

**Step 5: Commit**

```bash
git add pkg/cmd/agentguidance/today.go pkg/cmd/agentguidance/today_test.go
git commit -m "Add Today helper with STRIPE_AGENT_GUIDANCE_TODAY override"
```

---

### Task 3: Implement `MaybeEmit` and helpers (table-driven TDD)

**Files:**
- Create: `pkg/cmd/agentguidance/agentguidance.go`
- Create: `pkg/cmd/agentguidance/agentguidance_test.go`

**Step 1: Write failing test**

Write `pkg/cmd/agentguidance/agentguidance_test.go`:

```go
package agentguidance

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMaybeEmit(t *testing.T) {
	today := time.Date(2026, 5, 21, 12, 0, 0, 0, time.Local)
	todayISO := "2026-05-21"
	yesterdayISO := "2026-05-20"

	noEnv := func(string) string { return "" }
	claudeEnv := func(k string) string {
		if k == "CLAUDECODE" {
			return "1"
		}
		return ""
	}

	tests := []struct {
		name          string
		getEnv        func(string) string
		snoozedUntil  string
		args          []string
		expectMessage bool
	}{
		{
			name:          "not_an_agent_silent",
			getEnv:        noEnv,
			snoozedUntil:  "",
			args:          []string{"customers", "list"},
			expectMessage: false,
		},
		{
			name:          "agent_writes_message",
			getEnv:        claudeEnv,
			snoozedUntil:  "",
			args:          []string{"customers", "list"},
			expectMessage: true,
		},
		{
			name:          "snoozed_today_silent",
			getEnv:        claudeEnv,
			snoozedUntil:  todayISO,
			args:          []string{"customers", "list"},
			expectMessage: false,
		},
		{
			name:          "stale_snooze_writes",
			getEnv:        claudeEnv,
			snoozedUntil:  yesterdayISO,
			args:          []string{"customers", "list"},
			expectMessage: true,
		},
		{
			name:          "suppressed_agent_guidance",
			getEnv:        claudeEnv,
			snoozedUntil:  "",
			args:          []string{"agent-guidance", "snooze"},
			expectMessage: false,
		},
		{
			name:          "garbage_snooze_value_writes",
			getEnv:        claudeEnv,
			snoozedUntil:  "not-a-date",
			args:          []string{"customers", "list"},
			expectMessage: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			MaybeEmit(tc.getEnv, &buf, tc.snoozedUntil, today, tc.args)
			if tc.expectMessage {
				assert.Contains(t, buf.String(), "Stripe CLI Agent Guidance")
				assert.Contains(t, buf.String(), "stripe spec search")
				assert.Contains(t, buf.String(), "stripe agent-guidance snooze")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}
```

**Step 2: Run the test, verify it fails**

Run: `go test ./pkg/cmd/agentguidance/... -run TestMaybeEmit -v`
Expected: FAIL with `undefined: MaybeEmit`.

**Step 3: Implement `MaybeEmit` + helpers**

Write `pkg/cmd/agentguidance/agentguidance.go`:

```go
package agentguidance

import (
	"fmt"
	"io"
	"time"

	"github.com/stripe/stripe-cli/pkg/useragent"
)

// MaybeEmit writes the agent guidance message to w when all gates pass:
//   - an AI agent is detected via env vars
//   - the command path (args[0]) is not on the suppression list
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
	if len(args) == 0 {
		return true
	}
	switch args[0] {
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

// Ensure unused imports don't break the build during scaffolding.
var _ = strings.Builder{}
```

**Important:** delete the trailing `var _ = strings.Builder{}` line before committing — it's only there if you scaffold the test first. Confirm with `goimports` / `make fmt`. Final file should NOT import `strings`.

**Step 4: Run the test, verify all 6 cases pass**

Run: `go test ./pkg/cmd/agentguidance/... -v`
Expected: all `TestMaybeEmit/*` and `TestToday/*` subtests PASS.

**Step 5: Format and lint**

Run: `make fmt`
Run: `make lint` (this runs golangci-lint v2 across the whole repo; expect a few seconds)
Expected: both clean. If lint complains about the new package, fix and re-run.

**Step 6: Commit**

```bash
git add pkg/cmd/agentguidance/agentguidance.go pkg/cmd/agentguidance/agentguidance_test.go
git commit -m "Add MaybeEmit, SnoozeDate, suppression and snooze helpers"
```

---

### Task 4: Implement the `agent-guidance snooze` cobra command

**Files:**
- Create: `pkg/cmd/agent_guidance.go`
- Create: `pkg/cmd/agent_guidance_test.go`

**Step 1: Write failing tests**

Write `pkg/cmd/agent_guidance_test.go`:

```go
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// newTempConfig returns a *config.Config pointing at a temp file so tests
// don't touch the developer's real ~/.config/stripe/config.toml.
func newTempConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	require.NoError(t, os.WriteFile(path, []byte(""), 0600))

	// Reset viper so each test starts clean.
	viper.Reset()
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	require.NoError(t, viper.ReadInConfig())

	cfg := &config.Config{ProfilesFile: path}
	return cfg, path
}

func TestAgentGuidanceSnooze_HappyPath(t *testing.T) {
	cfg, path := newTempConfig(t)

	cmd := newAgentGuidanceCmd(cfg)
	snooze, _, err := cmd.Find([]string{"snooze"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	snooze.SetOut(&stdout)
	snooze.SetErr(&stdout)

	require.NoError(t, snooze.RunE(snooze, []string{}))

	assert.Contains(t, stdout.String(), "Agent guidance snoozed")

	// Read the file directly to assert what was actually written.
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	today := time.Now().Format("2006-01-02")
	assert.Contains(t, string(contents), fmt.Sprintf(`snoozed_until = "%s"`, today))
	assert.True(t, strings.Contains(string(contents), "[agent_guidance]") ||
		strings.Contains(string(contents), "agent_guidance.snoozed_until"))
}

func TestAgentGuidanceSnooze_WriteFailure(t *testing.T) {
	cfg := &config.Config{ProfilesFile: "/nonexistent/path/that/cannot/be/written/config.toml"}
	viper.Reset()
	viper.SetConfigFile(cfg.ProfilesFile)
	viper.SetConfigType("toml")

	cmd := newAgentGuidanceCmd(cfg)
	snooze, _, err := cmd.Find([]string{"snooze"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	snooze.SetOut(&stdout)
	snooze.SetErr(&stdout)

	err = snooze.RunE(snooze, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to snooze")
	// Sanity-check it's an error wrap, not a nil pointer panic
	assert.False(t, errors.Is(err, nil))
}
```

**Step 2: Run the tests, verify they fail**

Run: `go test ./pkg/cmd/... -run TestAgentGuidanceSnooze -v`
Expected: FAIL with `undefined: newAgentGuidanceCmd`.

**Step 3: Implement the command**

Write `pkg/cmd/agent_guidance.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/agentguidance"
	"github.com/stripe/stripe-cli/pkg/config"
)

func newAgentGuidanceCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-guidance",
		Short: "Manage Stripe CLI agent guidance",
		Long: "Manage the agent guidance interstitial that helps AI agents " +
			"discover the right CLI surface for a task (public API vs. the " +
			"dynamic-API spec plugin).",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "snooze",
		Short: "Snooze the agent guidance message for the rest of today",
		RunE: func(c *cobra.Command, args []string) error {
			today := agentguidance.Today()
			if err := cfg.WriteConfigField(
				"agent_guidance.snoozed_until",
				agentguidance.SnoozeDate(today),
			); err != nil {
				return fmt.Errorf("failed to snooze agent guidance: %w", err)
			}
			fmt.Fprintln(c.OutOrStdout(), "✔ Agent guidance snoozed for the rest of today.")
			return nil
		},
	})

	return cmd
}
```

**Step 4: Run the tests, verify they pass**

Run: `go test ./pkg/cmd/... -run TestAgentGuidanceSnooze -v`
Expected: both subtests PASS.

**Step 5: Format and lint**

Run: `make fmt`
Run: `make lint`

**Step 6: Commit**

```bash
git add pkg/cmd/agent_guidance.go pkg/cmd/agent_guidance_test.go
git commit -m "Add stripe agent-guidance snooze command"
```

---

### Task 5: Wire into `pkg/cmd/root.go::Execute()`

**Files:**
- Modify: `pkg/cmd/root.go` (two locations)

**Step 1: Add the import**

Open `pkg/cmd/root.go`. In the import block at the top (lines ~7–34), add:

```go
"github.com/stripe/stripe-cli/pkg/cmd/agentguidance"
```

(Sort with `goimports` / `make fmt` — it goes alphabetically with the other `github.com/stripe/stripe-cli/pkg/cmd/...` entries, after `github.com/stripe/stripe-cli/pkg/cmd/`.)

**Step 2: Call `MaybeEmit` from `Execute()`**

In `pkg/cmd/root.go::Execute()`, find this line (around line 120):

```go
emitClaudeCodePluginHint()
```

Add immediately after it:

```go
agentguidance.MaybeEmit(
	os.Getenv,
	os.Stderr,
	viper.GetString("agent_guidance.snoozed_until"),
	agentguidance.Today(),
	os.Args[1:],
)
```

**Step 3: Register the command**

In `pkg/cmd/root.go::init()`, find the block where commands are added (around line 240–255). Add this line near the other `rootCmd.AddCommand(...)` calls, in alphabetical position (after `newCommunityCmd`, before `newPluginCmd` works):

```go
rootCmd.AddCommand(newAgentGuidanceCmd(&Config))
```

**Step 4: Build and run all tests**

Run: `go build ./...`
Expected: exits 0.

Run: `make test`
Expected: full suite passes (existing + new tests).

**Step 5: Format and lint**

Run: `make fmt`
Run: `make lint`

**Step 6: Commit**

```bash
git add pkg/cmd/root.go
git commit -m "Wire agentguidance.MaybeEmit and agent-guidance command into root"
```

---

### Task 6: Add the E2E shell script

**Files:**
- Create: `scripts/test-agent-guidance.sh`

**Step 1: Write the script**

Write `scripts/test-agent-guidance.sh`:

```bash
#!/usr/bin/env bash
#
# End-to-end test for the agent-guidance interstitial. Builds the binary,
# points config at a tempdir, and runs five scenarios.
#
# Usage:  bash scripts/test-agent-guidance.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "Building stripe binary..."
make build > /dev/null
BIN="$(pwd)/stripe"

TMP_HOME="$(mktemp -d)"
trap 'rm -rf "$TMP_HOME"' EXIT
export XDG_CONFIG_HOME="$TMP_HOME/.config"
mkdir -p "$XDG_CONFIG_HOME/stripe"

# Prime an empty config file so viper has something to write to.
touch "$XDG_CONFIG_HOME/stripe/config.toml"
chmod 600 "$XDG_CONFIG_HOME/stripe/config.toml"

run_cmd() {
	# Run the binary with our temp config, capture stdout+stderr,
	# never fail the script if the command exits non-zero.
	"$BIN" "$@" 2>&1 || true
}

assert_contains() {
	local needle="$1"
	local haystack="$2"
	if ! echo "$haystack" | grep -q -- "$needle"; then
		echo "FAIL: expected to find '$needle' in:"
		echo "$haystack"
		exit 1
	fi
}

assert_not_contains() {
	local needle="$1"
	local haystack="$2"
	if echo "$haystack" | grep -q -- "$needle"; then
		echo "FAIL: did not expect to find '$needle' in:"
		echo "$haystack"
		exit 1
	fi
}

echo "=== Scenario 1: agent context, fresh config => message shows ==="
export CLAUDECODE=1
export STRIPE_AGENT_GUIDANCE_TODAY="2026-05-21"
OUT=$(run_cmd customers list)
assert_contains "Stripe CLI Agent Guidance" "$OUT"
echo "PASS"

echo "=== Scenario 2: snooze => silence on subsequent commands ==="
SNOOZE_OUT=$(run_cmd agent-guidance snooze)
assert_contains "Agent guidance snoozed" "$SNOOZE_OUT"
OUT=$(run_cmd customers list)
assert_not_contains "Stripe CLI Agent Guidance" "$OUT"
echo "PASS"

echo "=== Scenario 3: simulate next day => message shows again ==="
export STRIPE_AGENT_GUIDANCE_TODAY="2026-05-22"
OUT=$(run_cmd customers list)
assert_contains "Stripe CLI Agent Guidance" "$OUT"
echo "PASS"

echo "=== Scenario 4: human (no agent env) => silent ==="
unset CLAUDECODE
OUT=$(run_cmd customers list)
assert_not_contains "Stripe CLI Agent Guidance" "$OUT"
echo "PASS"

echo "=== Scenario 5: stripe spec invocation in agent context => silent ==="
export CLAUDECODE=1
export STRIPE_AGENT_GUIDANCE_TODAY="2026-05-23"  # fresh day, not snoozed
OUT=$(run_cmd spec --help)
assert_not_contains "Stripe CLI Agent Guidance" "$OUT"
echo "PASS"

echo
echo "All scenarios passed."
```

**Step 2: Make it executable**

```bash
chmod +x scripts/test-agent-guidance.sh
```

**Step 3: Run it**

Run: `bash scripts/test-agent-guidance.sh`
Expected: builds the binary, prints `PASS` after each scenario, ends with `All scenarios passed.`

If a scenario fails, the script exits with the failing output. Read the assert_* error message; common causes:
- Forgot to add the import in root.go (Task 5)
- Forgot to call `MaybeEmit` in `Execute()`
- Forgot to register `newAgentGuidanceCmd` in `init()`
- Suppression list in `agentguidance.go` doesn't include `spec` or `agent-guidance`

**Step 4: Commit**

```bash
git add scripts/test-agent-guidance.sh
git commit -m "Add E2E shell script for agent-guidance interstitial"
```

---

### Task 7: Final verification

**Step 1: Run the full test suite**

Run: `make test`
Expected: all tests pass, no new failures.

**Step 2: Run the linter**

Run: `make lint`
Expected: clean.

**Step 3: Run the E2E script one more time end-to-end**

Run: `bash scripts/test-agent-guidance.sh`
Expected: all five scenarios PASS.

**Step 4: Manual spot-check**

Build once: `make build`

Try these by hand:

```bash
# Real config (will modify ~/.config/stripe/config.toml — your choice)
CLAUDECODE=1 ./stripe customers list 2>/tmp/err.txt
cat /tmp/err.txt          # should contain the guidance message
./stripe agent-guidance snooze
grep -A1 agent_guidance ~/.config/stripe/config.toml   # should show today's date
```

After you're done, you can clear the snooze by hand if you want:

```bash
# either edit ~/.config/stripe/config.toml and remove the [agent_guidance] block,
# or just let it expire tomorrow
```

**Step 5: Diff review**

Run: `git log --oneline master..HEAD`
Expected: 6 commits (design doc + 5 implementation commits, depending on whether design doc was committed earlier).

Run: `git diff master --stat`
Sanity-check the file list:
- `docs/plans/2026-05-21-cli-agent-guidance-design.md` (new)
- `docs/plans/2026-05-21-cli-agent-guidance-implementation.md` (new — this file)
- `pkg/cmd/agentguidance/doc.go` (new)
- `pkg/cmd/agentguidance/today.go` (new)
- `pkg/cmd/agentguidance/today_test.go` (new)
- `pkg/cmd/agentguidance/agentguidance.go` (new)
- `pkg/cmd/agentguidance/agentguidance_test.go` (new)
- `pkg/cmd/agent_guidance.go` (new)
- `pkg/cmd/agent_guidance_test.go` (new)
- `pkg/cmd/root.go` (modified, ~10 lines added)
- `scripts/test-agent-guidance.sh` (new)

**Step 6: Done — ready for PR**

The branch `agent-guidance-interstitial` is ready. To open a PR (only when the user asks):

```bash
GH_HOST=github.com gh pr create --title "Add CLI agent guidance interstitial" --body "..."
```

---

## Notes for the implementing agent

- **Never skip `make lint` or `make fmt` after editing Go files.** The user's CLAUDE.md is explicit: lint failure after edits is a "critical error."
- **Don't push to remote** without explicit user request.
- **Don't widen the suppression list.** It's deliberate and matches the design doc. Adding to it requires going back to brainstorming.
- **Don't change the message text** without going back to design discussion. The exact wording was approved.
- **The `STRIPE_AGENT_GUIDANCE_TODAY` env var is intentional and undocumented user-facing.** Keep it that way; it's only for testing.
- **If `make test` fails on something unrelated** (a flaky existing test), don't try to fix it. Note it and move on; surface to the user.

## Skills referenced

- @superpowers:test-driven-development — every task follows TDD: failing test, run, implement, run, commit.
- @superpowers:verification-before-completion — run lint + tests before declaring a task done.
