// Package rendering implements the centralized UI rendering engine for plugin output.
package rendering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/briandowns/spinner"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// Format selects how the engine renders plugin output.
type Format string

const (
	// FormatText renders human-readable output.
	FormatText Format = "text"
	// FormatJSON renders a single JSON envelope for the command's final output.
	FormatJSON Format = "json"
)

// Block type names carried in DataBlock.Type.
const (
	blockTypeData     = "data"
	blockTypeWarning  = "warning"
	blockTypeNextStep = "nextstep"
	blockTypeError    = "error"
)

// Engine renders output blocks sent by plugins over the CoreCLIHelper RPCs.
//
// All rendering goes to the writers the host configured, never to the plugin's
// own stdio. Rendering is serialized so that blocks appear in the order the
// plugin sent them even when a plugin sends from multiple goroutines.
type Engine struct {
	format   Format
	stdout   io.Writer
	stderr   io.Writer
	spinners map[string]*activeSpinner
	mu       sync.Mutex
}

type activeSpinner struct {
	message string
	// s is nil when the target writer is not a terminal; in that case progress
	// is rendered as plain lines instead of an animation.
	s *spinner.Spinner
}

// NewEngine creates an Engine that writes to the process's stdout and stderr.
func NewEngine(format Format) *Engine {
	return NewEngineWithWriters(format, os.Stdout, os.Stderr)
}

// NewEngineWithWriters creates an Engine that writes to the given streams.
func NewEngineWithWriters(format Format, stdout, stderr io.Writer) *Engine {
	return &Engine{
		format:   format,
		stdout:   stdout,
		stderr:   stderr,
		spinners: make(map[string]*activeSpinner),
	}
}

// messageWriter returns the stream that incremental output goes to.
//
// In text mode, messages and progress are part of the visible output and go to
// stdout, matching what plugins print today. In JSON mode stdout is reserved
// for the single command envelope, so incremental output goes to stderr to
// avoid corrupting it.
func (e *Engine) messageWriter() io.Writer {
	if e.format == FormatJSON {
		return e.stderr
	}
	return e.stdout
}

// HandleMessage renders a single incremental message.
func (e *Engine) HandleMessage(req *proto.SendMessageRequest) {
	if req == nil || req.GetMessage() == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.renderMessage(req.GetMessage())
}

// HandleProgress renders a single progress update.
func (e *Engine) HandleProgress(req *proto.SendProgressRequest) {
	if req == nil || req.GetProgress() == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.renderProgress(req.GetProgress())
}

// HandleCommandOutput renders a command's final output blocks.
func (e *Engine) HandleCommandOutput(req *proto.SendCommandOutputRequest) {
	if req == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// A command's final output ends any progress still in flight; leaving a
	// spinner running would keep overwriting the result line.
	e.stopAllSpinners()

	if e.format == FormatJSON {
		e.renderJSONEnvelope(req)
		return
	}

	for _, block := range req.GetBlocks() {
		if block == nil {
			continue
		}
		switch b := block.Block.(type) {
		case *proto.OutputBlock_Message:
			e.renderMessage(b.Message)
		case *proto.OutputBlock_Data:
			e.renderData(b.Data)
		case *proto.OutputBlock_Progress:
			e.renderProgress(b.Progress)
		}
	}
}

