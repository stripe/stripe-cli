package doctor

// Human renderers for the doctor report, in three sections:
//
//  1. renderRule       — what the migration is and the recommended action.
//  2. renderSummary     — a glance at what needs to be done, bucketed into
//                         source code changes, manual Dashboard actions,
//                         and webhook changes.
//  3. renderDetails     — the same three buckets, in full: findings +
//                         verdicts, the Dashboard-facing remediation's
//                         message, and the webhook-handler breakdown.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/doctor/codesignals"
	"github.com/stripe/stripe-cli/pkg/doctor/remediation"
	"github.com/stripe/stripe-cli/pkg/doctor/webhookcheck"
)

func renderDoctor(r *DoctorReport) {
	if findings := actionFindings(r); len(findings) > 0 {
		renderActionPlan(r, findings)
	} else {
		renderRule(r)
		renderSummary(r)
		renderDetails(r)
	}
	if r.WebhookCheck != nil {
		fmt.Println()
		renderWebhookCheck(r.WebhookCheck)
	}
}

// ---------- action-plan renderer ----------
//
// When the declared remediation contributes Findings, doctor renders its
// numbered action-plan layout instead of the generic
// what-it-is/summary/details sections below: a plain-English title, the
// rule's own message, a short Code/Dashboard/Webhooks summary table, and
// each Finding as one numbered item. doctor's renderer treats every word
// inside a Finding opaquely — see remediation.Finding's doc comment; a
// topic that contributes none renders via the older generic layout
// instead.

