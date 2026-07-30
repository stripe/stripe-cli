package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// detailSections are the tabs a step can offer. Checks used to be one of them
// and is gone: it restated the review prompts and commands the summary already
// shows, differing only in printing verification text untruncated. Tabs are
// offered only when the step has something to put in them, so an empty tab is
// never advertised.
var detailSections = []string{"Summary", "Files", "Reference"}

// nodeDetailSections is the per-task view, reached only when a selection is not
// resolvable to a step. It keeps its own Checks tab because there is no step
// summary alongside it to carry that content.
var nodeDetailSections = []string{"Summary", "Files", "Checks", "Reference"}

// stepDetailSections returns the tabs this step actually has content for.
func (m Model) stepDetailSections(ch *coop.SessionStep) []string {
	sections := []string{"Summary"}
	for _, node := range ch.Nodes {
		if node.Implementation != nil && node.Implementation.File != "" {
			sections = append(sections, "Files")
			break
		}
	}
	for _, node := range ch.Nodes {
		if (node.NodeType == coop.NodeAsyncHandler && len(node.Events()) > 0) ||
			(node.NodeType == coop.NodeAPIRequest && node.Request() != nil) {
			sections = append(sections, "Reference")
			break
		}
	}
	return sections
}

func (m Model) renderDetail() string {
	if m.session == nil {
		return ""
	}
	if m.selected.kind == navigationStep {
		return m.renderStepDetail(m.selected.stepIndex)
	}
	nodeIndex, ok := m.selectedNodeIndex()
	if !ok {
		return ""
	}
	node, err := m.session.NodeByNumber(nodeIndex + 1)
	if err != nil {
		return ""
	}

	w, innerW := m.detailWidths()

	var md strings.Builder
	section := nodeDetailSections[m.detailTab%len(nodeDetailSections)]
	currentSnippet := m.sdkSnippetNode == nodeIndex && m.sdkSnippet != ""

	switch section {
	case "Summary":
		m.writeSummaryDetail(&md, node)
		m.writeStepSDKSnippetDetail(&md, node, currentSnippet)
	case "Files":
		m.writeImplementationDetail(&md, node, false)
	case "Checks":
		m.writeReviewCommandDetail(&md, node)
		m.writeAsyncHandlerCheckDetail(&md, node)
		m.writeVerificationDetail(&md, node)
	case "Reference":
		m.writeSDKReferenceDetail(&md, node, currentSnippet)
		m.writeAsyncHandlerReferenceDetail(&md, node)
	}

	if node.State == coop.NodeSkipped && node.Activity != "" {
		md.WriteString("*Skipped: " + node.Activity + "*\n\n")
	}

	content := strings.TrimSpace(md.String())
	if content == "" {
		return ""
	}

	var parts []string
	if header := m.renderDetailTabs(nodeDetailSections, section, innerW); header != "" {
		parts = append(parts, header)
	}
	if content != "" {
		parts = append(parts, clampLines(m.renderMarkdown(content, innerW), innerW))
	}
	body := clampLines(strings.Join(parts, "\n"), innerW)
	box := m.theme.DetailBoxStyle.Width(w).Render(body)
	return indentBlock(box, detailIndent)
}

func (m Model) renderStepDetail(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	w, innerW := m.detailWidths()

	var md strings.Builder
	ch := &m.session.Steps[stepIndex]
	sections := m.stepDetailSections(ch)
	section := sections[m.detailTab%len(sections)]
	switch section {
	case "Summary":
		m.writeStepSummaryDetail(&md, ch, innerW)
	case "Files":
		m.writeStepFilesDetail(&md, ch)
	case "Reference":
		m.writeStepReferenceDetail(&md, ch)
	}

	content := strings.TrimSpace(md.String())
	suffix := ""
	if _, ok := m.selectedReviewTarget(); ok {
		if m.rejecting {
			// The editor lives here, not in the footer: in the split workspace
			// the footer card is suppressed, so this is the only place the user
			// can see what they are typing.
			m.rejectionInput.SetWidth(max(innerW-lipgloss.Width("Request changes: "), 8))
			inputView := m.rejectionInput.View()
			if m.rejectionInput.Value() == "" {
				inputView = m.theme.DimmedStyle.Render(m.rejectionInput.Placeholder)
			}
			suffix = "\n" + m.theme.SoftErrorStyle.Render("Request changes: ") + inputView
			if m.rejectionError != "" {
				suffix += "\n" + m.theme.SoftErrorStyle.Render(m.rejectionError)
			}
		} else {
			suffix = "\n" + m.theme.PromptStyle.Render("Waiting for you") +
				m.theme.MutedStyle.Render("   ") + m.keyHint(m.keys.Confirm.Help().Key, "confirm step") +
				m.theme.MutedStyle.Render(" · ") + m.keyHint(m.keys.Reject.Help().Key, "request changes")
		}
	}
	if content == "" && suffix == "" {
		return ""
	}

	var parts []string
	if header := m.renderDetailTabs(sections, section, innerW); header != "" {
		parts = append(parts, header)
	}
	if content != "" {
		if section == "Summary" {
			parts = append(parts, clampLines(content, innerW))
		} else {
			parts = append(parts, clampLines(m.renderMarkdown(content, innerW), innerW))
		}
	}
	if suffix != "" {
		parts = append(parts, suffix)
	}
	body := clampLines(strings.Join(parts, "\n"), innerW)
	box := m.theme.DetailBoxStyle.Width(w).Render(body)
	return indentBlock(box, detailIndent)
}

