package tui

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func (m Model) renderFooter() string {
	// Completion view has its own footer; don't render step footer.
	if m.session != nil && m.session.IsComplete() {
		return ""
	}

	var lines []string

	switch {
	case m.agentHeartbeatMissing:
		lines = append(lines, m.theme.ErrorStyle.Render("  Agent is not running. Restart it, or run: stripe coop status"))
	case m.agentIdle():
		lines = append(lines, m.theme.AttentionStyle.Render("  Waiting for agent: no recent updates. Reconnect: stripe coop status"))
	}

	if m.statusMessage != "" {
		// Wrap: a status can now carry a command for the developer to run, which
		// is longer than the short acknowledgements this line used to hold and
		// would otherwise overflow the narrow layout.
		for _, line := range strings.Split(wordWrap(m.statusMessage, m.width-2), "\n") {
			lines = append(lines, m.theme.AttentionStyle.Render("  "+line))
		}
	}

	// Name the step and offer a way to it. The note is pinned outside the
	// scrolling area, so it is the one guarantee that a pending review is
	// visible no matter where the user has scrolled to.
	if stepIndex, ok := m.firstActionableReviewStep(); ok {
		lines = append(lines, "", m.reviewWaitingNote(stepIndex))
	}

	h := m.help
	h.SetWidth(m.width - 2)
	h.ShortSeparator = " · "
	actionLine := m.theme.FooterStyle.MaxWidth(m.width).Render("  " + h.View(m))

	// The card renders inline, under its own step. The exception is a terminal
	// too short to show it there at all, where a one-line fallback is the only
	// thing standing between the user and a screen with nothing to act on.
	if _, ok := m.selectedReviewTarget(); ok && !m.useSplitWorkspace() {
		if selected, isStep := m.selectedStepIndex(); isStep && !m.stepShowsInlineCard(selected) {
			// Name a failure here too. At this size the fallback line is the
			// only row left over for the step's state, so a silent "enter for
			// details" is the difference between seeing a failed check and
			// not knowing one exists.
			failed := 0
			if target, hasTarget := m.selectedReviewTarget(); hasTarget {
				failed = m.reviewFailedCheckCount(target.nodeNumbers)
			}
			lines = append(lines, m.theme.MutedStyle.Render("  "+overflowHint(nil, failed)))
		}
	}
	lines = append(lines, actionLine)
	if budget := m.footerHeightBudget(); budget > 0 && lipgloss.Height(strings.Join(lines, "\n")) > budget {
		lines = append(lines[:max(len(lines)-2, 0)], actionLine)
	}

	return strings.Join(lines, "\n")
}

// reviewWaitingElsewhere reports whether a review is waiting on a step other
// than the one under the cursor — the case where jumping actually moves you.
func (m Model) reviewWaitingElsewhere() bool {
	step, ok := m.firstActionableReviewStep()
	if !ok {
		return false
	}
	selected, isStep := m.selectedStepIndex()
	return !isStep || selected != step
}

// reviewWaitingNote names the step waiting on the user. The key bar below
// carries the jump key, so the note carries no hint of its own. Step titles are
// blueprint-authored and can be long, so the title is truncated rather than
// letting the line run past the terminal.
func (m Model) reviewWaitingNote(stepIndex int) string {
	const prefix = "  Waiting for you: "
	title := m.session.Steps[stepIndex].TitleText()

	// No key hint here. It only ever applied when the cursor had moved away —
	// which is exactly when the key bar below already lists the same key — so
	// it put the identical hint on two adjacent lines.
	budget := m.width
	if budget <= 0 {
		budget = lipgloss.Width(prefix) + lipgloss.Width(title)
	}
	if available := budget - lipgloss.Width(prefix); available > 0 && lipgloss.Width(title) > available {
		title = ansi.Truncate(title, available, "…")
	}

	note := m.theme.SoftAttentionStyle.Render(prefix) + m.theme.PromptStyle.Render(title)
	// The failure count rides along here because this line is pinned. Inside
	// the card it can be clipped by a short viewport, and on the task rows it
	// can be scrolled past; the note is the one place it cannot be missed.
	if target, ok := m.selectedReviewTarget(); ok {
		if failed := m.reviewFailedCheckCount(target.nodeNumbers); failed > 0 {
			note += m.theme.MutedStyle.Render(" · ") +
				m.theme.SoftErrorStyle.Render(fmt.Sprintf("✗%d", failed))
		}
	}
	return note
}

