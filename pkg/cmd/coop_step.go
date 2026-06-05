package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopStepCmd struct {
	cmd *cobra.Command

	// Common flags
	session string
	stdin   bool

	// start flags
	note string

	// done flags
	file        string
	lines       string
	snippet     string
	autoConfirm bool

	// verify flags
	check  string
	passed bool
}

func newCoopStepCmd() *coopStepCmd {
	sc := &coopStepCmd{}

	sc.cmd = &cobra.Command{
		Use:   "step <number> <action>",
		Short: "Update a step's lifecycle state",
		Long: `Manage the lifecycle of a co-op session step. Actions:
  start   — mark step as active (agent is working on it)
  done    — mark step as complete (moves to review unless --auto-confirm)
  verify  — add a verification check to the step
  skip    — skip a step with an optional reason`,
		Example: `  stripe coop step 1 start --note="Installing dependencies"
  stripe coop step 1 done --file=server.js --lines=1-15 --note="Added Stripe SDK"
  stripe coop step 1 verify --check="package.json has stripe" --passed
  stripe coop step 2 skip --note="Already installed"`,
		Args: cobra.ExactArgs(2),
		RunE: sc.runStepCmd,
	}

	sc.cmd.Flags().StringVar(&sc.session, "session", "", "Session ID (defaults to latest active session)")
	sc.cmd.Flags().BoolVar(&sc.stdin, "stdin", false, "Read implementation/verify data as JSON from stdin (avoids shell escaping issues)")
	sc.cmd.Flags().StringVar(&sc.note, "note", "", "Activity note or reason")
	sc.cmd.Flags().StringVar(&sc.file, "file", "", "File path for implementation")
	sc.cmd.Flags().StringVar(&sc.lines, "lines", "", "Line range (e.g. 1-15)")
	sc.cmd.Flags().StringVar(&sc.snippet, "snippet", "", "Code snippet")
	sc.cmd.Flags().BoolVar(&sc.autoConfirm, "auto-confirm", false, "Skip review state, go directly to done")
	sc.cmd.Flags().StringVar(&sc.check, "check", "", "Verification check label")
	sc.cmd.Flags().BoolVar(&sc.passed, "passed", false, "Whether the verification check passed")

	return sc
}

func (sc *coopStepCmd) runStepCmd(cmd *cobra.Command, args []string) error {
	stepNum, err := strconv.Atoi(args[0])
	if err != nil {
		return outputCoopError(fmt.Sprintf("Invalid step number %q: must be an integer", args[0]), "stripe coop status")
	}

	action := args[1]

	if sc.stdin {
		if err := sc.readFromStdin(); err != nil {
			return outputCoopError(err.Error(), "Ensure valid JSON is piped to stdin")
		}
	}

	store, err := coop.NewStore(Config.GetConfigFolder(""))
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	var session *coop.Session
	if sc.session != "" {
		session, err = store.Read(sc.session)
	} else {
		session, err = store.LatestActiveSession()
	}
	if err != nil {
		return outputCoopError(fmt.Sprintf("Cannot load session: %s", err), "stripe coop run <blueprint>")
	}

	switch action {
	case "start":
		return sc.doStart(store, session, stepNum)
	case "done":
		return sc.doDone(store, session, stepNum)
	case "verify":
		return sc.doVerify(store, session, stepNum)
	case "skip":
		return sc.doSkip(store, session, stepNum)
	case "await":
		return sc.doAwait(store, session, stepNum)
	default:
		return outputCoopError(fmt.Sprintf("Unknown action %q. Valid actions: start, done, verify, skip, await", action), "stripe coop step --help")
	}
}

func (sc *coopStepCmd) doStart(store *coop.Store, session *coop.Session, stepNum int) error {
	if err := session.TransitionStep(stepNum, coop.StepActive); err != nil {
		return outputCoopError(err.Error(), fmt.Sprintf("stripe coop status --session=%s", session.ID))
	}

	node, _ := session.NodeByNumber(stepNum)
	if sc.note != "" {
		node.Activity = sc.note
	}

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	return outputJSON(coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      stepNum,
		State:     string(coop.StepActive),
		Message:   fmt.Sprintf("Started: %s", node.Title),
		Next:      fmt.Sprintf("stripe coop step %d done --file=<path> --note=\"<what you did>\"", stepNum),
	})
}

