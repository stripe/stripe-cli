# CLI Agent Guidance — Design

**Date:** 2026-05-21
**Status:** Approved, ready for implementation plan
**Author:** Anirudh Goyal
**Related:** [Dynamic APIs DEAR](https://docs.google.com/document/d/1jCuymvByQspsIXB4YKfia3fwF1gP2MyaP0Re852uwkc/edit), [Ephemeral APIs Product Shaping](https://docs.google.com/document/d/1yug4ljE5OzBbSTB5d5Pq903OSpPMQWSuD7SiPgZosvE/edit)

## Problem

Dynamic APIs expose ~2,000 dashboard-only endpoints to agents through a new `stripe spec` plugin (search → details → execute). Dogfooding shows agents don't discover the plugin on their own — they default to `stripe get`/`stripe post` from training data, even when the spec plugin is installed and the right tool exists. Two failure modes result: silent wrong answers from public-API calls, and incorrect dead ends ("not possible via the CLI"). Lower-capability models almost never reach the spec plugin without explicit human prompting.

## Goal

Add a CLI-level interstitial that fires on every Stripe CLI command in an AI-agent context, reminding the agent that multiple API surfaces exist and surfacing the `stripe spec` commands. The message can be snoozed for the rest of the day.

Non-goals: solving discovery for humans, per-agent-session snooze (deferred to a later iteration), telemetry on adoption (fast-follow PR).

## Decisions

| Question | Decision |
|---|---|
| Snooze granularity | **Per-machine-day.** Date stored in `~/.config/stripe/config.toml`. Per-agent-session deferred. |
| Trigger | **AI agents only**, gated on `useragent.DetectAIAgent()`. Humans never see it. |
| Snooze UX | **Plain non-interactive command:** `stripe agent-guidance snooze`. No prompts. |
| Local testing | **Full E2E on built binary** + Go unit tests. Hidden `STRIPE_AGENT_GUIDANCE_TODAY` env override lets us simulate "tomorrow." |
| Implementation shape | **Conservative.** Mirror existing `claudecodehint.go` prior art. Hook in `Execute()`, suppress on a small command list, nested config table. |

## Architecture

The change is contained in the `stripe-cli` Go repo. Nothing in `cli-spec-plugin` changes.

**New package:** `pkg/cmd/agentguidance/` — owns the message text, suppression logic, and date arithmetic. Pure functions, no cobra/viper dependency, trivially unit-testable.

**New command file:** `pkg/cmd/agent_guidance.go` — registers `stripe agent-guidance snooze`.

**Modified files:**
- `pkg/cmd/root.go` — one line in `Execute()` calling `agentguidance.MaybeEmit(...)` after `emitClaudeCodePluginHint()`. One line registering the command.
- No schema change to `Config` struct. Snooze field is read/written via existing `viper`-backed APIs.

**Config file impact:** A new top-level table appears in `~/.config/stripe/config.toml` only after first snooze:

```toml
[agent_guidance]
  snoozed_until = "2026-05-21"
```

No migration needed. Absent table → not snoozed.

## Components

### `agentguidance` package

```go
// MaybeEmit writes the agent guidance message to w when:
//   - DetectAIAgent(getEnv) returns non-empty
//   - args[0] is not in the suppression list
//   - snoozedUntil is not today's local ISO date
func MaybeEmit(getEnv func(string) string, w io.Writer, snoozedUntil string, today time.Time, args []string)

// SnoozeDate returns today's local date as ISO ("2026-05-21").
func SnoozeDate(today time.Time) string

// Today returns time.Now(), or a value parsed from STRIPE_AGENT_GUIDANCE_TODAY when set.
func Today() time.Time
```

Suppression list: `agent-guidance`, `spec`, `completion`, `version`, `--version`, `-v`, `help`, `--help`, `-h`, plus bare `stripe`. These are commands where the message is either redundant (the agent has already found `spec`) or off-topic (utility/metadata).

### Cobra command

```go
// pkg/cmd/agent_guidance.go
func newAgentGuidanceCmd(cfg *config.Config) *cobra.Command {
    cmd := &cobra.Command{Use: "agent-guidance", Short: "Manage Stripe CLI agent guidance"}
    cmd.AddCommand(&cobra.Command{
        Use:   "snooze",
        Short: "Snooze the agent guidance message for the rest of today",
        RunE: func(c *cobra.Command, args []string) error {
            today := agentguidance.Today()
            if err := cfg.WriteConfigField("agent_guidance.snoozed_until", agentguidance.SnoozeDate(today)); err != nil {
                return fmt.Errorf("failed to snooze agent guidance: %w", err)
            }
            fmt.Fprintln(c.OutOrStdout(), "✔ Agent guidance snoozed for the rest of today.")
            return nil
        },
    })
    return cmd
}
```

### Wiring in `pkg/cmd/root.go`

```go
// In Execute(), right after emitClaudeCodePluginHint():
agentguidance.MaybeEmit(
    os.Getenv,
    os.Stderr,
    viper.GetString("agent_guidance.snoozed_until"),
    agentguidance.Today(),
    os.Args[1:],
)

// Command registration:
rootCmd.AddCommand(newAgentGuidanceCmd(&Config))
```

## Data flow

**Normal command (agent, not snoozed):**
1. Agent invokes `stripe accounts retrieve`
2. `Execute()` calls `MaybeEmit`
3. All gates pass → message written to stderr
4. Cobra runs the command, JSON to stdout
5. Agent reads stderr (guidance) + stdout (data) separately

**Snooze:**
1. Agent invokes `stripe agent-guidance snooze`
2. `MaybeEmit` runs, hits `agent-guidance` in suppression list, no-ops
3. Snooze RunE writes `agent_guidance.snoozed_until = <today>` via `Config.WriteConfigField`
4. Prints `✔ Agent guidance snoozed for the rest of today.`

**Subsequent commands (same day):** `MaybeEmit` reads `snoozed_until == today`, no-ops.

**Next day:** Stored value `< today`, message shows again.

**Human (no agent env):** First gate fails, no-op. Humans never see the message.

**`stripe spec ...`:** Suppressed. Agent has reached the right tool already.

## Error handling

- **Emission path:** Best-effort, silent. `MaybeEmit` does no I/O beyond `fmt.Fprint`. Cannot fail in a way that should block the user's command.
- **Malformed `snoozed_until`:** String compare against today's ISO date — garbage values just fall through and message shows. No parsing, no error path.
- **Snooze write failure:** Surfaced as wrapped error from RunE. Cobra's existing error path prints to stderr and exits 1. No partial state — `WriteConfigField` is atomic.
- **Malformed `STRIPE_AGENT_GUIDANCE_TODAY`:** Falls back to `time.Now()`. Hidden env var — only set by people who know what they're doing.
- **Concurrent snoozes:** Both writes set the same value (today's ISO date). Idempotent. No locking needed.
- **Fresh install (no config.toml):** `viper.GetString` returns `""`, message shows. First snooze creates the file via existing `WriteConfigField` path.

Explicitly out of scope for v1: unsnoozing, multi-day snooze, per-agent-type snooze, telemetry.

## Testing

**Unit tests (`pkg/cmd/agentguidance/agentguidance_test.go`):**

Two table-driven tests:
- `TestMaybeEmit` with cases: `not_an_agent`, `agent_writes_message`, `snoozed_today_silent`, `stale_snooze_writes`, `suppressed_command`, `garbage_snooze_value`.
- `TestToday` with cases: `no_override`, `valid_override`, `malformed_override`.

**Cobra command tests (`pkg/cmd/agent_guidance_test.go`):**
- `TestAgentGuidanceSnooze_HappyPath` — asserts config write and stdout confirmation.
- `TestAgentGuidanceSnooze_WriteFailure` — asserts wrapped error on write failure.

**E2E shell script (`scripts/test-agent-guidance.sh`):**

Builds the binary, exercises five scenarios with `XDG_CONFIG_HOME` pointed at a tempdir:
1. Agent context, fresh config → message shows on `customers list`
2. Snooze → confirmation; subsequent `customers list` is silent
3. Override `STRIPE_AGENT_GUIDANCE_TODAY` to next day → message shows again
4. Unset agent env → human invocation, no message
5. `stripe spec --help` in agent context → no message (self-suppression)

**Manual checklist:** real `stripe accounts retrieve` against test-mode key with `CLAUDECODE=1`; verify stderr/stdout separation; verify `[agent_guidance]` table appears in real config after snooze; verify no regression on existing `claudecodehint`.

## Implementation order (suggested for stacked PRs)

This change is small enough for a single PR. Outline below for completeness:

1. **Single PR:** new `agentguidance` package + cobra command + wiring + tests + E2E script.

If we wanted to split it (probably overkill), a stacked split would be: (1) package + unit tests, (2) cobra command + tests, (3) wiring in root.go + E2E script. But the change is ~200 LoC total — one PR is appropriate.

## Open questions for follow-up PRs

- **Telemetry:** wire `guidance_displayed` and `guidance_snoozed` into the existing telemetry client to measure adoption. Fast-follow.
- **Per-agent-session snooze:** if the once-a-day snooze proves too coarse (fresh agents later in the day inherit the snooze and miss the guidance), implement process-tree fingerprinting. Defer until we have evidence.
- **Unsnooze command:** `stripe agent-guidance unsnooze` if anyone asks.
