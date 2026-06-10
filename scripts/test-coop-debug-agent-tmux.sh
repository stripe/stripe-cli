#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
tmp_dir="${TMPDIR:-/tmp}/coop-debug-agent-tmux-$$"
stripe_bin="$tmp_dir/stripe"
tmux_session="coop-debug-agent-test-$$"
pass=0
fail=0
tui_pane=""
agent_pane=""

export XDG_CONFIG_HOME="$tmp_dir/xdg"

cleanup() {
  tmux kill-session -t "$tmux_session" 2>/dev/null || true
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

record_pass() {
  pass=$((pass + 1))
  echo "  ✓ $1"
}

record_fail() {
  fail=$((fail + 1))
  echo "  ✗ $1"
  echo "  ┌─ TUI capture"
  capture_tui | sed 's/^/  │ /'
  echo "  ├─ agent capture"
  capture_agent | sed 's/^/  │ /'
  echo "  └─────────"
}

capture_tui() {
  tmux capture-pane -t "$tui_pane" -p 2>/dev/null || true
}

capture_agent() {
  tmux capture-pane -t "$agent_pane" -p 2>/dev/null || true
}

wait_for_tui() {
  local pattern="$1" timeout="${2:-10}" i=0
  while [ "$i" -lt $((timeout * 4)) ]; do
    capture_tui | rg -q "$pattern" && return 0
    sleep 0.25
    i=$((i + 1))
  done
  return 1
}

assert_tui_visible() {
  local pattern="$1" label="$2"
  if capture_tui | rg -q "$pattern"; then
    record_pass "$label"
  else
    record_fail "$label (missing: $pattern)"
  fi
}

assert_agent_visible() {
  local pattern="$1" label="$2"
  if capture_agent | rg -q "$pattern"; then
    record_pass "$label"
  else
    record_fail "$label (agent missing: $pattern)"
  fi
}

echo "[debug-agent-tmux] building test stripe binary"
mkdir -p "$tmp_dir"
(cd "$repo_root" && go build -o "$stripe_bin" cmd/stripe/main.go)

echo "[debug-agent-tmux] launching coop start --debug-agent"
tmux new-session -d -s "$tmux_session" -x 173 -y 50 \
  "XDG_CONFIG_HOME='$XDG_CONFIG_HOME' '$stripe_bin' coop start one-time-payment --language=node --debug-agent; echo TUI_EXIT:\$?; sleep 60"
tui_pane="$(tmux display-message -p -t "$tmux_session" '#{pane_id}')"

if ! wait_for_tui "Stripe Co-op" 10; then
  agent_pane="$tui_pane"
  record_fail "TUI started"
else
  record_pass "TUI started"
fi

agent_pane="$(tmux list-panes -t "$tmux_session" -F '#{pane_id} #{pane_width}' | sort -k2,2nr | sed -n '1s/ .*//p')"
if [ "$agent_pane" = "$tui_pane" ]; then
  agent_pane="$(tmux list-panes -t "$tmux_session" -F '#{pane_id}' | sed -n '2p')"
fi

tui_size="$(tmux display-message -p -t "$tui_pane" '#{pane_width}x#{pane_height}')"
if [ "$tui_size" = "68x50" ] || [ "$tui_size" = "69x50" ] || [ "$tui_size" = "70x50" ]; then
  record_pass "TUI pane is approximately 40% ($tui_size)"
else
  record_fail "unexpected TUI pane size $tui_size"
fi

if wait_for_tui "Review" 10; then
  record_pass "debug agent reaches first review"
else
  record_fail "debug agent reaches first review"
fi
assert_tui_visible "Confirmation steps" "review acceptance check visible"
assert_agent_visible "\\[debug-agent\\]" "debug agent logs visible"

tmux send-keys -t "$tui_pane" "r"
sleep 0.5
tmux send-keys -t "$tui_pane" "Please tighten the debug checkout path"
sleep 0.25
tmux send-keys -t "$tui_pane" Enter
if wait_for_tui "Review" 10; then
  record_pass "debug agent reruns after request changes"
else
  record_fail "debug agent reruns after request changes"
fi

for _ in $(seq 1 12); do
  if capture_tui | rg -q "Integration complete"; then
    break
  fi
  if capture_tui | rg -q "Review"; then
    tmux send-keys -t "$tui_pane" "c"
  fi
  sleep 0.75
done

if wait_for_tui "Integration complete" 10; then
  record_pass "debug agent reaches completion view"
else
  record_fail "debug agent reaches completion view"
fi

echo "[debug-agent-tmux] results: $pass passed, $fail failed"
if [ "$fail" -gt 0 ]; then
  exit 1
fi
