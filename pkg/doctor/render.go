package doctor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/internal/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
)

func renderDoctor(r *DoctorReport) {
	if findings := actionFindings(r); len(findings) > 0 {
		renderActionPlan(r, findings)
	} else {
		renderRule(r)
		renderSummary(r)
		renderDetails(r)
	}
}

// ---------- action-plan renderer ----------

// actionFindings returns the declared remediation's text-eligible
// Findings. A Finding with Heading == "" opts out of the text renderer.
func actionFindings(r *DoctorReport) []remediation.Finding {
	if r.RemediationResult == nil {
		return nil
	}
	var out []remediation.Finding
	for _, f := range r.RemediationResult.Findings {
		if f.Heading != "" {
			out = append(out, f)
		}
	}
	return out
}

func renderActionPlan(r *DoctorReport, findings []remediation.Finding) {
	title := r.RemediationResult.Title
	if title == "" {
		title = strings.ToUpper(r.Topic)
	}
	fmt.Println(titleStyle.Render(title))
	fmt.Println()

	if r.Message != "" {
		for _, line := range wrapText(r.Message, 74) {
			fmt.Println(line)
		}
		fmt.Println()
	}

	if rows := r.RemediationResult.Summary; len(rows) > 0 {
		width := 0
		for _, row := range rows {
			if len(row.Category) > width {
				width = len(row.Category)
			}
		}
		width += 3
		for _, row := range rows {
			fmt.Printf("  %-*s%s\n", width, row.Category, row.Value)
		}
	}

	for i, f := range findings {
		fmt.Println()
		renderFinding(i+1, f)
	}

	if r.Docs != "" {
		fmt.Println()
		fmt.Println("Learn more:")
		fmt.Println(r.Docs)
	}
}

// renderFinding prints one numbered Finding. A Body paragraph containing
// a literal "\n" is printed as pre-broken lines instead of rewrapped —
// e.g. a lead-in sentence followed by a bare URL.
func renderFinding(num int, f remediation.Finding) {
	fmt.Printf("%d. %s\n", num, f.Heading)
	for _, p := range f.Body {
		fmt.Println()
		for _, seg := range strings.Split(p, "\n") {
			for _, line := range wrapText(seg, 73) {
				fmt.Println("   " + line)
			}
		}
	}
	if len(f.RequiresVerification) > 0 {
		fmt.Println()
		fmt.Println("   Before applying this, confirm:")
		for _, v := range f.RequiresVerification {
			for j, line := range wrapText(v, 69) {
				prefix := "     - "
				if j > 0 {
					prefix = "       "
				}
				fmt.Println(prefix + line)
			}
		}
	}
	for _, item := range f.Items {
		fmt.Println()
		fmt.Println("   " + item.Heading)
		for _, line := range wrapText(item.Body, 71) {
			fmt.Println("     " + line)
		}
	}
}

// wrapText greedily wraps s to width, splitting on whitespace.
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := []string{words[0]}
	for _, w := range words[1:] {
		last := lines[len(lines)-1]
		if len(last)+1+len(w) > width {
			lines = append(lines, w)
		} else {
			lines[len(lines)-1] = last + " " + w
		}
	}
	return lines
}

// renderRule prints the migration rule's own top-level fields.
func renderRule(r *DoctorReport) {
	if r.Message == "" && r.Action == "" && r.Docs == "" {
		return
	}
	fmt.Println(titleStyle.Render(strings.ToUpper(r.Topic)))
	if r.Message != "" {
		fmt.Println("  " + r.Message)
	}
	if r.Action != "" {
		fmt.Println(kv("Action", r.Action))
	}
	if r.Docs != "" {
		fmt.Println(kv("Docs", r.Docs))
	}
}

func renderSummary(r *DoctorReport) {
	fmt.Println()
	fmt.Println(titleStyle.Render("Summary"))
	fmt.Println(kv("Source code", sourceCodeSummary(r)))
	fmt.Println(kv("Dashboard", dashboardSummary(r)))
	fmt.Println(kv("Webhooks", webhookSummary(r)))
}