// renderDetailHeader draws the tab strip.
//
// The detail box has always had four sections cycled with tab, but nothing on
// screen said so: the section you land on rendered no header at all, and the
// others printed a bare word with no hint that it was one of several or that
// tab moved between them. Listing all four, with the active one marked, makes
// the control visible from the tab you start on.
// renderDetailTabs draws the section strip as actual tabs.
//
// They used to be words joined by " · ", which read as a sentence fragment
// rather than as a control — the tabs went unnoticed until someone pressed tab
// by accident. Filled backgrounds make the strip look like something you can
// move between: the active tab is a solid block in the selection color, the
// others sit on the panel ground.
func (m Model) renderDetailTabs(sections []string, section string, width int) string {
	// A lone tab is not a choice; do not spend a row advertising it.
	if len(sections) < 2 {
		return ""
	}

	var parts []string
	for _, name := range sections {
		if name == section {
			parts = append(parts, m.theme.TabActiveStyle.Render(name))
			continue
		}
		parts = append(parts, m.theme.TabInactiveStyle.Render(name))
	}
	strip := strings.Join(parts, m.theme.TabGapStyle.Render(" "))

	// Too narrow for the full strip — say where you are and that tab moves.
	if width > 0 && lipgloss.Width(ansi.Strip(strip)) > width {
		position := 1
		for i, name := range sections {
			if name == section {
				position = i + 1
			}
		}
		return m.theme.TabActiveStyle.Render(section) + m.theme.DimmedStyle.Render(
			fmt.Sprintf(" %d/%d · tab", position, len(sections)))
	}
	if width > 0 {
		strip += "\n" + m.theme.StepRuleStyle.Render(strings.Repeat("─", width))
	}
	return strip
}

// visibleDetailTabCount is how many tabs the card on screen is actually
// offering, so tab cycles through exactly those.
func (m Model) visibleDetailTabCount() int {
	if m.session == nil {
		return 0
	}
	if stepIndex, ok := m.selectedStepIndex(); ok {
		if stepIndex < 0 || stepIndex >= len(m.session.Steps) {
			return 0
		}
		return len(m.stepDetailSections(&m.session.Steps[stepIndex]))
	}
	return len(nodeDetailSections)
}

func (m Model) detailWidths() (int, int) {
	frameW, _ := m.theme.DetailBoxStyle.GetFrameSize()
	w := m.outlineRuleWidth()
	if w < 12 {
		w = 12
	}
	innerW := w - frameW
	if innerW < 8 {
		innerW = 8
	}
	return w, innerW
}

func indentBlock(s string, spaces int) string {
	if spaces <= 0 || s == "" {
		return s
	}
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func (m Model) detailLanguage() string {
	lang := m.session.Settings["language"]
	if lang == "" {
		lang = "javascript"
	}
	return lang
}

func (m Model) writeSummaryDetail(md *strings.Builder, node *coop.SessionNode) {
	description := node.DescriptionText()
	if description != "" {
		md.WriteString(description + "\n\n")
	}
	if node.ReviewPrompt != "" {
		md.WriteString("**To confirm:** " + node.ReviewPrompt + "\n\n")
	}
	if description == "" && node.ReviewPrompt == "" {
		md.WriteString("*No summary available for this task.*\n\n")
	}
}

func (m Model) writeStepSDKSnippetDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.NodeType != coop.NodeAPIRequest || node.Request() == nil {
		return
	}
	if currentSnippet {
		md.WriteString("**SDK example**\n")
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(m.sdkSnippet + "\n")
		md.WriteString("```\n\n")
		return
	}
	if nodeIndex, ok := m.selectedNodeIndex(); ok && m.sdkLoading && m.sdkLoadingNode == nodeIndex {
		md.WriteString("*Loading SDK example...*\n\n")
	}
}

