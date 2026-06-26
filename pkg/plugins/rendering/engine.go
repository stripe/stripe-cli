// Package rendering implements the centralized UI rendering engine for plugin output.
package rendering

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Engine struct {
	format   Format
	stdout   io.Writer
	stderr   io.Writer
	spinners map[string]string // id -> message (tracks active spinners)
	mu       sync.Mutex
}

func NewEngine(format Format) *Engine {
	return &Engine{
		format:   format,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
		spinners: make(map[string]string),
	}
}

func (e *Engine) HandleSendMessage(req *proto.SendMessageRequest) {
	switch e.format {
	case FormatJSON:
		fmt.Fprintln(e.stderr, req.Message)
	default:
		style := levelPrefix(req.Level)
		fmt.Fprintf(e.stdout, "%s %s\n", style, req.Message)
	}
}

func (e *Engine) HandleCommandOutput(req *proto.SendCommandOutputRequest) {
	switch e.format {
	case FormatJSON:
		envelope := JSONEnvelope{
			Command: req.Command,
		}
		for _, block := range req.Blocks {
			envelope.Data = append(envelope.Data, EnvelopeBlock{
				Type:    block.Type,
				Payload: json.RawMessage(block.Payload),
			})
		}
		enc := json.NewEncoder(e.stdout)
		enc.SetIndent("", "  ")
		enc.Encode(envelope)
	default:
		e.renderTextOutput(req)
	}
}

func (e *Engine) HandleProgress(req *proto.SendProgressRequest) {
	switch e.format {
	case FormatJSON:
		fmt.Fprintln(e.stderr, req.Message)
	default:
		e.handleSpinner(req)
	}
}

func (e *Engine) HandlePrompt(req *proto.PromptRequest) *proto.PromptResponse {
	switch e.format {
	case FormatJSON:
		// Non-interactive: return default (or first option for SELECT)
		value := req.DefaultValue
		if value == "" && req.Type == proto.PromptType_SELECT && len(req.Options) > 0 {
			value = req.Options[0]
		}
		return &proto.PromptResponse{Value: value}
	default:
		// POC: print prompt and return default. Real impl uses huh library.
		switch req.Type {
		case proto.PromptType_CONFIRM:
			fmt.Fprintf(e.stdout, "? %s (y/N) [%s]: ", req.Message, req.DefaultValue)
		case proto.PromptType_SELECT:
			fmt.Fprintf(e.stdout, "? %s\n", req.Message)
			for i, opt := range req.Options {
				fmt.Fprintf(e.stdout, "  %d. %s\n", i+1, opt)
			}
		default:
			fmt.Fprintf(e.stdout, "? %s [%s]: ", req.Message, req.DefaultValue)
		}
		return &proto.PromptResponse{Value: req.DefaultValue}
	}
}

func (e *Engine) handleSpinner(req *proto.SendProgressRequest) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch req.Type {
	case proto.ProgressType_STEP:
		fmt.Fprintf(e.stdout, "\033[32m✔\033[0m \033[2m%s\033[0m\n", req.Message)
	case proto.ProgressType_SPINNER_START:
		e.spinners[req.Id] = req.Message
		fmt.Fprintf(e.stdout, "\033[2m⠋ %s...\033[0m\n", req.Message)
	case proto.ProgressType_SPINNER_UPDATE:
		e.spinners[req.Id] = req.Message
		fmt.Fprintf(e.stdout, "\033[2m⠋ %s...\033[0m\n", req.Message)
	case proto.ProgressType_SPINNER_STOP:
		delete(e.spinners, req.Id)
		if req.Success {
			fmt.Fprintf(e.stdout, "\033[32m✔\033[0m \033[2m%s\033[0m\n", req.Message)
		} else {
			fmt.Fprintf(e.stdout, "\033[31m✗\033[0m \033[2m%s\033[0m\n", req.Message)
		}
	default:
		fmt.Fprintf(e.stdout, "\033[2m%s\033[0m\n", req.Message)
	}
}

func (e *Engine) renderTextOutput(req *proto.SendCommandOutputRequest) {
	for _, block := range req.Blocks {
		switch block.Type {
		case "data":
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(block.Payload), &data); err == nil {
				for k, v := range data {
					fmt.Fprintf(e.stdout, "  %s: %v\n", k, v)
				}
			}
		case "warning":
			var w struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(block.Payload), &w); err == nil {
				fmt.Fprintf(e.stdout, "\033[33m⚠\033[0m  %s\n", w.Message)
			}
		case "nextstep":
			var ns struct {
				Description string `json:"description"`
				Command     string `json:"command"`
			}
			if err := json.Unmarshal([]byte(block.Payload), &ns); err == nil {
				fmt.Fprintf(e.stdout, "\033[34m→\033[0m %s: %s\n", ns.Description, ns.Command)
			}
		case "error":
			var errBlock struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(block.Payload), &errBlock); err == nil {
				fmt.Fprintf(e.stdout, "\033[31m✗\033[0m %s\n", errBlock.Message)
			}
		}
	}
}

func levelPrefix(level proto.MessageLevel) string {
	switch level {
	case proto.MessageLevel_SUCCESS:
		return "\033[32m✔\033[0m"
	case proto.MessageLevel_WARNING:
		return "\033[33m⚠\033[0m"
	case proto.MessageLevel_ERROR:
		return "\033[31m✗\033[0m"
	default:
		return "\033[34mℹ\033[0m"
	}
}
