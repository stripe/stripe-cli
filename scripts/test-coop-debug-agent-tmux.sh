#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
tmp_dir="${TMPDIR:-/tmp}/coop-debug-agent-tmux-$$"
stripe_bin="$tmp_dir/stripe"
tmux_session=""
pass=0
fail=0
tui_pane=""
agent_pane=""

cleanup() {
  if [ -n "$tmux_session" ]; then
    tmux kill-session -t "$tmux_session" 2>/dev/null || true
  fi
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

run_smoke() {
  local name="$1" session_width="$2" min_tui_width="$3" max_tui_width="$4" expect_wide="$5"
  local xdg_config_home="$tmp_dir/xdg-$name"

  tmux_session="coop-debug-agent-test-$name-$$"
  tui_pane=""
  agent_pane=""
  mkdir -p "$xdg_config_home"

  echo "[debug-agent-tmux] launching $name coop start --debug-agent"
  tmux new-session -d -s "$tmux_session" -x "$session_width" -y 50 \
    "XDG_CONFIG_HOME='$xdg_config_home' '$stripe_bin' coop start one-time-payment --language=node --debug-agent; echo TUI_EXIT:\$?; sleep 60"
  tui_pane="$(tmux display-message -p -t "$tmux_session" '#{pane_id}')"

  if ! wait_for_tui "Stripe Co-op" 10; then
    agent_pane="$tui_pane"
    record_fail "$name TUI started"
  else
    record_pass "$name TUI started"
  fi

  agent_pane="$(tmux list-panes -t "$tmux_session" -F '#{pane_id} #{pane_width}' | sort -k2,2nr | sed -n '1s/ .*//p')"
  if [ "$agent_pane" = "$tui_pane" ]; then
    agent_pane="$(tmux list-panes -t "$tmux_session" -F '#{pane_id}' | sed -n '2p')"
  fi

  local tui_size tui_width tui_height
  tui_size="$(tmux display-message -p -t "$tui_pane" '#{pane_width}x#{pane_height}')"
  tui_width="${tui_size%x*}"
  tui_height="${tui_size#*x}"
  if [ "$tui_width" -ge "$min_tui_width" ] && [ "$tui_width" -le "$max_tui_width" ] && [ "$tui_height" = "50" ]; then
    record_pass "$name TUI pane width is expected ($tui_size)"
  else
    record_fail "$name unexpected TUI pane size $tui_size"
  fi

  if [ "$expect_wide" = "true" ]; then
    if wait_for_tui "Press enter to inspect this (step|section)" 10; then
      record_pass "$name split workspace prompt visible"
    else
      record_fail "$name split workspace prompt visible"
    fi
  fi

  if wait_for_tui "Review" 10; then
    record_pass "$name debug agent reaches first review"
  else
    record_fail "$name debug agent reaches first review"
  fi
  assert_tui_visible "Confirmation steps" "$name review acceptance check visible"
  assert_agent_visible "\\[debug-agent\\]" "$name debug agent logs visible"

  tmux send-keys -t "$tui_pane" "r"
  sleep 0.5
  tmux send-keys -t "$tui_pane" "Please tighten the debug checkout path"
  sleep 0.25
  tmux send-keys -t "$tui_pane" Enter
  if wait_for_tui "Review" 10; then
    record_pass "$name debug agent reruns after request changes"
  else
    record_fail "$name debug agent reruns after request changes"
  fi

  for _ in $(seq 1 20); do
    local pane
    pane="$(capture_tui)"
    if echo "$pane" | rg -q "Integration complete"; then
      break
    fi
    if echo "$pane" | rg -q "confirm all"; then
      tmux send-keys -t "$tui_pane" "c"
    elif echo "$pane" | rg -q "Waiting for you: review section"; then
      tmux send-keys -t "$tui_pane" "f"
    elif echo "$pane" | rg -q "Review"; then
      tmux send-keys -t "$tui_pane" "c"
    fi
    sleep 0.75
  done

  if wait_for_tui "Integration complete" 10; then
    record_pass "$name debug agent reaches completion view"
  else
    record_fail "$name debug agent reaches completion view"
  fi

  tmux kill-session -t "$tmux_session" 2>/dev/null || true
  tmux_session=""
}

run_smoke narrow 173 68 70 false
run_smoke wide 260 102 106 true

echo "[debug-agent-tmux] results: $pass passed, $fail failed"
if [ "$fail" -gt 0 ]; then
  exit 1
fi