// confirmationClause drops a prompt's opening instruction when the card has
// already printed the exact command.
//
// Blueprint prompts are written to cover every way a step might be exercised —
// "Run the relevant Stripe CLI trigger or complete the upstream flow, then
// confirm the handler receives the expected event." With the command spelled
// out directly above, that first clause is a vaguer restatement of it, and the
// part worth reading is what to look for afterwards.
func confirmationClause(prompt string, haveCommand bool) string {
	if !haveCommand {
		return prompt
	}
	idx := strings.Index(prompt, ", then ")
	if idx < 0 {
		return prompt
	}
	if !strings.Contains(strings.ToLower(prompt[:idx]), "run ") {
		return prompt
	}
	rest := strings.TrimSpace(prompt[idx+len(", then "):])
	if rest == "" {
		return prompt
	}
	return capitalize(rest)
}

// recoveryHint says what to do about a failed check. The reader is holding two
// options and the card never named either: fix it themselves, or hand it back.
func (m Model) recoveryHint(cause string) string {
	fix := "Fix it and re-run"
	if name := missingConfigPattern.FindStringSubmatch(cause); name != nil {
		for _, candidate := range name[1:] {
			if candidate != "" {
				fix = "Set " + candidate + " and re-run"
				break
			}
		}
	}
	return fix + ", or press " + m.keys.Reject.Help().Key + " to send this back to the agent."
}

// writeStepSummaryDetail renders the expanded step.
//
// It mirrors the review card deliberately: same headings, same hues, same
// flush-left layout. The two used to diverge — the card had structure while
// this view emitted unstyled headings, hardcoded indents, and every task's
// prose concatenated into one paragraph — so opening a step took you from a
// readable summary to a wall of text.
func (m Model) writeStepSummaryDetail(md *strings.Builder, ch *coop.SessionStep, width int) {
	wrapWidth := width - 2
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	if goal := strings.TrimSpace(ch.DescriptionText()); goal != "" {
		writeWrapped(md, m.theme.MutedStyle.Render("Goal ")+m.theme.MutedStyle.Render(goal), wrapWidth)
		md.WriteString("\n")
	}

	prompts := stepReviewPrompts(ch)
	target, hasTarget := m.selectedReviewTarget()
	instruction := ""
	if hasTarget {
		if command := m.reviewCommandLabel(target.nodeNumbers); command != "" {
			instruction = command
		}
	}

	// A step with nothing pending has nothing to confirm. Writing the heading
	// unconditionally left "To confirm" standing over an empty card the moment
	// arriving on a step opened it, which is when finished steps started being
	// opened at all.
	if instruction == "" && len(prompts) == 0 && !hasTarget {
		// Hanging indent, like the small card this expands from, so the same
		// sentence keeps the same shape whether it is trimmed or shown whole.
		if summary := m.stepCardSummary(m.selectedStepIndexOrZero()); summary != "" {
			for _, line := range strings.Split(wrapHanging(summary, wrapWidth, 2), "\n") {
				md.WriteString(line + "\n")
			}
			md.WriteString("\n")
		}
		return
	}

	md.WriteString(actionLabel(m.theme, "To confirm") + "\n")

	// The command comes first. The blueprint prose is written to cover every
	// way a step might be exercised — "run the relevant Stripe CLI trigger or
	// complete the upstream flow" — which tells a reader holding a terminal
	// nothing they can type. The exact command was already on the card, two
	// lines further down, phrased as an aside.
	command := instruction
	switch {
	case command != "":
		// Every command under the first, not under the "Run " label, so a step
		// with two triggers reads as two commands rather than as a sentence
		// that wrapped.
		for i, line := range strings.Split(command, "\n") {
			label := m.theme.MutedStyle.Render("Run ")
			if i > 0 {
				label = strings.Repeat(" ", lipgloss.Width("Run "))
			}
			writeWrapped(md, label+m.theme.BrandStyle.Render(line), wrapWidth)
		}
	case hasTarget:
		if venue := m.reviewVenueLabel(target.nodeNumbers); venue != "" {
			writeWrapped(md, m.theme.MutedStyle.Render("Check ")+venue, wrapWidth)
		}
	}
	// Each instruction is its own paragraph. Run together, two prompts read as
	// one wrapped sentence — the reader cannot see where the first thing to do
	// ends and the second begins. A blank line after the command block
	// separates what to type from what to look for once it has run.
	for i, prompt := range prompts {
		if i > 0 || command != "" {
			md.WriteString("\n")
		}
		writeWrapped(md, m.theme.InstructionStyle.Render(confirmationClause(prompt, command != "")), wrapWidth)
	}
	md.WriteString("\n")

	// What is blocking, and what to do about it. A count of failed checks says
	// something is wrong without saying whether the reader is meant to fix it,
	// re-run it, or hand it back — which was the one question the card never
	// answered.
	if failed, _ := stepCheckResults(ch); len(failed) > 0 {
		cause := stepBlockingCause(ch)
		headline := pluralChecks(len(failed)) + " could not finish"
		if cause != "" {
			headline += ": " + cause
		}
		// Hangs under its own glyph, like the small card's version of the same
		// sentence.
		failure := m.theme.SoftErrorStyle.Render("✗ ") +
			highlightIdentifiers(headline, m.theme.MutedStyle, m.theme.IdentifierStyle)
		for _, line := range strings.Split(wrapHanging(failure, wrapWidth, 2), "\n") {
			md.WriteString(line + "\n")
		}
		md.WriteString("\n")
		writeWrapped(md, m.theme.MutedStyle.Render(m.recoveryHint(cause)), wrapWidth)
		md.WriteString("\n")
	}

	// No task list and no checks section here. Tasks render outside the card,
	// under it, and each one carries its own check results — the two lists were
	// the same list, with the second copy restating every title from the first.
}

