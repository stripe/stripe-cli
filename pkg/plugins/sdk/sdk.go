// Package sdk provides a high-level API for plugins to send output through
// the core CLI's centralized rendering engine.
//
// Plugins do not construct protobuf messages themselves; they call the semantic
// methods here. Every method returns an error so callers can decide whether to
// fall back to local rendering — see Unsupported, which identifies the only two
// conditions under which falling back is safe.
package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// ErrNoHelper is returned when the plugin was invoked without a core CLI helper
// (v1/v2 protocol, or standalone/dev mode). Callers should render locally.
var ErrNoHelper = errors.New("no core CLI helper available")

var idCounter uint64

// CLI wraps a CoreCLIHelper to provide convenient output methods for plugins.
type CLI struct {
	helper plugins.CoreCLIHelper
}

// New creates a new CLI SDK instance from a CoreCLIHelper. A nil helper is
// valid and yields a CLI whose Available reports false and whose methods return
// ErrNoHelper, so callers can use one code path for both cases.
func New(helper plugins.CoreCLIHelper) *CLI {
	return &CLI{helper: helper}
}

// Available reports whether output can be sent to a core CLI at all.
func (c *CLI) Available() bool {
	return c != nil && c.helper != nil
}

// Unsupported reports whether err means the core CLI cannot render this output,
// which is the only case where a plugin may safely render it locally instead:
//
//   - ErrNoHelper: no helper was provided, so nothing was sent.
//   - gRPC Unimplemented: the core CLI predates this RPC, so nothing was rendered.
//
// Any other error must be surfaced, not swallowed. A transport failure can
// happen after the core already rendered the output, and rendering locally in
// that case would show the user a duplicate.
func Unsupported(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNoHelper) {
		return true
	}
	return status.Code(err) == codes.Unimplemented
}

func (c *CLI) sendMessage(msg string, level proto.MessageLevel) error {
	if !c.Available() {
		return ErrNoHelper
	}
	return c.helper.SendMessage(&proto.SendMessageRequest{
		Message: &proto.MessageBlock{Message: msg, Level: level},
	})
}

func (c *CLI) sendProgress(block *proto.ProgressBlock) error {
	if !c.Available() {
		return ErrNoHelper
	}
	return c.helper.SendProgress(&proto.SendProgressRequest{Progress: block})
}

// --- Messages ---

// Message sends an informational message to the user.
func (c *CLI) Message(msg string) error {
	return c.sendMessage(msg, proto.MessageLevel_INFO)
}

// Success sends a success message to the user.
func (c *CLI) Success(msg string) error {
	return c.sendMessage(msg, proto.MessageLevel_SUCCESS)
}

// Warn sends a warning message to the user.
func (c *CLI) Warn(msg string) error {
	return c.sendMessage(msg, proto.MessageLevel_WARNING)
}

// Error sends an error message to the user.
func (c *CLI) Error(msg string) error {
	return c.sendMessage(msg, proto.MessageLevel_ERROR)
}

// --- Progress ---

// Progress sends a one-shot step indicator (checkmark line, no animation).
func (c *CLI) Progress(msg string) error {
	return c.sendProgress(&proto.ProgressBlock{
		Id:      nextID("step"),
		Message: msg,
		Type:    proto.ProgressType_STEP,
	})
}

// Spinner represents an active progress spinner that can be updated and stopped.
type Spinner struct {
	id  string
	cli *CLI
}

// ProgressStart begins a spinner and returns a handle to control it.
//
// The returned handle is always non-nil, so callers can defer Stop without a nil
// check even when the core CLI is unavailable.
func (c *CLI) ProgressStart(msg string) (*Spinner, error) {
	s := &Spinner{id: nextID("spinner"), cli: c}
	err := c.sendProgress(&proto.ProgressBlock{
		Id:      s.id,
		Message: msg,
		Type:    proto.ProgressType_SPINNER_START,
	})
	return s, err
}

