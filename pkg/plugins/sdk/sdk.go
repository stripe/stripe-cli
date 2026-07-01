// Package sdk provides a high-level API for plugins to send output through
// the core CLI's centralized rendering engine.
package sdk

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

var spinnerCounter uint64

// CLI wraps a CoreCLIHelper to provide convenient output methods for plugins.
type CLI struct {
	helper plugins.CoreCLIHelper
}

// New creates a new CLI SDK instance from a CoreCLIHelper.
func New(helper plugins.CoreCLIHelper) *CLI {
	return &CLI{helper: helper}
}

// --- Messages ---

// Message sends an informational message to the user.
func (c *CLI) Message(msg string) error {
	return c.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Message{Message: &proto.MessageBlock{
				Message: msg,
				Level:   proto.MessageLevel_INFO,
			}},
		}},
	})
}

// Success sends a success message to the user.
func (c *CLI) Success(msg string) error {
	return c.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Message{Message: &proto.MessageBlock{
				Message: msg,
				Level:   proto.MessageLevel_SUCCESS,
			}},
		}},
	})
}

// Warn sends a warning message to the user.
func (c *CLI) Warn(msg string) error {
	return c.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Message{Message: &proto.MessageBlock{
				Message: msg,
				Level:   proto.MessageLevel_WARNING,
			}},
		}},
	})
}

// Error sends an error message to the user.
func (c *CLI) Error(msg string) error {
	return c.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Message{Message: &proto.MessageBlock{
				Message: msg,
				Level:   proto.MessageLevel_ERROR,
			}},
		}},
	})
}

// --- Progress ---

// Progress sends a one-shot step indicator (checkmark line, no animation).
func (c *CLI) Progress(msg string) error {
	return c.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Progress{Progress: &proto.ProgressBlock{
				Id:      fmt.Sprintf("step-%d", atomic.AddUint64(&spinnerCounter, 1)),
				Message: msg,
				Type:    proto.ProgressType_STEP,
			}},
		}},
	})
}

// Spinner represents an active progress spinner that can be updated and stopped.
type Spinner struct {
	id     string
	helper plugins.CoreCLIHelper
}

// ProgressStart begins a spinner and returns a handle to control it.
func (c *CLI) ProgressStart(msg string) *Spinner {
	id := fmt.Sprintf("spinner-%d", atomic.AddUint64(&spinnerCounter, 1))
	c.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Progress{Progress: &proto.ProgressBlock{
				Id:      id,
				Message: msg,
				Type:    proto.ProgressType_SPINNER_START,
			}},
		}},
	})
	return &Spinner{id: id, helper: c.helper}
}

// Update changes the spinner's message while it's still running.
func (s *Spinner) Update(msg string) {
	s.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Progress{Progress: &proto.ProgressBlock{
				Id:      s.id,
				Message: msg,
				Type:    proto.ProgressType_SPINNER_UPDATE,
			}},
		}},
	})
}

// Stop stops the spinner. Pass true for success (checkmark), false for failure (x).
func (s *Spinner) Stop(success bool) {
	s.helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{
			Block: &proto.OutputBlock_Progress{Progress: &proto.ProgressBlock{
				Id:      s.id,
				Message: "",
				Type:    proto.ProgressType_SPINNER_STOP,
				Success: success,
			}},
		}},
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
	return OutputBlock{blockType: "warning", payload: map[string]string{
		"code":    "",
		"message": msg,
	}}
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
	return OutputBlock{blockType: "nextstep", payload: map[string]string{
		"code":        "",
		"description": description,
		"command":     command,
	}}
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
	req := &proto.SendCommandOutputRequest{
		Command: command,
	}

	for _, b := range blocks {
		payload, err := json.Marshal(b.payload)
		if err != nil {
			return err
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

type PromptType int

const (
	PromptText    PromptType = 0
	PromptConfirm PromptType = 1
	PromptSelect  PromptType = 2
)

// Prompt asks the user a question and returns their response.
func (c *CLI) Prompt(opts PromptOpts) (string, error) {
	req := &proto.PromptRequest{
		Message:      opts.Message,
		Type:         proto.PromptType(opts.Type),
		Options:      opts.Options,
		DefaultValue: opts.Default,
	}
	resp, err := c.helper.Prompt(req)
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}