// writeWrapped wraps to the available width with no indentation: hierarchy in
// this view comes from weight and hue, as it does in the card.
func writeWrapped(md *strings.Builder, text string, width int) {
	for _, line := range strings.Split(wordWrap(text, width), "\n") {
		md.WriteString(line + "\n")
	}
}

// taskEvidence is the agent's own note about a task, condensed to one line.
// taskEvidence is the agent's own note about a task. It is stripped of CLI
// progress noise and flattened, but not truncated: the pane has vertical room,
// and a leading ellipsis on text that would have fit reads as damage. The
// character budget belongs to the card, which is height-constrained.
func stepReviewPrompts(ch *coop.SessionStep) []string {
	var prompts []string
	seen := map[string]bool{}
	for _, node := range ch.Nodes {
		if node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
			continue
		}
		seen[node.ReviewPrompt] = true
		prompts = append(prompts, node.ReviewPrompt)
	}
	return prompts
}

// stepCheckResults names failures and counts passes, as the card does.
func stepCheckResults(ch *coop.SessionStep) ([]string, int) {
	var failed []string
	passed := 0
	for _, node := range ch.Nodes {
		for _, verification := range node.Verifications {
			if verification.Passed {
				passed++
				continue
			}
			if check := summarizeCheckFailure(verification.Check, failedCheckBudget); check != "" {
				failed = append(failed, node.TitleText()+": "+check)
			}
		}
	}
	return failed, passed
}

func (m Model) writeStepFilesDetail(md *strings.Builder, ch *coop.SessionStep) {
	wrote := false
	for _, node := range ch.Nodes {
		if node.Implementation == nil || node.Implementation.File == "" {
			continue
		}
		md.WriteString("- `" + implementationFileLabel(node.Implementation) + "` — " + node.TitleText() + "\n")
		wrote = true
	}
	if wrote {
		md.WriteString("\n")
		return
	}
	md.WriteString("*No files reported for this step yet.*\n\n")
}