// Update changes the spinner's message while it's still running.
func (s *Spinner) Update(msg string) error {
	return s.cli.sendProgress(&proto.ProgressBlock{
		Id:      s.id,
		Message: msg,
		Type:    proto.ProgressType_SPINNER_UPDATE,
	})
}

// Stop stops the spinner. Pass true for success (checkmark), false for failure
// (x). finalMsg is the line left behind; an empty string reuses the spinner's
// current message.
func (s *Spinner) Stop(finalMsg string, success bool) error {
	return s.cli.sendProgress(&proto.ProgressBlock{
		Id:      s.id,
		Message: finalMsg,
		Type:    proto.ProgressType_SPINNER_STOP,
		Success: success,
	})
}

// --- Command Output (block-builder pattern) ---

// OutputBlock is a typed block of output data.
type OutputBlock struct {
	blockType string
	payload   interface{}
}

// Data creates a data block with arbitrary key-value payload.
func Data(v interface{}) OutputBlock {
	return OutputBlock{blockType: "data", payload: v}
}

// Warning creates a warning block.
func Warning(msg string) OutputBlock {
	return WarningWithCode("", msg)
}

// WarningWithCode creates a warning block with an explicit code.
func WarningWithCode(code, msg string) OutputBlock {
	return OutputBlock{blockType: "warning", payload: map[string]string{
		"code":    code,
		"message": msg,
	}}
}

// NextStep creates a next-step block.
func NextStep(description, command string) OutputBlock {
	return NextStepWithCode("", description, command)
}

// NextStepWithCode creates a next-step block with an explicit code.
func NextStepWithCode(code, description, command string) OutputBlock {
	return OutputBlock{blockType: "nextstep", payload: map[string]string{
		"code":        code,
		"description": description,
		"command":     command,
	}}
}

// Output sends structured command output to core CLI.
// Only used for success output. For errors, use cli.Error() + return an error
// from the command (non-zero exit code).
//
// Blocks are rendered in the order provided.
//
// Usage:
//
//	cli.Output("apps upload",
//	    sdk.Data(map[string]any{"app_id": "app_1234", "version": "1.0.0"}),
//	    sdk.Warning("Your App ID is permanent once uploaded"),
//	    sdk.NextStep("View status", "https://dashboard.stripe.com/apps/app_1234"),
//	)
func (c *CLI) Output(command string, blocks ...OutputBlock) error {
	if !c.Available() {
		return ErrNoHelper
	}

	req := &proto.SendCommandOutputRequest{Command: command}
	for _, b := range blocks {
		payload, err := json.Marshal(b.payload)
		if err != nil {
			return fmt.Errorf("could not encode %s block: %w", b.blockType, err)
		}
		req.Blocks = append(req.Blocks, &proto.OutputBlock{
			Block: &proto.OutputBlock_Data{Data: &proto.DataBlock{
				Type:    b.blockType,
				Payload: string(payload),
			}},
		})
	}

	return c.helper.SendCommandOutput(req)
}

// --- Prompts ---

// PromptOpts configures a prompt.
type PromptOpts struct {
	Message string
	Type    PromptType
	Default string
	Options []string // required for PromptSelect
}

// PromptType selects the kind of prompt to show.
type PromptType int

// Prompt kinds.
const (
	PromptText    PromptType = 0
	PromptConfirm PromptType = 1
	PromptSelect  PromptType = 2
)

// Prompt asks the user a question and returns their response.
func (c *CLI) Prompt(opts PromptOpts) (string, error) {
	if !c.Available() {
		return "", ErrNoHelper
	}

	resp, err := c.helper.Prompt(&proto.PromptRequest{
		Message:      opts.Message,
		Type:         proto.PromptType(opts.Type),
		Options:      opts.Options,
		DefaultValue: opts.Default,
	})
	if err != nil {
		return "", err
	}
	return resp.GetValue(), nil
}

// nextID returns a process-unique id so concurrent progress indicators from the
// same plugin don't collide.
func nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddUint64(&idCounter, 1))
}