// actionFindings returns the declared remediation's text-eligible
// Findings, if any — a Finding with Heading == "" is a remediation's way
// of opting a section out of the text renderer (e.g. no webhook signals
// were scanned at all) rather than contributing a blank numbered item.
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

	rows := [][2]string{
		{"Code", codeSummaryPhrase(r)},
		{summaryLabel(r), dashboardHeadline(r)},
		{"Webhooks", webhookSummary(r)},
	}
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	width += 3
	for _, row := range rows {
		fmt.Printf("  %-*s%s\n", width, row[0], row[1])
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

// renderFinding prints one numbered Finding: its heading, each Body
// paragraph (word-wrapped, except a paragraph containing a literal
// "\n", which is printed as pre-broken lines — e.g. a lead-in sentence
// followed by a bare URL), then any RequiresVerification checklist, then
// each Item as its own sub-heading plus an indented body line.
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

// summaryLabel is the action-plan table's second row label — the
// declared remediation's own SummaryLabel, falling back to "Dashboard"
// when unset.
func summaryLabel(r *DoctorReport) string {
	if r.RemediationResult != nil && r.RemediationResult.SummaryLabel != "" {
		return r.RemediationResult.SummaryLabel
	}
	return "Dashboard"
}

// codeSummaryPhrase is the action-plan table's "Code" row: the declared
// remediation's own CodeSummary when set, else a generic phrase — how
// many scanned locations need attention, and — when every one of them
// shares the same verdict class — what kind of attention.
func codeSummaryPhrase(r *DoctorReport) string {
	if r.RemediationResult != nil && r.RemediationResult.CodeSummary != "" {
		return r.RemediationResult.CodeSummary
	}
	if len(r.Findings) == 0 {
		return fmt.Sprintf("no findings — %d files scanned, %d parsed", r.Stats.FilesScanned, r.Stats.FilesParsed)
	}
	if len(r.Summary) == 1 {
		for class, n := range r.Summary {
			noun := "locations"
			if n == 1 {
				noun = "location"
			}
			return fmt.Sprintf("%d %s %s", n, noun, classAction(class))
		}
	}
	var parts []string
	for class, n := range r.Summary {
		parts = append(parts, fmt.Sprintf("%d %s", n, classLabel(class)))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// classAction is classLabel's sibling for the action-plan table's "Code"
// row — a short clause read as "N locations <classAction>", rather than
// a standalone noun phrase.
func classAction(class string) string {
	switch class {
	case "CANDIDATE":
		return "safe to remove"
	case "SKIP":
		return "can be skipped"
	case "REVIEW":
		return "need review"
	case "CAUTION":
		return "need an API-version-specific change"
	case "ADVISE":
		return "flagged for advisory review"
	default:
		return "need " + strings.ToLower(class)
	}
}

// dashboardHeadline is the action-plan table's "Dashboard" row: the
// declared remediation's own short Headline, falling back to its bare
// Status when unset.
func dashboardHeadline(r *DoctorReport) string {
	if r.RemediationResult == nil {
		return "no Dashboard-facing remediation declared"
	}
	if r.RemediationResult.Headline != "" {
		return r.RemediationResult.Headline
	}
	return statusLabel(r.RemediationResult.Status)
}

// ---------- section 1: what this is ----------

// renderRule prints the migration rule's own top-level fields (as fetched
// live from the doctor_rules API): what the migration is and the
// recommended action, ahead of everything else.
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

// ---------- section 2: what needs to be done ----------

// renderSummary prints a one-line-per-bucket summary of what needs to be
// done: source code changes, manual Dashboard actions, and webhook
// changes. renderDetails below carries the full version of each.
func renderSummary(r *DoctorReport) {
	fmt.Println()
	fmt.Println(titleStyle.Render("Summary"))
	fmt.Println(kv("Source code", sourceCodeSummary(r)))
	fmt.Println(kv("Dashboard", dashboardSummary(r)))
	fmt.Println(kv("Webhooks", webhookSummary(r)))
}

// sourceCodeSummary tallies findings by verdict class, same counts
// renderFindings' own trailing summary line shows in detail.
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

// dashboardSummary names the declared remediation's status — the
// Dashboard-facing action doctor computed for it (see renderDashboardActions
// for the full message).
func dashboardSummary(r *DoctorReport) string {
	if r.RemediationResult == nil {
		return "no Dashboard-facing remediation declared"
	}
	return statusLabel(r.RemediationResult.Status)
}

// webhookSummary counts how many of the expected webhook handlers were
// NOT found in the scanned code, across whichever remediation's Evidence
// carries that signal.
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

// webhookEvents returns the webhook-handler evidence the declared
// remediation carries, or nil if there is none.
func webhookEvents(r *DoctorReport) []codesignals.EventSignal {
	if r.RemediationResult == nil {
		return nil
	}
	events, _ := r.RemediationResult.Evidence["webhook_events"].([]codesignals.EventSignal)
	return events
}

// statusLabel turns a remediation Status into the same short, plain-English
// voice classLabel uses for verdict classes.
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

// ---------- section 3: the details ----------

// renderDetails prints the full version of each bucket renderSummary
// condensed: findings + verdicts, the Dashboard-facing remediation's full
// message, and the webhook-handler breakdown.
func renderDetails(r *DoctorReport) {
	fmt.Println()
	fmt.Println(titleStyle.Render("Details"))
	renderFindings(r)
	renderDashboardActions(r)
	renderWebhookHandlers(r)
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
	case "ADVISE":
		return "advisory"
	default:
		return strings.ToLower(class)
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

// renderFindings is the source-code-changes bucket, in detail: every
// finding with its file/line and the exact verdict message.
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

// renderDashboardActions is the manual-Dashboard-actions bucket, in
// detail: the declared remediation's full message from whatever
// pkg/doctor/remediation's registry computed for it (see runRemediation
// in cmd.go) — not just the status renderSummary shows.
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
	// Notes are their own action items, independent of Status — e.g. a
	// Dashboard deep link that always applies — so each gets its own line
	// rather than trailing onto the message above.
	if len(res.Notes) > 0 {
		fmt.Println()
		for _, n := range res.Notes {
			fmt.Println(infoLine(n))
		}
	}
}

// renderWebhookHandlers is the webhook-changes bucket, in detail: the same
// "Expected webhook handlers in your code" breakdown doctor's engine used
// to render generically — now sourced from whichever remediation's
// Evidence carries it (today: webhook_events, from the paymentmethods
// remediation's own codesignals scan).
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

func renderWebhookCheck(r *webhookcheck.Report) {
	fmt.Println(titleStyle.Render("Live check: delayed-notification webhooks"))
	if r.ListenReady {
		fmt.Println(okLine("stripe listen ready (signing secret captured, not shown)"))
	}
	if r.Triggered != "" {
		fmt.Println(okLine("triggered " + r.Triggered))
	}
	for _, e := range r.Events {
		if e.Signature == "verified" {
			fmt.Println(okLine(fmt.Sprintf("received %s: signature verified", e.Type)))
		} else {
			fmt.Println(failLine(fmt.Sprintf("received %s: signature INVALID", e.Type)))
		}
	}
	if r.Verified {
		fmt.Println(okLine(titleStyle.Render("verified: the handler round-trips end to end")))
	} else {
		fmt.Println(warnLine("not fully verified" + mutedStyle.Render(" — "+r.Note)))
	}
}

// ---------- guide ----------
