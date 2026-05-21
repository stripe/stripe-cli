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
