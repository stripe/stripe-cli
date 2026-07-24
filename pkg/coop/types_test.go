package coop

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandResponseValidate(t *testing.T) {
	input := []CommandInput{{Name: "note", Flag: "--note", Description: "Work summary."}}
	tests := []struct {
		name    string
		resp    CommandResponse
		wantErr string
	}{
		{name: "exact success", resp: CommandResponse{OK: true, Continuation: Continue("stripe coop status")}},
		{
			name: "template success",
			resp: CommandResponse{OK: true, Continuation: Continuation{
				NextTemplate: `stripe coop agent report-work --note="<note>"`, RequiredInputs: input,
			}},
		},
		{name: "terminal success", resp: CommandResponse{OK: true, State: "completed"}},
		{
			name: "exact recovery",
			resp: CommandResponse{
				OK: false, Error: "disk full",
				Recovery: Continue("stripe coop status").Recovery("Inspect the session."),
			},
		},
		{
			name: "template recovery",
			resp: CommandResponse{OK: false, Error: "missing note", Recovery: Continuation{
				NextTemplate: `stripe coop agent report-work --note="<note>"`, RequiredInputs: input,
			}.Recovery("Supply a note.")},
		},
		{
			name:    "placeholder in exact next",
			resp:    CommandResponse{OK: true, Continuation: Continue("stripe coop run <blueprint>")},
			wantErr: "template placeholder",
		},
		{
			name: "both continuation forms",
			resp: CommandResponse{OK: true, Continuation: Continuation{
				Next: "stripe coop status", NextTemplate: "stripe coop status", RequiredInputs: input,
			}},
			wantErr: "both next",
		},
		{
			name:    "template without inputs",
			resp:    CommandResponse{OK: true, Continuation: Continuation{NextTemplate: "stripe coop run <blueprint>"}},
			wantErr: "required_inputs",
		},
		{
			name:    "timeout without continuation",
			resp:    CommandResponse{OK: true, Continuation: Continuation{WaitTimeoutSeconds: 300}},
			wantErr: "requires a continuation",
		},
		{
			name:    "failure without recovery",
			resp:    CommandResponse{OK: false, Error: "failed"},
			wantErr: "recovery",
		},
		{
			name: "failure with top-level next",
			resp: CommandResponse{
				OK: false, Error: "failed", Continuation: Continue("stripe coop status"),
				Recovery: Continue("stripe coop status").Recovery("Inspect."),
			},
			wantErr: "inside recovery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCommandResponseJSONKeepsContinuationFieldsFlat(t *testing.T) {
	resp := CommandResponse{
		OK: true,
		Continuation: Continuation{
			NextTemplate:       `stripe coop run "<blueprint>"`,
			RequiredInputs:     []CommandInput{{Name: "blueprint", Description: "Blueprint ID."}},
			WaitTimeoutSeconds: 300,
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"ok": true,
		"next_template": "stripe coop run \"<blueprint>\"",
		"required_inputs": [{"name": "blueprint", "description": "Blueprint ID."}],
		"wait_timeout_seconds": 300
	}`, string(data))
}