func (sc *coopStepCmd) doDone(store *coop.Store, session *coop.Session, stepNum int) error {
	targetState := coop.StepReview
	node, _ := session.NodeByNumber(stepNum)
	if sc.autoConfirm || (node != nil && node.AutoConfirm) {
		targetState = coop.StepDone
	}

	if err := session.TransitionStep(stepNum, targetState); err != nil {
		return outputCoopError(err.Error(), fmt.Sprintf("stripe coop step %d start", stepNum))
	}

	node, _ = session.NodeByNumber(stepNum)
	if sc.file != "" || sc.snippet != "" || sc.note != "" {
		node.Implementation = &coop.Implementation{
			File:    sc.file,
			Lines:   sc.lines,
			Snippet: sc.snippet,
			Note:    sc.note,
		}
	}
	node.Activity = ""

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	stateStr := string(targetState)
	var msg, next string

	if targetState == coop.StepReview {
		msg = fmt.Sprintf("Ready for review: %s (waiting for human to confirm)", node.Title)
		next = fmt.Sprintf("stripe coop step %d await", stepNum)
	} else {
		msg = fmt.Sprintf("Completed: %s", node.Title)
		if nextStep := session.NextPendingStep(stepNum); nextStep > 0 {
			nextNode, _ := session.NodeByNumber(nextStep)
			next = fmt.Sprintf("stripe coop step %d start --note=\"Beginning: %s\"", nextStep, nextNode.Title)
		} else if session.IsComplete() {
			msg += " All steps complete! Run next-steps — developer picks from TUI."
			next = "stripe coop next-steps"
		} else {
			next = "stripe coop status"
		}
	}

	return outputJSON(coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      stepNum,
		State:     stateStr,
		Message:   msg,
		Next:      next,
	})
}

func (sc *coopStepCmd) doVerify(store *coop.Store, session *coop.Session, stepNum int) error {
	node, err := session.NodeByNumber(stepNum)
	if err != nil {
		return outputCoopError(err.Error(), "stripe coop status")
	}

	if sc.check == "" {
		return outputCoopError("--check flag is required for verify action", fmt.Sprintf("stripe coop step %d verify --check=\"<label>\" --passed", stepNum))
	}

	node.Verifications = append(node.Verifications, coop.Verification{
		Check:  sc.check,
		Passed: sc.passed,
	})

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	status := "✓"
	if !sc.passed {
		status = "✗"
	}

	return outputJSON(coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      stepNum,
		State:     string(node.State),
		Message:   fmt.Sprintf("Verification added: %s %s", status, sc.check),
		Next:      fmt.Sprintf("stripe coop step %d done --file=<path> --note=\"<what you did>\"", stepNum),
	})
}

