package doctor

// Human renderers for the doctor report.

import (
	"fmt"
	"sort"
	"strings"
)

func renderDoctor(r *DoctorReport) {
	if r.Degraded != "" {
		fmt.Println(warnLine("scan-only: " + r.Degraded))
	} else {
		renderAccount(r.Account)
	}
	renderFindings(r)
	renderSignals(r)
}

func renderAccount(a AccountSummary) {
	{
		name := a.ID
		if a.Name != "" {
			name += " (" + a.Name + ")"
		}
		fmt.Println(titleStyle.Render("Account ") + name + mutedStyle.Render("  test mode"))
		var vs []string
		for v, n := range a.EventVersions {
			vs = append(vs, fmt.Sprintf("%s ×%d", v, n))
		}
		sort.Strings(vs)
		fmt.Println(kv("event API versions", strings.Join(vs, ", ")))
		verLine := fmt.Sprintf("all at/after %s", a.Cutoff)
		if !a.VersionsOK {
			verLine = failStyle.Render(fmt.Sprintf("traffic predates %s", a.Cutoff))
		}
		fmt.Println(kv("DPM version cutoff", verLine))
		cfg := fmt.Sprintf("%d configuration(s), %d methods on / %d off", a.Configs, a.MethodsOn, a.MethodsOff)
		if a.ActiveConfig != "" {
			cfg += mutedStyle.Render(" — active: " + a.ActiveConfig)
		}
		fmt.Println(kv("dashboard methods", cfg))
		if len(a.Unavailable) > 0 {
			fmt.Println(warnLine("toggled ON but NOT available (capability inactive — will not render): " + strings.Join(a.Unavailable, ", ")))
		}
		if a.Configs > 1 {
			fmt.Println(infoLine(fmt.Sprintf("%d active configurations — verdicts use the default; call sites passing payment_method_configuration explicitly are governed by that config instead", a.Configs)))
		}
	}
}

func renderFindings(r *DoctorReport) {
	if len(r.Findings) == 0 {
		fmt.Println("\n" + okLine(fmt.Sprintf("no findings — %d files scanned, %d parsed", r.Stats.FilesScanned, r.Stats.FilesParsed)))
	} else {
		// An account-level verdict (BLOCKED/CAUTION/UNKNOWN) is one fact, not
		// N facts: state it once as a headline and list findings compactly.
		shared := r.Findings[0].Verdict
		sharedClass := r.Findings[0].Class
		for _, f := range r.Findings {
			if f.Verdict != shared {
				shared = ""
				break
			}
		}
		accountLevel := shared != "" && (sharedClass == "BLOCKED" || sharedClass == "CAUTION" || sharedClass == "UNKNOWN")

		fmt.Printf("\n%s\n", titleStyle.Render(fmt.Sprintf("Findings (%d)", len(r.Findings))))
		if accountLevel {
			glyph := failLine
			if sharedClass != "BLOCKED" {
				glyph = warnLine
			}
			fmt.Println(glyph(titleStyle.Render("all findings: ") + shared))
			fmt.Println()
		}
		for _, f := range r.Findings {
			glyph := warnLine
			switch f.Class {
			case "CANDIDATE":
				glyph = okLine
			case "BLOCKED":
				glyph = failLine
			}
			fmt.Println(glyph(fmt.Sprintf("%s:%d  %s %s", f.File, f.Line, mutedStyle.Render("["+f.Intent+"]"), f.Value)))
			if !accountLevel {
				fmt.Println(kv("", f.Verdict))
			}
		}
		var sum []string
		for class, n := range r.Summary {
			sum = append(sum, fmt.Sprintf("%s ×%d", class, n))
		}
		sort.Strings(sum)
		fmt.Println("\n" + mutedStyle.Render("  "+strings.Join(sum, "  ")))
	}
}

func renderSignals(r *DoctorReport) {
	if r.WebhookHandlers != nil && len(r.WebhookHandlers.Events) > 0 {
		fmt.Println("\n" + titleStyle.Render("Expected webhook handlers in YOUR code"))
		for _, e := range r.WebhookHandlers.Events {
			if e.Present {
				fmt.Println(okLine(fmt.Sprintf("%-46s %s", e.Event, mutedStyle.Render(e.Files[0]))))
			} else {
				fmt.Println(warnLine(fmt.Sprintf("%-46s not found in scanned code", e.Event)))
			}
		}
	}
	if len(r.FrontendSignals) > 0 {
		fmt.Println("\n" + titleStyle.Render("Frontend warnings"))
		for _, w := range r.FrontendSignals {
			note := w.Note
			if note == "" {
				note = "legacy client-side token"
			}
			fmt.Println(failLine(fmt.Sprintf("%q in %s — %s", w.Signal, w.File, note)))
		}
	}
	if len(r.Triage) > 0 {
		fmt.Println("\n" + titleStyle.Render("Which migration applies"))
		for _, t := range r.Triage {
			fmt.Println(okLine(titleStyle.Render("detected: ") + t.Detected))
			fmt.Println(kv("", accentStyle.Render(t.Recommend)))
			max := len(t.Evidence)
			if max > 3 {
				max = 3
			}
			for _, ev := range t.Evidence[:max] {
				fmt.Println(kv("", mutedStyle.Render(fmt.Sprintf("%s  (%s)", ev.File, ev.Token))))
			}
		}
	}
	if len(r.ManifestChecks) > 0 {
		fmt.Println("\n" + titleStyle.Render("Package version floors"))
		for _, m := range r.ManifestChecks {
			switch {
			case m.Found == "":
				fmt.Println(infoLine(fmt.Sprintf("%-28s not in package.json (floor %s applies only if used)", m.Package, m.Floor)))
			case m.OK:
				fmt.Println(okLine(fmt.Sprintf("%-28s %s (floor %s)", m.Package, m.Found, m.Floor)))
			default:
				fmt.Println(failLine(fmt.Sprintf("%-28s %s is below required %s", m.Package, m.Found, m.Floor)))
			}
		}
	}

	if r.LiveDrill != nil {
		fmt.Println()
		renderDrill(r.LiveDrill)
	}
}

func renderDrill(r *DrillReport) {
	fmt.Println(titleStyle.Render("Live check: delayed-notification webhooks"))
	if r.ListenReady {
		fmt.Println(okLine("stripe listen ready (signing secret captured, not shown)"))
	}
	if r.Triggered != "" {
		fmt.Println(okLine("triggered " + r.Triggered))
	}
	for _, e := range r.Events {
		if e.Signature == "verified" {
			fmt.Println(okLine(fmt.Sprintf("received %-45s signature verified", e.Type)))
		} else {
			fmt.Println(failLine(fmt.Sprintf("received %-45s signature INVALID", e.Type)))
		}
	}
	if r.Verified {
		fmt.Println(okLine(titleStyle.Render("verified: the handler round-trips end to end")))
	} else {
		fmt.Println(warnLine("not fully verified" + mutedStyle.Render(" — "+r.Note)))
	}
}

// ---------- guide ----------
