package doctor

// Pack-declared code signals: expected webhook events, legacy frontend
// tokens, package version floors, and the which-migration triage.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackSignals are a pack's declared code signals: which webhook events the
// migrated integration must handle, which frontend tokens indicate
// from-state code, and which package version floors apply.
type PackSignals struct {
	WebhookEvents  []string
	FrontendTokens []FrontendToken
	ManifestFloors []ManifestFloor
}

type FrontendToken struct {
	Token string
	Note  string
}

type ManifestFloor struct {
	Package string
	Min     string // "8.0.0"
}

// TriageBranch describes one detectable integration from-state and which
// migration guide applies to it. Branches are evaluated independently — a
// repo can legitimately be in several states at once (that is the point of
// triage).
type TriageBranch struct {
	Detected  string   // what this evidence means
	Recommend string   // which topic/guide to follow, and why
	Tokens    []string // code tokens whose presence constitutes evidence
}

// scanTriage evaluates each branch's tokens over the tree and returns the
// branches with evidence, in declaration order.
func scanTriage(root string, branches []TriageBranch) []TriageResult {
	if len(branches) == 0 {
		return nil
	}
	results := make([]TriageResult, len(branches))
	for i, b := range branches {
		results[i] = TriageResult{Detected: b.Detected, Recommend: b.Recommend}
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := specs[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(src)
		for i, b := range branches {
			for _, t := range b.Tokens {
				if strings.Contains(text, t) {
					results[i].Evidence = append(results[i].Evidence, TriageEvidence{File: path, Token: t})
				}
			}
		}
		return nil
	})
	var out []TriageResult
	for _, r := range results {
		if len(r.Evidence) > 0 {
			out = append(out, r)
		}
	}
	return out
}

// scanSignals walks the scanned directory for the pack's declared signals.
// These are honest substring/manifest checks, reported as signals — never
// verdicts.
func scanSignals(root string, sig *PackSignals) (*HandlerSignals, []FrontendWarning, []ManifestCheck) {
	if sig == nil {
		return nil, nil, nil
	}
	h := &HandlerSignals{}
	for _, ev := range sig.WebhookEvents {
		h.Events = append(h.Events, EventSignal{Event: ev})
	}
	var fw []FrontendWarning
	var mc []ManifestCheck

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if base == "package.json" && len(sig.ManifestFloors) > 0 {
			mc = append(mc, checkManifest(path, sig.ManifestFloors)...)
			return nil
		}
		if _, ok := specs[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(src)
		for i := range h.Events {
			if strings.Contains(text, h.Events[i].Event) {
				h.Events[i].Files = append(h.Events[i].Files, path)
				h.Events[i].Present = true
			}
		}
		for _, t := range sig.FrontendTokens {
			if strings.Contains(text, t.Token) {
				fw = append(fw, FrontendWarning{File: path, Signal: t.Token, Note: t.Note})
				break
			}
		}
		return nil
	})
	h.AllPresent = len(h.Events) > 0
	for _, e := range h.Events {
		if !e.Present {
			h.AllPresent = false
		}
	}
	return h, fw, mc
}

// checkManifest compares declared dependency versions against pack floors.
// Absent packages are OK (the floor applies only when the package is used).
func checkManifest(path string, floors []ManifestFloor) []ManifestCheck {
	var out []ManifestCheck
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return out
	}
	deps := map[string]string{}
	for k, v := range pkg.Dependencies {
		deps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		deps[k] = v
	}
	for _, f := range floors {
		found, present := deps[f.Package]
		c := ManifestCheck{Package: f.Package, Floor: f.Min, Found: found, File: path, OK: true}
		if present {
			c.OK = versionAtLeast(found, f.Min)
		}
		out = append(out, c)
	}
	return out
}

// versionAtLeast does a lenient semver-ish compare: strips range prefixes and
// compares numeric dot segments. Unparseable versions count as OK (never
// false-alarm on what we can't read).
func versionAtLeast(have, want string) bool {
	clean := strings.TrimLeft(have, "^~>=v ")
	hp := strings.Split(clean, ".")
	wp := strings.Split(want, ".")
	for i := 0; i < len(wp); i++ {
		if i >= len(hp) {
			return false
		}
		var hn, wn int
		if _, err := fmt.Sscanf(strings.TrimSpace(hp[i]), "%d", &hn); err != nil {
			return true // unparseable -> no alarm
		}
		if _, err := fmt.Sscanf(wp[i], "%d", &wn); err != nil {
			return true
		}
		if hn != wn {
			return hn > wn
		}
	}
	return true
}