func (m Model) renderReviewCard() string {
	return m.renderReviewCardWithMaxHeight(0)
}

// The hint on a card that could not show everything. Which one applies depends
// on whether the heading survived the clip: repeating "To confirm" directly
// under the heading of the same name read as a second section rather than as an
// overflow affordance, but when the card collapses entirely the hint is the only
// thing left naming the action, so there it has to say it.
const confirmHeading = "To confirm"

const (
	cardOverflowHint  = "▼ enter for details"
	cardCollapsedHint = "To confirm: enter for details"
)

// overflowHint picks between them by looking at what survived.
//
// A failure clipped out of the card is the one thing the hint must not stay
// quiet about: the card's rule is that failures are named, and a reviewer who
// sees a bare "enter for details" has no reason to think anything is wrong.
func overflowHint(kept []string, failed int) string {
	if failed > 0 && !containsFailureLine(kept) {
		return fmt.Sprintf("▼ %s failed — enter for details", pluralChecks(failed))
	}
	for _, line := range kept {
		if strings.Contains(ansi.Strip(line), confirmHeading) {
			return cardOverflowHint
		}
	}
	return cardCollapsedHint
}

func containsFailureLine(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(ansi.Strip(line)), "✗") {
			return true
		}
	}
	return false
}

// markClippedLine marks the last surviving line when content was dropped below
// it. Without this a check clipped mid-wrap — "creates a new price on every
// request instead of reusing" — reads as a finished sentence rather than a
// truncated one.
func markClippedLine(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	last := len(lines) - 1
	if strings.TrimSpace(ansi.Strip(lines[last])) == "" {
		return lines
	}
	lines[last] += "…"
	return lines
}

// clipCardContent cuts the card's body to the rows available, leaving a hint in
// place of what it dropped.
func (m Model) clipCardContent(wrapped []string, maxContentLines, failed int) []string {
	if len(wrapped) <= maxContentLines {
		return wrapped
	}
	switch {
	case maxContentLines < 1:
		return nil
	case maxContentLines == 1:
		// One line left: spend it on a failure if there is one, since "Review
		// step" tells the reviewer nothing they did not already know.
		if failed > 0 {
			return []string{m.theme.MutedStyle.Render(overflowHint(nil, failed))}
		}
		return []string{m.theme.DimmedStyle.Render("Review step")}
	default:
		// Keep naming the action even when the card collapses, so a narrow
		// terminal still says there is something to do.
		kept := trimDanglingSection(wrapped[:maxContentLines-1])
		more := m.theme.MutedStyle.Render(overflowHint(kept, failed))
		return append(markClippedLine(kept), more)
	}
}

