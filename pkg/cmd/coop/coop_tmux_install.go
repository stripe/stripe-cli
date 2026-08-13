package coopcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

var (
	statFile = os.Stat

	// canOfferBrewInstall gates the interactive install offer to the one case
	// where installing needs no elevation and the user is present: Homebrew,
	// a real TTY, not CI, not a remote box someone else administers.
	canOfferBrewInstall = func() bool {
		if _, err := exec.LookPath("brew"); err != nil {
			return false
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return false
		}
		return os.Getenv("CI") == "" && os.Getenv("SSH_CONNECTION") == ""
	}

	runTmuxInstall = func() error {
		cmd := exec.Command("brew", "install", "tmux")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
)

// offerTmuxSetup handles a missing tmux. On macOS with Homebrew it offers to
// install interactively; everywhere else it prints the right command and moves
// on — never running a package manager itself, and never with elevation.
// Returns true when tmux is usable afterwards.
func (rc *coopRunCmd) offerTmuxSetup() bool {
	if runtime.GOOS == "windows" {
		// No native tmux exists; the manual-split flow is the product here.
		return false
	}
	if declinedTmuxInstall() {
		return false
	}

	if runtime.GOOS == "darwin" && canOfferBrewInstall() {
		var choice string
		if err := selectString("tmux gives co-op a side-by-side view (agent + review TUI). Install it?",
			[]huh.Option[string]{
				huh.NewOption("Yes — brew install tmux", "install"),
				huh.NewOption("Not now", "skip"),
				huh.NewOption("No, and don't ask again", "never"),
			},
			&choice,
		); err != nil {
			return false
		}
		switch choice {
		case "install":
			if err := runTmuxInstall(); err != nil {
				fmt.Printf("tmux install failed (%v) — continuing without it.\n", err)
				return false
			}
			return rc.hasTmux()
		case "never":
			persistTmuxInstallDecline()
		}
		return false
	}

	if hint := tmuxInstallHint(); hint != "" {
		fmt.Printf("Tip: install tmux for a side-by-side view: %s\n", hint)
	}
	return false
}

// tmuxInstallHint returns the install command for this environment, or "" when
// suggesting one would be wrong (nothing detected, or nothing to detect).
func tmuxInstallHint() string {
	if _, err := statFile("/.dockerenv"); err == nil || os.Getenv("container") != "" {
		// An in-container install evaporates on rebuild; fix the image instead.
		return "add tmux to your image, e.g. RUN apt-get update && apt-get install -y tmux"
	}
	if _, err := statFile("/etc/NIXOS"); err == nil {
		return "add tmux to your NixOS configuration (or nix profile install nixpkgs#tmux)"
	}
	sudo := "sudo "
	if os.Geteuid() == 0 {
		sudo = ""
	}
	managers := []struct{ bin, cmd string }{
		{"apt-get", sudo + "apt-get install -y tmux"},
		{"dnf", sudo + "dnf install -y tmux"},
		{"pacman", sudo + "pacman -S tmux"},
		{"apk", sudo + "apk add tmux"},
		{"zypper", sudo + "zypper install -y tmux"},
		{"brew", "brew install tmux"},
	}
	for _, m := range managers {
		if _, err := exec.LookPath(m.bin); err == nil {
			return m.cmd
		}
	}
	return ""
}

// nativeSplitHint names this terminal's own split keystroke so "open another
// terminal" costs one keypress instead of a context switch. Empty when the
// terminal is unknown.
func nativeSplitHint() string {
	if os.Getenv("WT_SESSION") != "" {
		return "Tip: Alt+Shift+D splits this Windows Terminal pane."
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "vscode":
		if runtime.GOOS == "darwin" {
			return `Tip: Cmd+\ splits the VS Code terminal.`
		}
		return "Tip: Ctrl+Shift+5 splits the VS Code terminal."
	case "iTerm.app":
		return "Tip: Cmd+D splits this iTerm2 window."
	case "ghostty":
		if runtime.GOOS == "darwin" {
			return "Tip: Cmd+D splits this Ghostty window."
		}
		return "Tip: Ctrl+Shift+O splits this Ghostty window."
	case "WarpTerminal":
		return "Tip: Cmd+D splits this Warp window."
	case "WezTerm":
		return `Tip: Ctrl+Alt+Shift+% splits this WezTerm window.`
	case "Apple_Terminal":
		return "Tip: Cmd+T opens a new Terminal tab."
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "Tip: Ctrl+Shift+Enter opens a new kitty window."
	}
	return ""
}

func tmuxDeclineMarkerPath() string {
	return filepath.Join(coopConfigFolder(), "tmux-install-declined")
}

func declinedTmuxInstall() bool {
	_, err := statFile(tmuxDeclineMarkerPath())
	return err == nil
}

func persistTmuxInstallDecline() {
	if err := os.MkdirAll(coopConfigFolder(), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(tmuxDeclineMarkerPath(), []byte("declined\n"), 0o644)
}
