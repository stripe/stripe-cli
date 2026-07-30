package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// Frame capture harness.
//
// Design mocks drawn by hand repeatedly misrepresented real data — a fabricated
// failure line, a step rendered with one Run: command when its nodes carry two,
// a vertical budget quoted from a blueprint size that does not exist. Rendering
// the real thing catches all of that in seconds, so the harness renders every
// interesting state across every width tier that production actually uses, in
// both color modes, and writes the frames out for review.
//
//	COOP_TUI_CAPTURE_DIR=/tmp/frames go test ./pkg/coop/tui -run TestCaptureFrames
//
// Without the env var the test still runs — it just asserts the frames render
// without panicking and stay inside their terminal, so the harness itself is
// covered in CI.

// captureSizes covers the width tiers production switches on. 100 is the
// threshold where the split workspace engages (outline.go useSplitWorkspace),
// so 96 and 100 bracket it — that boundary is where a layout rule most easily
// gets written for one mode and forgotten in the other.
//
// The split-workspace nav column is clamped to 34–48 columns, but those are
// widths of a column inside a wide terminal, not terminal widths themselves;
// they are exercised by the 100 and 120 captures rather than by shrinking the
// whole frame, which production never does.
var captureSizes = []layoutSize{
	{name: "tiny", width: 40, height: 12},
	{name: "narrow", width: 56, height: 18},
	{name: "short", width: 80, height: 12},
	{name: "normal", width: 80, height: 24},
	{name: "presplit", width: 96, height: 30},
	{name: "split", width: 100, height: 30},
	{name: "wide", width: 120, height: 40},
}

// monochrome renders what a terminal with color disabled actually shows.
//
// This is deliberately not ansi.Strip: Strip removes every escape sequence,
// including bold and italic, which would understate a design whose hierarchy
// depends on weight surviving color loss. colorprofile.Ascii drops color SGR
// parameters and keeps the rest, which is what a real no-color terminal does.
func monochrome(s string) string {
	var buf bytes.Buffer
	w := &colorprofile.Writer{Forward: &buf, Profile: colorprofile.Ascii}
	if _, err := w.WriteString(s); err != nil {
		return s
	}
	return buf.String()
}

// logOverflow reports lines that exceed the terminal width without failing.
// Used for real-session captures, where the data is local to the developer.
func logOverflow(t *testing.T, label string, size layoutSize, rendered string) {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		if width := lipgloss.Width(line); width > size.width {
			t.Logf("overflow %s @%s: %d > %d: %q", label, size.name, width, size.width, ansi.Strip(line))
		}
	}
}

func captureFrame(m *Model, size layoutSize) string {
	m.spinner = staticSpinner()
	m.ready = true
	m.width = size.width
	m.height = size.height
	m.viewport = viewport.New(viewport.WithWidth(size.width), viewport.WithHeight(10))
	m.resizeViewport()
	m.syncViewport()
	return m.View().Content
}

func writeCapture(t *testing.T, scenario, mode string, size layoutSize, rendered string) {
	t.Helper()
	dir := os.Getenv("COOP_TUI_CAPTURE_DIR")
	if dir == "" {
		return
	}
	require.NoError(t, os.MkdirAll(dir, 0o755))
	name := fmt.Sprintf("%s--%s--%s-%dx%d.txt", scenario, mode, size.name, size.width, size.height)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(rendered), 0o644))
}

type captureScenario struct {
	name  string
	model func() Model
}

// captureScenarios reuses the model builders the layout and stress suites
// already maintain, so new states get captured automatically as those grow.
func captureScenarios() []captureScenario {
	return []captureScenario{
		{"waiting", waitingLayoutModel},
		{"active_step", activeStepLayoutModel},
		{"review_step_long_prompt", reviewStepLongPromptLayoutModel},
		{"failed_check", failedCheckLayoutModel},
		{"step_goal", goalLayoutModel},
		{"step_review_many_changes", stepReviewLayoutModel},
		{"request_changes_input", requestChangesLayoutModel},
		{"manual_navigation", manualNavigationLayoutModel},
		{"expanded_details", expandedDetailsLayoutModel},
		{"completion", completionLayoutModel},
		{"stress_long_review", stressLongReviewModel},
		{"stress_crowded_step_review", stressCrowdedStepReviewModel},
		{"stress_long_claim_url", stressLongClaimURLModel},
		{"stress_many_steps_manual_nav", stressManyStepsManualNavigationModel},
		{"stress_long_rejection_input", stressLongRejectionInputModel},
	}
}

func TestCaptureFrames(t *testing.T) {
	for _, scenario := range captureScenarios() {
		for _, size := range captureSizes {
			m := scenario.model()
			rendered := captureFrame(&m, size)

			writeCapture(t, scenario.name, "color", size, rendered)
			writeCapture(t, scenario.name, "mono", size, monochrome(rendered))

			assertLayoutFits(t, rendered, size)
		}
	}
}