func (m Model) renderReviewCardWithMaxHeight(maxHeight int) string {
	target, ok := m.selectedReviewTarget()
	if !ok {
		return ""
	}
	// Too short for the bordered card, which costs two rows of frame. Fall back
	// to a borderless line so the reviewer still learns there is something
	// waiting, rather than the card vanishing outright. Never do this while the
	// user is typing feedback — the input has to stay on screen.
	if maxHeight > 0 && maxHeight < 3 && !m.rejecting {
		if maxHeight < 1 {
			return ""
		}
		return m.theme.MutedStyle.Render("  " + overflowHint(nil, m.reviewFailedCheckCount(target.nodeNumbers)))
	}
	w, _ := m.reviewCardWidths()

	var lines []cardLine
	lines = append(lines, cardLine{text: m.theme.ReviewStyle.Render("Review step")})

	// The blueprint's own description says what the step is for. It is phrased
	// as an imperative, exactly like the review prompt below it, so without a
	// label the reviewer reads two commands and cannot tell which one to act
	// on. "Goal:" marks it as context.
	if goal := m.reviewGoalLabel(target.stepIndex); goal != "" {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Goal ") + m.theme.MutedStyle.Render(goal),
		})
	}

	// Feedback the user already gave outranks a fresh instruction: the agent is
	// reworking, so there is nothing to confirm yet.
	if note := m.requestedChangeLabel(target.nodeNumbers); note != "" {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: feedbackLabel(m.theme, "Requested change")})
		lines = append(lines, cardLine{text: note})
	}

	// What the reviewer should do comes next: the blueprint's own review prompt
	// is written for a human, unlike the agent's verification notes.
	prompts := m.reviewPromptLabels(target.nodeNumbers)
	if len(prompts) > 0 {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: actionLabel(m.theme, confirmHeading)})
		for _, prompt := range prompts {
			if len(prompts) > 1 {
				lines = append(lines, cardLine{text: m.theme.BrandStyle.Render("• ") + m.theme.InstructionStyle.Render(prompt)})
				continue
			}
			lines = append(lines, cardLine{text: m.theme.InstructionStyle.Render(prompt)})
		}
	}
	// Say where to go. A review command already answers that, so only fall back
	// to the node type when there isn't one.
	if command := m.reviewCommandLabel(target.nodeNumbers); command != "" {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Run ") + m.theme.BrandStyle.Render(command),
		})
	} else if venue := m.reviewVenueLabel(target.nodeNumbers); venue != "" {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Check ") + venue,
		})
	}

	// Failures are named; passes collapse to a count. A reviewer needs to know
	// which check failed far more than which ones succeeded.
	failed := m.reviewFailedCheckLabels(target.nodeNumbers)
	passed := m.reviewPassedCheckCount(target.nodeNumbers)
	if len(failed) > 0 || passed > 0 {
		lines = append(lines, cardLine{})
		lines = append(lines, cardLine{text: checksLabel(m.theme, len(failed) > 0)})
	}
	for _, label := range failed {
		lines = append(lines, cardLine{
			text: m.theme.SoftErrorStyle.Render("✗ ") +
				highlightIdentifiers(label, m.theme.MutedStyle, m.theme.IdentifierStyle),
		})
	}
	if passed > 0 {
		lines = append(lines, cardLine{
			text: m.theme.SoftSuccessStyle.Render("✓ ") + m.theme.MutedStyle.Render(fmt.Sprintf("%s passed", pluralChecks(passed))),
		})
	}

	metadataStart := len(lines)
	// A single-task step already names itself in the outline, so only spell out
	// the covered tasks when the step actually groups several.
	if len(target.nodeNumbers) > 1 {
		if included := m.reviewNodeTitleLabel(target.nodeNumbers); included != "" {
			lines = append(lines, cardLine{
				text: m.theme.MutedStyle.Render("Includes ") + m.theme.DimmedStyle.Render(included),
			})
		}
	}
	if changed := m.reviewChangedLabel(target.nodeNumbers); changed != "" {
		lines = append(lines, cardLine{
			text: m.theme.MutedStyle.Render("Changed ") + m.theme.FileAnnotationStyle.Render(changed),
		})
	}
	if len(lines) > metadataStart {
		lines = append(lines[:metadataStart], append([]cardLine{{}}, lines[metadataStart:]...)...)
	}
	// The feedback editor is pinned: it is appended last, so plain tail
	// truncation would cut the very thing the user is typing into. Wrap it
	// separately and always keep it, trimming the body above it instead.
	var pinned []cardLine
	if m.rejecting {
		m.rejectionInput.SetWidth(m.requestChangesInputWidth())
		inputView := m.rejectionInput.View()
		if m.rejectionInput.Value() == "" {
			inputView = m.theme.DimmedStyle.Render(m.rejectionInput.Placeholder)
		}
		pinned = append(pinned, cardLine{text: m.theme.ErrorStyle.Render("Request changes: ") + inputView})
		if m.rejectionError != "" {
			pinned = append(pinned, cardLine{text: m.theme.ErrorStyle.Render(m.rejectionError)})
		}
	}

	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, line.wrap(w-4)...)
	}
	var wrappedPinned []string
	for _, line := range pinned {
		wrappedPinned = append(wrappedPinned, line.wrap(w-4)...)
	}

	if maxHeight > 0 {
		wrapped = m.clipCardContent(wrapped, maxHeight-2-len(wrappedPinned), len(failed))
	}
	return m.renderReviewCardLines(w, maxHeight, len(failed), append(wrapped, wrappedPinned...))
}

// cardLine is one logical line of the review card. indent is applied to
// continuation lines so wrapped text stays visually attached to its label
// instead of reading as a new item.
type cardLine struct {
	text   string
	indent string
}

func (l cardLine) wrap(width int) []string {
	var out []string
	available := width - lipgloss.Width(l.indent)
	for _, segment := range strings.Split(l.text, "\n") {
		for i, piece := range strings.Split(wordWrap(segment, available), "\n") {
			if i > 0 {
				piece = l.indent + piece
			}
			out = append(out, piece)
		}
	}
	return out
}