// HandlePrompt asks the user a question and returns their answer.
//
// Milestone 1 keeps this deliberately minimal: prompts still render as plain
// text and resolve to the caller's default. Interactive prompting is a separate
// milestone.
func (e *Engine) HandlePrompt(req *proto.PromptRequest) *proto.PromptResponse {
	if req == nil {
		return &proto.PromptResponse{}
	}

	value := req.GetDefaultValue()
	if value == "" && req.GetType() == proto.PromptType_SELECT && len(req.GetOptions()) > 0 {
		value = req.GetOptions()[0]
	}

	if e.format == FormatJSON {
		// Non-interactive: never block waiting on input we can't render.
		return &proto.PromptResponse{Value: value}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	w := e.messageWriter()
	switch req.GetType() {
	case proto.PromptType_CONFIRM:
		fmt.Fprintf(w, "? %s (y/N) [%s]: \n", req.GetMessage(), value)
	case proto.PromptType_SELECT:
		fmt.Fprintf(w, "? %s\n", req.GetMessage())
		for i, opt := range req.GetOptions() {
			fmt.Fprintf(w, "  %d. %s\n", i+1, opt)
		}
	default:
		fmt.Fprintf(w, "? %s [%s]: \n", req.GetMessage(), value)
	}

	return &proto.PromptResponse{Value: value}
}

// --- text rendering (callers hold e.mu) ---

func (e *Engine) renderMessage(msg *proto.MessageBlock) {
	if msg == nil {
		return
	}

	w := e.messageWriter()
	prefix := levelPrefix(msg.GetLevel(), w)
	if prefix == "" {
		fmt.Fprintln(w, msg.GetMessage())
		return
	}
	fmt.Fprintf(w, "%s %s\n", prefix, msg.GetMessage())
}

func (e *Engine) renderData(data *proto.DataBlock) {
	if data == nil {
		return
	}

	w := e.stdout
	color := ansi.Color(w)

	switch data.GetType() {
	case blockTypeData:
		fields, err := orderedFields(data.GetPayload())
		if err != nil {
			// Never drop output we can't parse — show it verbatim.
			fmt.Fprintln(w, strings.TrimRight(data.GetPayload(), "\n"))
			return
		}
		for _, f := range fields {
			fmt.Fprintf(w, "  %s: %s\n", f.key, f.value)
		}
	case blockTypeWarning:
		var payload struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(data.GetPayload()), &payload); err != nil || payload.Message == "" {
			fmt.Fprintln(w, strings.TrimRight(data.GetPayload(), "\n"))
			return
		}
		fmt.Fprintln(w, color.Yellow("⚠"), payload.Message)
	case blockTypeNextStep:
		var payload struct {
			Description string `json:"description"`
			Command     string `json:"command"`
		}
		if err := json.Unmarshal([]byte(data.GetPayload()), &payload); err != nil {
			fmt.Fprintln(w, strings.TrimRight(data.GetPayload(), "\n"))
			return
		}
		switch {
		case payload.Description == "" && payload.Command == "":
			fmt.Fprintln(w, strings.TrimRight(data.GetPayload(), "\n"))
		case payload.Command == "":
			fmt.Fprintf(w, "%s %s\n", color.Blue("→"), payload.Description)
		default:
			fmt.Fprintf(w, "%s %s: %s\n", color.Blue("→"), payload.Description, payload.Command)
		}
	case blockTypeError:
		var payload struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(data.GetPayload()), &payload); err != nil || payload.Message == "" {
			fmt.Fprintln(e.stderr, strings.TrimRight(data.GetPayload(), "\n"))
			return
		}
		fmt.Fprintf(e.stderr, "%s %s\n", ansi.Color(e.stderr).Red("✗"), payload.Message)
	default:
		// Unknown block types are forwarded rather than swallowed so a newer
		// plugin against an older core still shows something useful.
		fmt.Fprintln(w, strings.TrimRight(data.GetPayload(), "\n"))
	}
}

func (e *Engine) renderProgress(p *proto.ProgressBlock) {
	if p == nil {
		return
	}

	w := e.messageWriter()
	color := ansi.Color(w)

	switch p.GetType() {
	case proto.ProgressType_SPINNER_START:
		e.startSpinner(p.GetId(), p.GetMessage(), w)
	case proto.ProgressType_SPINNER_UPDATE:
		e.updateSpinner(p.GetId(), p.GetMessage(), w)
	case proto.ProgressType_SPINNER_STOP:
		e.stopSpinner(p.GetId(), p.GetMessage(), p.GetSuccess(), w)
	case proto.ProgressType_STEP:
		fmt.Fprintf(w, "%s %s\n", color.Green("✔"), faint(w, p.GetMessage()))
	default:
		fmt.Fprintln(w, faint(w, p.GetMessage()))
	}
}

func (e *Engine) startSpinner(id, msg string, w io.Writer) {
	// A repeated start for the same id is treated as an update so a plugin
	// retry can't leave an orphaned animation running.
	if existing, ok := e.spinners[id]; ok {
		e.finishSpinner(id, existing, "", true, w)
	}
	e.spinners[id] = &activeSpinner{message: msg, s: ansi.StartNewSpinner(msg, w)}
}

func (e *Engine) updateSpinner(id, msg string, w io.Writer) {
	existing, ok := e.spinners[id]
	if !ok {
		// Update without a start: render it as a fresh spinner rather than
		// dropping the message.
		e.startSpinner(id, msg, w)
		return
	}
	existing.message = msg
	if existing.s != nil {
		ansi.StartSpinner(existing.s, msg, w)
		return
	}
	fmt.Fprintln(w, msg)
}