func sourceCodeSummary(r *DoctorReport) string {
	if len(r.Findings) == 0 {
		return fmt.Sprintf("no findings — %d files scanned, %d parsed", r.Stats.FilesScanned, r.Stats.FilesParsed)
	}
	var parts []string
	for class, n := range r.Summary {
		parts = append(parts, fmt.Sprintf("%d %s", n, classLabel(class)))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func dashboardSummary(r *DoctorReport) string {
	if r.RemediationResult == nil {
		return "no Dashboard-facing remediation declared"
	}
	return statusLabel(r.RemediationResult.Status)
}

// webhookSummary counts expected webhook handlers not found in the scanned code.
func webhookSummary(r *DoctorReport) string {
	events := webhookEvents(r)
	if len(events) == 0 {
		return "no webhook signals checked"
	}
	missing := 0
	for _, e := range events {
		if !e.Present {
			missing++
		}
	}
	if missing == 0 {
		return fmt.Sprintf("all %d expected handlers found in scanned code", len(events))
	}
	noun := "handlers"
	if missing == 1 {
		noun = "handler"
	}
	return fmt.Sprintf("%d expected %s not found in scanned code", missing, noun)
}

func webhookEvents(r *DoctorReport) []codesignals.EventSignal {
	if r.RemediationResult == nil {
		return nil
	}
	events, _ := r.RemediationResult.Evidence["webhook_events"].([]codesignals.EventSignal)
	return events
}

func statusLabel(s remediation.Status) string {
	switch s {
	case remediation.StatusReady:
		return "ready"
	case remediation.StatusCaution:
		return "caution"
	case remediation.StatusBlocked:
		return "blocked"
	case remediation.StatusReview:
		return "needs review"
	case remediation.StatusUnknown:
		return "unknown"
	default:
		return string(s)
	}
}

func renderDetails(r *DoctorReport) {
	fmt.Println()
	fmt.Println(titleStyle.Render("Details"))
	renderFindings(r)
	renderDashboardActions(r)
	renderWebhookHandlers(r)
}

func classLabel(class string) string {
	switch class {
	case "CANDIDATE":
		return "safe to remove"
	case "SKIP":
		return "skip — looks intentional"
	case "REVIEW":
		return "needs review"
	case "ADVISE":
		return "advisory"
	default:
		return strings.ToLower(class)
	}
}

// withoutClassPrefix strips the "CLASS: " lead-in from a verdict string.
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
		// An ADVISE rule's verdict is the same fact for every finding, not N
		// facts: state it once as a headline and list findings compactly.
		shared := r.Findings[0].Verdict
		sharedClass := r.Findings[0].Class
		for _, f := range r.Findings {
			if f.Verdict != shared {
				shared = ""
				break
			}
		}
		ruleLevel := shared != "" && sharedClass == "ADVISE"

		fmt.Printf("\n%s\n", titleStyle.Render(fmt.Sprintf("Findings (%d)", len(r.Findings))))
		if ruleLevel {
			fmt.Println(warnLine(titleStyle.Render(classLabel(sharedClass)+": ") + withoutClassPrefix(shared)))
			fmt.Println()
		}
		for i, f := range r.Findings {
			if i > 0 {
				fmt.Println()
			}
			glyph := warnLine
			if f.Class == "CANDIDATE" {
				glyph = okLine
			}
			fmt.Println(glyph(fmt.Sprintf("%s:%d  %s", f.File, f.Line, f.Value)))
			if !ruleLevel {
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

func renderDashboardActions(r *DoctorReport) {
	if r.RemediationResult == nil {
		return
	}
	res := r.RemediationResult
	fmt.Printf("\n%s\n", titleStyle.Render("Dashboard actions"))
	line := fmt.Sprintf("%s: %s", res.Topic, res.Message)
	switch res.Status {
	case remediation.StatusReady:
		fmt.Println(okLine(line))
	case remediation.StatusBlocked:
		fmt.Println(failLine(line))
	case remediation.StatusUnknown:
		fmt.Println(infoLine(line))
	default: // caution, review
		fmt.Println(warnLine(line))
	}
	// Notes are their own action items, shown separately from Status.
	if len(res.Notes) > 0 {
		fmt.Println()
		for _, n := range res.Notes {
			fmt.Println(infoLine(n))
		}
	}
}

func renderWebhookHandlers(r *DoctorReport) {
	events := webhookEvents(r)
	if len(events) == 0 {
		return
	}
	fmt.Printf("\n%s\n", titleStyle.Render("Expected webhook handlers in your code"))
	for i, e := range events {
		if i > 0 {
			fmt.Println()
		}
		switch {
		case e.Present:
			fmt.Println(okLine(fmt.Sprintf("%s: %s", e.Event, mutedStyle.Render(e.Files[0]))))
		case e.Note != "":
			fmt.Println(warnLine(fmt.Sprintf("%s: not found — %s", e.Event, e.Note)))
		default:
			fmt.Println(warnLine(fmt.Sprintf("%s: not found in scanned code", e.Event)))
		}
	}
}