func (m Model) reviewCardWidths() (int, int) {
	w := min(m.contentWidth()-2, 84)
	if w < 20 {
		w = m.contentWidth() - 2
	}
	frameW, _ := m.theme.ReviewCardStyle.GetFrameSize()
	innerW := w - frameW
	if innerW < 8 {
		innerW = 8
	}
	return w, innerW
}

func (m Model) requestChangesInputWidth() int {
	_, innerW := m.reviewCardWidths()
	width := innerW - lipgloss.Width("Request changes: ")
	if width < 8 {
		return 8
	}
	return width
}

func (m Model) renderReviewCardLines(width, maxHeight, failed int, lines []string) string {
	style := m.theme.ReviewCardStyle.Width(width).MaxWidth(width + 4)
	for {
		rendered := style.Render(strings.Join(lines, "\n"))
		if maxHeight <= 0 || lipgloss.Height(rendered) <= maxHeight {
			return rendered
		}
		if len(lines) <= 2 {
			return style.MaxHeight(maxHeight).Render(strings.Join(lines, "\n"))
		}
		// Trim here too, not only at the caller: this loop shrinks the card
		// until it fits, and without the trim it could stop just below a
		// heading — leaving "Agent checks" over a hint, with the failure it was
		// introducing dropped entirely.
		kept := trimDanglingSection(lines[:len(lines)-2])
		if len(kept) == 0 {
			return style.MaxHeight(maxHeight).Render(strings.Join(lines, "\n"))
		}
		lines = append(markClippedLine(kept), m.theme.MutedStyle.Render(overflowHint(kept, failed)))
	}
}

func (m Model) requestChangesPlaceholder(_ reviewTarget) string {
	return "Describe what should change in this step"
}

func (m Model) reviewChangedLabel(nodeNumbers []int) string {
	_, innerW := m.reviewCardWidths()
	var labels []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.Implementation == nil || node.Implementation.File == "" {
			continue
		}
		label := truncatePath(implementationFileLabel(node.Implementation), innerW)
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return ""
	}
	if len(labels) > 3 {
		return strings.Join(labels[:3], ", ") + fmt.Sprintf(" +%d more", len(labels)-3)
	}
	return strings.Join(labels, ", ")
}

func (m Model) reviewNodeTitleLabel(nodeNumbers []int) string {
	if m.session == nil {
		return ""
	}
	var titles []string
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.TitleText() == "" {
			continue
		}
		titles = append(titles, node.TitleText())
	}
	if len(titles) == 0 {
		return ""
	}
	if len(titles) > 3 {
		return strings.Join(titles[:3], ", ") + fmt.Sprintf(" +%d more", len(titles)-3)
	}
	return strings.Join(titles, ", ")
}

// reviewPromptLabel returns the instruction for the reviewer. Blueprint review
// prompts are authored for a human and say what to go and check; the agent's
// verification notes are free-form prose describing what it already did, so
// they belong in the detail view rather than in the card.
// sectionNames are the box's headings, used to avoid truncating a card so that
// a heading is left with nothing under it.
var sectionNames = map[string]bool{
	confirmHeading: true,
	"Agent checks": true,

	"Requested change": true,
}

// trimDanglingSection drops trailing blank lines and any heading left with no
// content beneath it, so a truncated card never ends on an empty section.
func trimDanglingSection(lines []string) []string {
	for len(lines) > 0 {
		last := strings.TrimSpace(ansi.Strip(lines[len(lines)-1]))
		if last != "" && !sectionNames[last] {
			break
		}
		lines = lines[:len(lines)-1]
	}
	return lines
}

// Section headings carry hue as well as weight. The layout is flush left, so
// color and weight are the only things left to separate a heading from the
// sentence under it — and the headings do not all mean the same thing, so they
// do not all look the same:
//
//	Do this           the action, in the same orange as "needs you"
//	Checks            evidence, in the review purple
//	Requested change  feedback already given, in the error hue
//
// All three stay bold, so the structure survives a terminal without color.
func actionLabel(t Theme, text string) string {
	return t.ActionHeadingStyle.Render(text)
}

// evidenceLabel colors the Checks heading by its own outcome, so the heading
// carries information rather than just naming the section: red when something
// failed, green when everything passed.
func evidenceLabel(t Theme, text string) string {
	return t.ReviewStyle.Bold(true).Render(text)
}

