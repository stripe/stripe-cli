package coopcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// coopTmuxSocket names the dedicated tmux server the launcher creates when the
// user isn't already inside tmux. A separate socket keeps co-op's server,
// session names, and shipped config away from the user's own tmux. It is never
// used from inside an existing tmux session: tmux only refuses to nest into
// the *same* server, so attaching a second socket from within a pane nests
// silently with stacked prefix keys.
const coopTmuxSocket = "stripe-coop"

// coopTmuxBaseConf works on every tmux version co-op supports. ASCII only: the
// status line is read back through the attaching client's locale, and without
// a UTF-8 LANG/LC_ALL non-ASCII characters silently degrade to underscores.
const coopTmuxBaseConf = `# Written by 'stripe coop start' on every launch; edits are overwritten.
set -g mouse on
set -g set-clipboard on
# Forward the active pane's title to the terminal tab; off by default in tmux.
set -g set-titles on
set -g set-titles-string "#T"
# Pass the review bell through to the terminal instead of drawing it in tmux.
set -g bell-action any
set -g visual-bell off
set -g default-terminal "tmux-256color"
set -g status-left " Stripe Co-op "
set -g status-left-length 20
set -g status-right " mouse: scroll+click | detach: C-b d | quit: q "
set -g status-right-length 60
`

// coopTmuxModernConf is appended for tmux >= 3.2, where these options exist.
const coopTmuxModernConf = `set -g allow-passthrough on
set -g focus-events on
set -g extended-keys on
set -as terminal-features ",*:RGB"
`

var tmuxVersionOutput = func() (string, error) {
	out, err := exec.Command("tmux", "-V").Output()
	return string(out), err
}

// tmuxVersionAtLeast parses `tmux -V` output like "tmux 3.7b" or "tmux 3.2a"
// and reports whether it is at least major.minor. Unparseable output (e.g.
// "tmux next-3.8", or the openbsd builds that print no number) returns false:
// the baseline config works everywhere, missing modern options only degrades.
func tmuxVersionAtLeast(versionOutput string, major, minor int) bool {
	fields := strings.Fields(strings.TrimSpace(versionOutput))
	if len(fields) < 2 {
		return false
	}
	numeric := strings.TrimRightFunc(fields[1], func(r rune) bool {
		return r < '0' || r > '9'
	})
	majorStr, minorStr, ok := strings.Cut(numeric, ".")
	if !ok {
		return false
	}
	gotMajor, err := strconv.Atoi(majorStr)
	if err != nil {
		return false
	}
	gotMinor, err := strconv.Atoi(minorStr)
	if err != nil {
		return false
	}
	return gotMajor > major || (gotMajor == major && gotMinor >= minor)
}

func coopTmuxConf() string {
	conf := coopTmuxBaseConf
	if out, err := tmuxVersionOutput(); err == nil && tmuxVersionAtLeast(out, 3, 2) {
		conf += coopTmuxModernConf
	}
	return conf
}

// writeCoopTmuxConf writes the shipped tmux config for the dedicated co-op
// server and returns its path. Rewritten on every launch so CLI upgrades take
// effect without a migration.
func writeCoopTmuxConf() (string, error) {
	folder := coopConfigFolder()
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", fmt.Errorf("creating co-op config folder: %w", err)
	}
	path := filepath.Join(folder, "coop.tmux.conf")
	if err := os.WriteFile(path, []byte(coopTmuxConf()), 0o644); err != nil {
		return "", fmt.Errorf("writing co-op tmux config: %w", err)
	}
	return path, nil
}
