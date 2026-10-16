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
	name := a.ID
	if a.Name != "" {
		name += " — " + a.Name
	}
	fmt.Println(titleStyle.Render("Account ") + name + mutedStyle.Render("  ·  test mode"))

	var vs []string
	for v, n := range a.EventVersions {
		vs = append(vs, fmt.Sprintf("%s (%d)", v, n))
	}
	sort.Strings(vs)
	fmt.Println(kv("API version, recent activity", strings.Join(vs, ", ")))
	fmt.Println(infoLine("sampled from the last 20 events on this account (Stripe's Events API — not webhooks)"))

	verLine := fmt.Sprintf("safe — recent traffic is already on API version %s or later", a.Cutoff)
	switch {
	case a.VersionsMixed:
		verLine = warnStyle.Render(fmt.Sprintf("risky — safe once ALL recent traffic is on API version %s or later; some requests are still older, so removing it now would silently drop payment methods on those unless automatic_payment_methods[enabled]=true is also added", a.Cutoff))
	case !a.VersionsOK:
		verLine = failStyle.Render(fmt.Sprintf("unsafe on its own — becomes safe once this account's traffic moves to API version %s or later; until then, removing it would silently drop payment methods unless automatic_payment_methods[enabled]=true is also added", a.Cutoff))
	}
	fmt.Println(kv("removing payment_method_types", verLine))

	total := a.MethodsOn + a.MethodsOff
	cfg := "no active payment-method configuration found in the Dashboard"
	if a.Configs > 0 {
		cfg = fmt.Sprintf("%d of %d payment methods turned on", a.MethodsOn, total)
		cfg += mutedStyle.Render(fmt.Sprintf(", in the %q configuration", a.ActiveConfig))
	}
	fmt.Println(kv("Dashboard payment methods", cfg))

	if len(a.Unavailable) > 0 {
		fmt.Println(warnLine("enabled in the Dashboard but not available on this account, so they will NOT render for customers: " + strings.Join(a.Unavailable, ", ")))
	}
	if a.Configs > 1 {
		fmt.Println(infoLine(fmt.Sprintf("%d payment-method configurations exist — this report follows the default one; code that passes payment_method_configuration explicitly follows that configuration instead", a.Configs)))
	}
}

// classLabel turns a verdict_class into a short, plain-English phrase for
// the human renderer. The JSON field keeps the original CAPS value.
func classLabel(class string) string {
	switch class {
	case "CANDIDATE":
		return "safe to remove"
	case "SKIP":
		return "skip — looks intentional"
	case "REVIEW":
		return "needs review"
	case "CAUTION":
		return "caution"
	case "BLOCKED":
		return "blocked (unsafe to just delete)"
	case "UNKNOWN":
		return "unknown"
	case "ADVISE":
		return "advisory"
	default:
		return strings.ToLower(class)
	}
}

// intentLabel turns a finding's intent bucket into a short, plain-English
// description of the hardcoded value, instead of the internal bucket name.
func intentLabel(intent string) string {
	switch intent {
	case "default-shaped":
		return "hardcoded to card/link only"
	case "deliberate":
		return "hardcoded to one payment method"
	case "static":
		return "hardcoded list of payment methods"
	case "dynamic":
		return "computed at runtime"
	default:
		return intent
	}
}

// withoutClassPrefix strips the "CLASS: " lead-in from a verdict string,
// since the renderer already shows the class via classLabel and a glyph.
func withoutClassPrefix(verdict string) string {
	if i := strings.Index(verdict, ": "); i > 0 {
		return verdict[i+2:]
	}
	return verdict
}

func renderFindings(r *DoctorReport) {
	if len(r.Findings) == 0 {
		fmt.Println("\n" + okLine(fmt.Sprintf("no findings — %d files scanned, %d parsed", r.Stats.FilesScanned, r.Stats.FilesParsed)))
	} else {
		// A single account-level judgment (BLOCKED/CAUTION/UNKNOWN) is one
		// fact, not N facts: state it once as a headline and list findings
		// compactly.
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
			fmt.Println(glyph(titleStyle.Render(classLabel(sharedClass)+": ") + withoutClassPrefix(shared)))
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
			fmt.Println(glyph(fmt.Sprintf("%s:%d  %s %s", f.File, f.Line, mutedStyle.Render("["+intentLabel(f.Intent)+"]"), f.Value)))
			if !accountLevel {
				fmt.Println(kv(classLabel(f.Class), withoutClassPrefix(f.Verdict)))
			}
		}
		var sum []string
		for class, n := range r.Summary {
			sum = append(sum, fmt.Sprintf("%d %s", n, classLabel(class)))
		}
		sort.Strings(sum)
		fmt.Println("\n" + mutedStyle.Render("  Summary: "+strings.Join(sum, ", ")))
	}
}

func renderSignals(r *DoctorReport) {
	if r.WebhookHandlers != nil && len(r.WebhookHandlers.Events) > 0 {
		fmt.Println("\n" + titleStyle.Render("Expected webhook handlers in your code"))
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