// checksLabel names whose checks these are — the agent's, not the reader's
// task — and colors the heading by its own outcome.
func checksLabel(t Theme, failed bool) string {
	if failed {
		return t.SoftErrorStyle.Render("Agent checks")
	}
	return t.SoftSuccessStyle.Render("Agent checks")
}

func feedbackLabel(t Theme, text string) string {
	return t.ErrorStyle.Bold(true).Render(text)
}

// reviewGoalLabel is the blueprint's own statement of what the step is for.
func (m Model) reviewGoalLabel(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	return strings.TrimSpace(m.session.Steps[stepIndex].DescriptionText())
}

// requestedChangeLabel surfaces feedback the user already sent. It was stored
// on the node and never rendered anywhere, so by the time the agent came back
// the user had no reminder of what they had asked for.
func (m Model) requestedChangeLabel(nodeNumbers []int) string {
	var notes []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		note := strings.TrimSpace(node.RejectionNote)
		if note == "" || seen[note] {
			continue
		}
		seen[note] = true
		notes = append(notes, note)
	}
	return strings.Join(notes, " ")
}

func (m Model) reviewPromptLabels(nodeNumbers []int) []string {
	if prompts := m.reviewBlueprintConfirmationPrompts(nodeNumbers); len(prompts) > 0 {
		return prompts
	}
	return []string{"Confirm the completed work matches this step and its verification evidence."}
}

// reviewFailedCheckLabels names the checks the agent could not get passing.
// failedCheckBudget keeps a named failure to roughly two wrapped lines in the
// card. The full text remains available in the detail view.
const failedCheckBudget = 160

// reviewFailedCheckCount counts the checks that did not pass.
//
// Not len(reviewFailedCheckLabels): that list collapses duplicate lines so the
// card does not print the same finding twice, and once identical root causes
// started normalizing to identical text it began collapsing genuinely separate
// checks as well. The label list is for display; this is for counting.
func (m Model) reviewFailedCheckCount(nodeNumbers []int) int {
	count := 0
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		for _, verification := range node.Verifications {
			if !verification.Passed {
				count++
			}
		}
	}
	return count
}

func (m Model) reviewFailedCheckLabels(nodeNumbers []int) []string {
	var checks []string
	seen := map[string]bool{}
	showStepTitle := len(nodeNumbers) > 1
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		for _, verification := range node.Verifications {
			check := summarizeCheckFailure(verification.Check, failedCheckBudget)
			if verification.Passed || check == "" || seen[check] {
				continue
			}
			seen[check] = true
			if showStepTitle && node.TitleText() != "" {
				check = node.TitleText() + ": " + check
			}
			checks = append(checks, check)
		}
	}
	if len(checks) > 3 {
		return append(checks[:3], fmt.Sprintf("+%d more failed", len(checks)-3))
	}
	return checks
}

func (m Model) reviewPassedCheckCount(nodeNumbers []int) int {
	passed := 0
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		for _, verification := range node.Verifications {
			if verification.Passed {
				passed++
			}
		}
	}
	return passed
}

// checkNoisePrefixes are lines agents paste in from CLI output they ran. They
// describe the run, not the result, so they are dropped before summarizing.
var checkNoisePrefixes = []string{
	"Setting up fixture for:",
	"Running fixture for:",
	"Checking for new versions",
	"A newer version of the Stripe CLI is available",
	"We recommend updating regularly",
	"Trigger succeeded!",
	"Ran fixture",
}

// summarizeCheck condenses an agent-authored verification note to one scannable
// line.
//
// These notes are frequently pasted CLI transcripts rather than prose. Agents
// splice command output into the middle of their own sentence, so the note
// opens with a fragment, carries dozens of progress lines, and states the
// actual finding last. Keeping the head — whether by sentence split or plain
// truncation — therefore returns the chatter and discards the result. Drop the
// pure progress lines, then keep the tail. The untouched text stays in the
// detail view.
// summarizeCheck reduces a verification to the one line a reviewer needs.
//
// When the agent supplies --detail this is barely needed: Check is already a
// label. It matters for sessions where the agent pasted a whole transcript into
// --check, and the shape of that prose is specific. The finding is not at the
// end — agent transcripts trail off into what *did* work ("The handler reads the
// raw body, verifies the signature…") after stating what did not. An earlier
// tail-biased version of this showed exactly that trailing reassurance under a
// red ✗, and opened it with a bare "…".
//
// So: drop the progress noise, then lead with the reason. "because X" is where
// agents put it, and a sentence carrying a failure marker beats one that does
// not. Truncation is always at the end, so a line never opens mid-word.
func summarizeCheck(s string, budget int) string {
	return summarizeCheckResult(s, budget, false)
}