func (e *Engine) stopSpinner(id, msg string, success bool, w io.Writer) {
	existing, ok := e.spinners[id]
	if !ok {
		// Stop without a start still reports the outcome.
		e.renderSpinnerOutcome(msg, success, w)
		return
	}
	e.finishSpinner(id, existing, msg, success, w)
}

func (e *Engine) finishSpinner(id string, s *activeSpinner, msg string, success bool, w io.Writer) {
	delete(e.spinners, id)
	finalMsg := msg
	if finalMsg == "" {
		finalMsg = s.message
	}
	stopSpinner(s.s)
	e.renderSpinnerOutcome(finalMsg, success, w)
}

func (e *Engine) renderSpinnerOutcome(msg string, success bool, w io.Writer) {
	if msg == "" {
		return
	}
	color := ansi.Color(w)
	mark := color.Green("✔")
	if !success {
		mark = color.Red("✗")
	}
	fmt.Fprintf(w, "%s %s\n", mark, faint(w, msg))
}

// stopAllSpinners clears any spinner still running, without claiming success or
// failure for work whose outcome the plugin never reported.
func (e *Engine) stopAllSpinners() {
	if len(e.spinners) == 0 {
		return
	}
	for _, id := range sortedSpinnerIDs(e.spinners) {
		stopSpinner(e.spinners[id].s)
		delete(e.spinners, id)
	}
}

// stopSpinner halts an animation without printing a final line. ansi.StopSpinner
// is avoided here because it prints an empty line when given an empty message.
func stopSpinner(s *spinner.Spinner) {
	if s == nil {
		return
	}
	s.FinalMSG = ""
	s.Stop()
}

func sortedSpinnerIDs(spinners map[string]*activeSpinner) []string {
	ids := make([]string, 0, len(spinners))
	for id := range spinners {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// --- JSON rendering ---

func (e *Engine) renderJSONEnvelope(req *proto.SendCommandOutputRequest) {
	envelope := JSONEnvelope{Command: req.GetCommand()}
	for _, block := range req.GetBlocks() {
		if block == nil {
			continue
		}
		db := block.GetData()
		if db == nil {
			continue
		}
		payload := json.RawMessage(db.GetPayload())
		if !json.Valid(payload) {
			// Keep the envelope valid JSON even if a plugin sent garbage.
			quoted, err := json.Marshal(db.GetPayload())
			if err != nil {
				continue
			}
			payload = quoted
		}
		envelope.Data = append(envelope.Data, EnvelopeBlock{
			Type:    db.GetType(),
			Payload: payload,
		})
	}

	enc := json.NewEncoder(e.stdout)
	enc.SetIndent("", "  ")
	// Encode failures here mean the stream is broken; there is nowhere useful
	// left to report that, so the error is intentionally dropped.
	_ = enc.Encode(envelope)
}

// --- helpers ---

type field struct {
	key   string
	value string
}

// orderedFields decodes a JSON object into its fields, preserving the order
// they appear in the payload so rendered output matches what the plugin built.
func orderedFields(payload string) ([]field, error) {
	dec := json.NewDecoder(strings.NewReader(payload))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("payload is not a JSON object")
	}

	var fields []field
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected object key %v", keyTok)
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		fields = append(fields, field{key: key, value: renderValue(raw)})
	}
	return fields, nil
}

// renderValue formats a JSON value for text output: strings unquoted, and
// everything else compacted onto a single line.
func renderValue(raw json.RawMessage) string {
	if strings.TrimSpace(string(raw)) == "null" {
		// Rendering the literal "null" to a user is noise; an absent value
		// reads better as an empty one.
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return string(raw)
	}
	return buf.String()
}

// faint renders de-emphasized text for the given writer. ansi.Faint is keyed to
// os.Stdout, which is the wrong stream when the host renders elsewhere.
func faint(w io.Writer, text string) string {
	color := ansi.Color(w)
	return color.Sprintf(color.Faint(text))
}

func levelPrefix(level proto.MessageLevel, w io.Writer) string {
	color := ansi.Color(w)
	switch level {
	case proto.MessageLevel_SUCCESS:
		return fmt.Sprint(color.Green("✔"))
	case proto.MessageLevel_WARNING:
		return fmt.Sprint(color.Yellow("⚠"))
	case proto.MessageLevel_ERROR:
		return fmt.Sprint(color.Red("✗"))
	default:
		// INFO carries the plugin's own formatting; adding a prefix here would
		// double up on what plugins already print.
		return ""
	}
}