func (m Model) writeStepReferenceDetail(md *strings.Builder, ch *coop.SessionStep) {
	wrote := false
	for _, node := range ch.Nodes {
		events := node.Events()
		if node.NodeType == coop.NodeAsyncHandler && len(events) > 0 {
			md.WriteString("- `" + events[0].EventType + "` webhook trigger for " + node.TitleText() + "\n")
			wrote = true
		}
		if request := node.Request(); node.NodeType == coop.NodeAPIRequest && request != nil {
			md.WriteString("- `" + strings.ToUpper(request.Method) + " " + request.Path + "` for " + node.TitleText() + "\n")
			wrote = true
		}
	}
	// The long-form output behind each check. --detail exists so an agent can
	// keep its command logs and reasoning without drowning the card, and this
	// is where that text is read: the card shows the label, the Reference tab
	// shows what is behind it.
	for _, node := range ch.Nodes {
		for _, verification := range node.Verifications {
			if !verification.HasDetail() {
				continue
			}
			marker := "✗"
			if verification.Passed {
				marker = "✓"
			}
			// A heading and a paragraph, not a list item with an indented
			// body: the renderer strips the indent that would attach the two,
			// and pre-wrapping here fights the wrapping it does itself.
			md.WriteString("\n**" + marker + " " +
				summarizeCheckResult(verification.Check, 0, !verification.Passed) + "**\n\n")
			md.WriteString(strings.TrimSpace(verification.DetailText()) + "\n")
			wrote = true
		}
	}

	if wrote {
		md.WriteString("\n")
		return
	}
	md.WriteString("*No reference metadata for this step yet.*\n\n")
}

func (m Model) writeAsyncHandlerCheckDetail(md *strings.Builder, node *coop.SessionNode) {
	events := node.Events()
	if node.NodeType != coop.NodeAsyncHandler || len(events) == 0 {
		return
	}
	md.WriteString("**How to verify:**\n\n")
	md.WriteString("1. `stripe listen --forward-to localhost:<port>/webhook`\n")
	md.WriteString("2. Perform an API or Dashboard action that emits `" + events[0].EventType + "`\n")
	md.WriteString("3. Confirm your handler processes the event\n\n")
}

func (m Model) writeAsyncHandlerReferenceDetail(md *strings.Builder, node *coop.SessionNode) {
	events := node.Events()
	if node.NodeType != coop.NodeAsyncHandler || len(events) == 0 {
		return
	}
	md.WriteString("**Webhook event:**\n\n")
	md.WriteString("`" + events[0].EventType + "`\n\n")
}

func (m Model) writeReviewCommandDetail(md *strings.Builder, node *coop.SessionNode) {
	command := reviewCommandForNode(node)
	if command == "" {
		return
	}
	md.WriteString("**Review command:**\n\n")
	md.WriteString("`" + strings.ReplaceAll(command, "`", "'") + "`\n\n")
}

func (m Model) writeSDKReferenceDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.NodeType != coop.NodeAPIRequest {
		return
	}
	if currentSnippet {
		md.WriteString("**SDK example**\n")
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(m.sdkSnippet + "\n")
		md.WriteString("```\n\n")
		return
	}
	if nodeIndex, ok := m.selectedNodeIndex(); ok && m.sdkLoading && m.sdkLoadingNode == nodeIndex {
		md.WriteString("*Loading reference...*\n\n")
	}
}

func (m Model) writeImplementationDetail(md *strings.Builder, node *coop.SessionNode, currentSnippet bool) {
	if node.Implementation == nil {
		return
	}
	if currentSnippet {
		md.WriteString("---\n\n")
	}
	imp := node.Implementation
	md.WriteString("**Agent wrote:** `" + implementationFileLabel(imp) + "`\n\n")
	if imp.Snippet != "" {
		md.WriteString("```" + m.detailLanguage() + "\n")
		md.WriteString(imp.Snippet + "\n")
		md.WriteString("```\n\n")
	}
	if imp.Note != "" {
		md.WriteString("> " + imp.Note + "\n\n")
	}
}

func implementationFileLabel(imp *coop.Implementation) string {
	if imp.File == "" {
		return ""
	}
	if imp.Lines == "" {
		return imp.File
	}
	return imp.File + ":" + imp.Lines
}

func (m Model) writeVerificationDetail(md *strings.Builder, node *coop.SessionNode) {
	if len(node.Verifications) == 0 {
		return
	}
	for _, v := range node.Verifications {
		if v.Passed {
			md.WriteString("- ✓ " + v.Check + "\n")
		} else {
			md.WriteString("- ✗ " + v.Check + "\n")
		}
	}
	md.WriteString("\n")
}

// keyHint renders a binding the way the footer's help component does, so a key
// looks like a key wherever it appears.
func (m Model) keyHint(key, description string) string {
	return m.theme.KeyStyle.Render(key) + " " + m.theme.KeyDescriptionStyle.Render(description)
}