// summarizeCheckFailure is summarizeCheck for a check that did not pass, where
// preferring the sentence that explains the failure is the whole point.
func summarizeCheckFailure(s string, budget int) string {
	return summarizeCheckResult(s, budget, true)
}

func summarizeCheckResult(s string, budget int, failed bool) string {
	var kept []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || isProgressLine(line) {
			continue
		}
		kept = append(kept, line)
	}
	summary := strings.Join(strings.Fields(strings.Join(kept, " ")), " ")
	if summary == "" {
		return ""
	}

	if failed {
		if reason := failureReason(summary); reason != "" {
			summary = reason
		}
	}
	if budget <= 0 || lipgloss.Width(summary) <= budget {
		return summary
	}
	return ansi.Truncate(summary, budget, "…")
}

// identifierPattern matches the tokens a reviewer scans for in a check: Stripe
// event names (checkout.session.completed) and environment variables
// (STRIPE_WEBHOOK_SECRET). Both are the subject of the sentence, and in a wall
// of prose at one weight they were impossible to pick out.
var identifierPattern = regexp.MustCompile(`\b([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*){2,}|[A-Z][A-Z0-9]*(?:_[A-Z0-9]+){1,})\b`)

// highlightIdentifiers emphasizes those tokens inside an already-styled line.
// The surrounding style is reapplied after each match so the rest of the line
// keeps its color rather than reverting to the terminal default.
func highlightIdentifiers(text string, base, accent lipgloss.Style) string {
	matches := identifierPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return base.Render(text)
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		b.WriteString(base.Render(text[last:m[0]]))
		b.WriteString(accent.Render(text[m[0]:m[1]]))
		last = m[1]
	}
	b.WriteString(base.Render(text[last:]))
	return b.String()
}

// failureMarkers are the phrases agents use when reporting that something did
// not work. Ordered by how specific they are.
var failureMarkers = []string{
	"could not", "cannot", "was not able", "unable to",
	"is not configured", "not configured", "missing",
	"failed", "error", "does not", "did not",
}

// failureReason picks the clause explaining why a check did not pass.
func failureReason(summary string) string {
	for _, sentence := range splitSentences(summary) {
		lower := strings.ToLower(sentence)
		if !containsAny(lower, failureMarkers) {
			continue
		}
		// Agents write "<what was tried> because <the actual reason>". The
		// reason is the half worth the single line available.
		if idx := strings.Index(lower, " because "); idx >= 0 {
			return canonicalCause(tidyClause(sentence[idx+len(" because "):]))
		}
		return canonicalCause(tidyClause(sentence))
	}
	return ""
}

// missingConfigPattern matches the way agents report an unset environment
// variable, in the several shapes they phrase it.
var missingConfigPattern = regexp.MustCompile(
	`([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)\s+(?:is|was)\s+not\s+(?:configured|set|present)|` +
		`(?:has|have)\s+no\s+([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)|` +
		`(?:missing|no)\s+([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)`)

// canonicalCause reduces a failure to the configuration it is actually blocked
// on, when it names one.
//
// Two tasks blocked on the same unset variable were reporting it in different
// words and at different lengths — "X is not configured, so a signed end-to-end
// delivery could not be completed in this environment." against "X is not
// configured." — because the agent wrote each sentence differently and the
// extraction faithfully preserved both. The consequence clause is noise; the
// variable is the thing to act on, and stating it identically makes one root
// cause read as one root cause.
func canonicalCause(reason string) string {
	m := missingConfigPattern.FindStringSubmatch(reason)
	if m == nil {
		return reason
	}
	for _, name := range m[1:] {
		if name != "" {
			return name + " is not configured."
		}
	}
	return reason
}

// splitSentences breaks on sentence-ending punctuation followed by a space. It
// keeps abbreviations and decimals intact by requiring the next character to
// start a new word.
func splitSentences(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s)-1; i++ {
		if s[i] != '.' && s[i] != '!' && s[i] != '?' && s[i] != ';' {
			continue
		}
		if s[i+1] != ' ' {
			continue
		}
		if sentence := strings.TrimSpace(s[start : i+1]); sentence != "" {
			out = append(out, sentence)
		}
		start = i + 1
	}
	if sentence := strings.TrimSpace(s[start:]); sentence != "" {
		out = append(out, sentence)
	}
	return out
}