func (sc *coopStepCmd) doAwait(store *coop.Store, session *coop.Session, stepNum int) error {
	node, err := session.NodeByNumber(stepNum)
	if err != nil {
		return outputCoopError(err.Error(), "stripe coop status")
	}

	// Auto-confirm steps skip human review entirely
	if node.AutoConfirm && node.State == coop.StepReview {
		session.TransitionStep(stepNum, coop.StepDone)
		store.Write(session)
		next := ""
		msg := fmt.Sprintf("Step %d auto-confirmed. Proceed to next step.", stepNum)
		if nextStep := session.NextPendingStep(stepNum); nextStep > 0 {
			nextNode, _ := session.NodeByNumber(nextStep)
			next = fmt.Sprintf("stripe coop step %d start --note=\"Beginning: %s\"", nextStep, nextNode.Title)
		} else if session.IsComplete() {
			msg = fmt.Sprintf("Step %d auto-confirmed. All steps complete!", stepNum)
			next = "stripe coop next-steps"
		}
		return outputJSON(coop.CommandResponse{
			OK:        true,
			SessionID: session.ID,
			Step:      stepNum,
			State:     "confirmed",
			Message:   msg,
			Next:      next,
		})
	}

	if node.State != coop.StepReview {
		next := ""
		msg := fmt.Sprintf("Step %d is already %s.", stepNum, node.State)
		if session.IsComplete() {
			msg = fmt.Sprintf("Step %d confirmed. All steps done — run the next command now to show the developer their options.", stepNum)
			next = "stripe coop next-steps"
		} else if nextStep := session.NextPendingStep(stepNum); nextStep > 0 {
			nextNode, _ := session.NodeByNumber(nextStep)
			msg = fmt.Sprintf("Step %d is done. Proceed to next step.", stepNum)
			next = fmt.Sprintf("stripe coop step %d start --note=\"Beginning: %s\"", nextStep, nextNode.Title)
		}
		return outputJSON(coop.CommandResponse{
			OK:        true,
			SessionID: session.ID,
			Step:      stepNum,
			State:     string(node.State),
			Message:   msg,
			Next:      next,
		})
	}

	// Poll until state changes from review (10-minute timeout)
	store.WriteHeartbeat(session.ID)
	defer store.RemoveHeartbeat(session.ID)

	deadline := time.Now().Add(10 * time.Minute)
	for {
		if time.Now().After(deadline) {
			return outputJSON(coop.CommandResponse{
				OK:        true,
				SessionID: session.ID,
				Step:      stepNum,
				State:     "timeout",
				Message:   "Timed out waiting for developer confirmation (10 minutes). The developer may still be reviewing. Re-run this command to wait again.",
				Next:      fmt.Sprintf("stripe coop step %d await", stepNum),
			})
		}

		time.Sleep(500 * time.Millisecond)
		store.WriteHeartbeat(session.ID)

		session, err = store.Read(session.ID)
		if err != nil {
			return fmt.Errorf("reading session: %w", err)
		}

		node, err = session.NodeByNumber(stepNum)
		if err != nil {
			return fmt.Errorf("reading step: %w", err)
		}

		if node.State == coop.StepReview {
			continue
		}

		// Step was confirmed or rejected
		if node.State == coop.StepDone {
			next := ""
			msg := fmt.Sprintf("Step %d confirmed by developer. Proceed to next step.", stepNum)
			if nextStep := session.NextPendingStep(stepNum); nextStep > 0 {
				nextNode, _ := session.NodeByNumber(nextStep)
				next = fmt.Sprintf("stripe coop step %d start --note=\"Beginning: %s\"", nextStep, nextNode.Title)
			} else if session.IsComplete() {
				msg = fmt.Sprintf("Step %d confirmed. All steps complete! You MUST run the next command immediately — it shows the developer their options and blocks until they choose.", stepNum)
				next = "stripe coop next-steps"
			} else {
				next = "stripe coop status"
			}
			return outputJSON(coop.CommandResponse{
				OK:        true,
				SessionID: session.ID,
				Step:      stepNum,
				State:     "confirmed",
				Message:   msg,
				Next:      next,
			})
		}

		if node.State == coop.StepActive {
			msg := fmt.Sprintf("Step %d rejected by developer.", stepNum)
			if node.RejectionNote != "" {
				msg += fmt.Sprintf("\nFeedback: %s", node.RejectionNote)
			}
			msg += "\nAsk the developer what they'd like you to change, then redo the step."
			return outputJSON(coop.CommandResponse{
				OK:        true,
				SessionID: session.ID,
				Step:      stepNum,
				State:     "rejected",
				Message:   msg,
				Next:      fmt.Sprintf("stripe coop step %d done --file=<path> --note=\"<what you fixed>\"", stepNum),
			})
		}

		// Some other state (skipped, etc)
		return outputJSON(coop.CommandResponse{
			OK:        true,
			SessionID: session.ID,
			Step:      stepNum,
			State:     string(node.State),
			Message:   fmt.Sprintf("Step %d is now %s", stepNum, node.State),
		})
	}
}

// readFromStdin reads implementation data from stdin as JSON.
// Accepts: {"file":"...", "lines":"...", "snippet":"...", "note":"...", "check":"...", "passed":true}
func (sc *coopStepCmd) readFromStdin() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("--stdin flag set but no data received on stdin")
	}

	var input struct {
		File    string `json:"file"`
		Lines   string `json:"lines"`
		Snippet string `json:"snippet"`
		Note    string `json:"note"`
		Check   string `json:"check"`
		Passed  bool   `json:"passed"`
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("parsing stdin JSON: %w", err)
	}

	if input.File != "" {
		sc.file = input.File
	}
	if input.Lines != "" {
		sc.lines = input.Lines
	}
	if input.Snippet != "" {
		sc.snippet = input.Snippet
	}
	if input.Note != "" {
		sc.note = input.Note
	}
	if input.Check != "" {
		sc.check = input.Check
	}
	if input.Passed {
		sc.passed = true
	}
	return nil
}

func (sc *coopStepCmd) doSkip(store *coop.Store, session *coop.Session, stepNum int) error {
	if err := session.TransitionStep(stepNum, coop.StepSkipped); err != nil {
		return outputCoopError(err.Error(), fmt.Sprintf("stripe coop status --session=%s", session.ID))
	}

	node, _ := session.NodeByNumber(stepNum)
	if sc.note != "" {
		node.Activity = sc.note
	}

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	next := ""
	if nextStep := session.NextPendingStep(stepNum); nextStep > 0 {
		nextNode, _ := session.NodeByNumber(nextStep)
		next = fmt.Sprintf("stripe coop step %d start --note=\"Beginning: %s\"", nextStep, nextNode.Title)
	} else {
		next = "stripe coop status"
	}

	return outputJSON(coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      stepNum,
		State:     string(coop.StepSkipped),
		Message:   fmt.Sprintf("Skipped: %s", node.Title),
		Next:      next,
	})
}