// TestCaptureExtremeShapes renders the shapes that bound the layout: the most
// steps, the most tasks in one step, and the longest step description.
//
// It used to load three named blueprints by id from the embedded filesystem.
// Blueprints come from the API now and are no longer bundled, so the shapes are
// constructed here instead — the value was always the geometry, not those
// particular files.
func TestCaptureExtremeShapes(t *testing.T) {
	longDescription := strings.Repeat(
		"This step wires the integration end to end and has a description long "+
			"enough to wrap several times in a narrow pane. ", 4)

	manySteps := &coop.Session{Blueprint: "many-steps", Status: coop.SessionActive}
	for i := 1; i <= 6; i++ {
		manySteps.Steps = append(manySteps.Steps, tuiStep(
			fmt.Sprintf("s%d", i), fmt.Sprintf("Step %d with a title that has to scan well", i),
			tuiNode(coop.NodeAPIRequest, fmt.Sprintf("s%dn1", i), "Do the thing", coop.NodeDone),
		))
	}

	crowded := &coop.Session{Blueprint: "crowded-step", Status: coop.SessionActive}
	var crowdedNodes []coop.SessionNode
	for i := 1; i <= 7; i++ {
		crowdedNodes = append(crowdedNodes, tuiNode(
			coop.NodeAPIRequest, fmt.Sprintf("n%d", i),
			fmt.Sprintf("Task %d with a moderately long label", i), coop.NodeReview))
	}
	crowded.Steps = []coop.SessionStep{tuiStep("crowded", "One step holding every task", crowdedNodes...)}

	longDesc := &coop.Session{Blueprint: "long-description", Status: coop.SessionActive}
	step := tuiStep("long", "A step with a very long description",
		tuiNode(coop.NodeUIComponent, "n1", "Check the flow", coop.NodeReview))
	step.Description = coop.MessageDescriptor{DefaultMessage: longDescription}
	longDesc.Steps = []coop.SessionStep{step}

	for _, sc := range []struct {
		name    string
		session *coop.Session
	}{
		{"extreme_many_steps", manySteps},
		{"extreme_crowded_step", crowded},
		{"extreme_long_description", longDesc},
	} {
		for _, size := range captureSizes {
			m := testModel()
			m.session = sc.session
			m.selectStep(0)
			rendered := captureFrame(&m, size)

			writeCapture(t, sc.name, "color", size, rendered)
			writeCapture(t, sc.name, "mono", size, monochrome(rendered))
			assertLayoutFits(t, rendered, size)
			assertBoxesBalance(t, rendered)
		}
	}
}

// TestCaptureRealSession renders whatever sessions exist on this machine. It is
// skipped when there are none, so it never fails in CI, but locally it is the
// only way to see genuine agent-authored prose — which is nothing like the
// tidy strings in the fixtures.
func TestCaptureRealSession(t *testing.T) {
	dir := os.Getenv("COOP_TUI_SESSION_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		dir = filepath.Join(home, ".config", "stripe", "coop")
	}
	entries, err := filepath.Glob(filepath.Join(dir, "coop_*.json"))
	if err != nil || len(entries) == 0 {
		t.Skip("no local co-op sessions to render")
	}

	store, err := coop.NewStoreAt(dir)
	require.NoError(t, err)

	rendered := 0
	for _, entry := range entries {
		id := strings.TrimSuffix(filepath.Base(entry), ".json")
		session, err := store.Read(id)
		if err != nil || session == nil || len(session.Steps) == 0 {
			continue
		}
		for _, size := range captureSizes {
			m := testModel()
			m.session = session
			m.selectStep(0)
			frame := captureFrame(&m, size)

			writeCapture(t, "session_"+id, "color", size, frame)
			writeCapture(t, "session_"+id, "mono", size, monochrome(frame))

			// Reported, not asserted: these sessions live on the developer's
			// machine and are not in the repo, so they cannot gate CI. The
			// fixture and blueprint captures carry the hard assertions.
			logOverflow(t, id, size, frame)
		}
		rendered++
	}
	if rendered == 0 {
		t.Skip("no readable co-op sessions to render")
	}
}

// TestFramesAreNotVisiblyBroken sweeps every captured frame for the two faults
// that make the UI look broken rather than merely cramped: content past the
// right edge, and a card whose borders do not balance.
//
// Both were found by eye first. A lone "╭────╮" hanging over blank space
// survived a full test suite because every assertion was about content, not
// about the frame holding together.
func TestFramesAreNotVisiblyBroken(t *testing.T) {
	for _, scenario := range captureScenarios() {
		for _, size := range captureSizes {
			t.Run(scenario.name+"/"+size.name, func(t *testing.T) {
				m := scenario.model()
				rendered := captureFrame(&m, size)

				assertLayoutFits(t, rendered, size)
				assertBoxesBalance(t, rendered)
			})
		}
	}
}

// assertBoxesBalance reports a card that opens and never closes. Split-workspace
// frames join two independently drawn columns onto shared lines, so a border
// from either column can appear on any line; the check is over the frame as a
// whole rather than per column.
func assertBoxesBalance(t *testing.T, rendered string) {
	t.Helper()
	plain := ansi.Strip(rendered)
	opens := strings.Count(plain, "╭")
	closes := strings.Count(plain, "╰")
	// One-directional. A card whose top has scrolled out of view closes without
	// opening, which is correct; a card that opens and never closes is the
	// fault — it runs off the bottom of the region with nothing terminating it.
	assert.LessOrEqual(t, opens, closes,
		"every card that opens should close; got %d open and %d closed:\n%s", opens, closes, plain)
}