// tidyClause makes a mid-sentence fragment stand on its own: a clause lifted
// from after "because" keeps the semicolon that joined it to the rest, which
// reads as though the line were cut short.
func tidyClause(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ";,")
	if s != "" && !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "!") && !strings.HasSuffix(s, "?") {
		s += "."
	}
	return capitalize(s)
}

func containsAny(s string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	// Leave identifiers like STRIPE_WEBHOOK_SECRET and checkout.session.completed
	// alone: forcing a case change on them makes them wrong, not tidy.
	if unicode.IsUpper(r[0]) || !unicode.IsLetter(r[0]) {
		return s
	}
	return string(unicode.ToUpper(r[0])) + string(r[1:])
}

// isProgressLine reports whether a line is only CLI progress output. A line that
// carries meaningful text alongside a known marker is kept, because agents
// append their finding to the last line of pasted output.
func isProgressLine(line string) bool {
	for _, prefix := range checkNoisePrefixes {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return lipgloss.Width(strings.TrimSpace(strings.TrimPrefix(line, prefix))) <= 60
	}
	return false
}

func pluralChecks(n int) string {
	if n == 1 {
		return "1 check"
	}
	return fmt.Sprintf("%d checks", n)
}

// reviewVenueLabel says where the reviewer goes to check the work. Node type is
// the reliable signal: blueprint review prompts are shared across every node of
// a given type and mostly describe what to confirm, not where to confirm it.
func (m Model) reviewVenueLabel(nodeNumbers []int) string {
	var venues []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		venue := reviewVenueForNode(node)
		if venue == "" || seen[venue] {
			continue
		}
		seen[venue] = true
		venues = append(venues, venue)
	}
	const limit = 2
	if len(venues) > limit {
		venues = venues[:limit]
	}
	return strings.Join(venues, "; ")
}

func reviewVenueForNode(node *coop.SessionNode) string {
	switch node.NodeType {
	case coop.NodeAPIRequest, coop.NodeTestHelper:
		return "the changed files below"
	case coop.NodeUIComponent:
		return "your running app"
	case coop.NodeDashboard:
		return "the Stripe Dashboard"
	case coop.NodeCLICommand:
		return "your terminal"
	case coop.NodeAsyncHandler, coop.NodeSetUpWebhooks:
		return "your webhook handler, after triggering the event"
	}
	return ""
}

func (m Model) reviewBlueprintConfirmationPrompts(nodeNumbers []int) []string {
	var prompts []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.ReviewPrompt == "" || seen[node.ReviewPrompt] {
			continue
		}
		seen[node.ReviewPrompt] = true
		prompts = append(prompts, node.ReviewPrompt)
	}
	const limit = 2
	if len(prompts) > limit {
		return append(prompts[:limit], fmt.Sprintf("Open details for %s.", pluralChecks(len(prompts)-limit)))
	}
	return prompts
}

func (m Model) reviewCommandLabel(nodeNumbers []int) string {
	var commands []string
	seen := map[string]bool{}
	for _, nodeNumber := range nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil {
			continue
		}
		command := reviewCommandForNode(node)
		if command == "" || seen[command] {
			continue
		}
		seen[command] = true
		commands = append(commands, command)
	}
	if len(commands) == 0 {
		return ""
	}
	return strings.Join(commands, "\n")
}

func reviewCommandForNode(node *coop.SessionNode) string {
	return node.ReviewCommand
}

// firstActionableReviewStep returns the step waiting on the user, so the footer
// can name it and jump to it rather than just reporting that one exists.
func (m Model) firstActionableReviewStep() (int, bool) {
	if m.session == nil {
		return 0, false
	}
	step := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			step++
			if m.session.Steps[i].Nodes[j].State == coop.NodeReview && m.reviewIsActionable(step) {
				return i, true
			}
		}
	}
	return 0, false
}

func (m Model) actionableReviewCount() int {
	if m.session == nil {
		return 0
	}
	count := 0
	countedSteps := map[int]bool{}
	step := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			step++
			if m.session.Steps[i].Nodes[j].State != coop.NodeReview || !m.reviewIsActionable(step) {
				continue
			}
			if !countedSteps[i] {
				count++
				countedSteps[i] = true
			}
		}
	}
	return count
}

func (m Model) agentIdle() bool {
	return m.agentIsIdle
}
