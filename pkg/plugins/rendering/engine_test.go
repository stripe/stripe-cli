package rendering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

func newTestEngine(format Format) (*Engine, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return NewEngineWithWriters(format, stdout, stderr), stdout, stderr
}

func messageBlock(msg string, level proto.MessageLevel) *proto.OutputBlock {
	return &proto.OutputBlock{Block: &proto.OutputBlock_Message{
		Message: &proto.MessageBlock{Message: msg, Level: level},
	}}
}

func progressBlock(p *proto.ProgressBlock) *proto.OutputBlock {
	return &proto.OutputBlock{Block: &proto.OutputBlock_Progress{Progress: p}}
}

// chatter builds a non-final request: incremental output sent while the command
// is still running.
func chatter(blocks ...*proto.OutputBlock) *proto.SendCommandOutputRequest {
	return &proto.SendCommandOutputRequest{Blocks: blocks}
}

func dataBlock(blockType, payload string) *proto.OutputBlock {
	return &proto.OutputBlock{Block: &proto.OutputBlock_Data{
		Data: &proto.DataBlock{Type: blockType, Payload: payload},
	}}
}

func TestMessageBlockRendersEachLevel(t *testing.T) {
	tests := []struct {
		level proto.MessageLevel
		want  string
	}{
		// INFO carries the plugin's own formatting, so it is passed through
		// unprefixed to keep parity with what plugins print locally.
		{proto.MessageLevel_INFO, "hello\n"},
		{proto.MessageLevel_SUCCESS, "✔ hello\n"},
		{proto.MessageLevel_WARNING, "⚠ hello\n"},
		{proto.MessageLevel_ERROR, "✗ hello\n"},
	}

	for _, tc := range tests {
		t.Run(tc.level.String(), func(t *testing.T) {
			engine, stdout, stderr := newTestEngine(FormatText)
			engine.HandleCommandOutput(chatter(messageBlock("hello", tc.level)))
			require.Equal(t, tc.want, stdout.String())
			require.Empty(t, stderr.String())
		})
	}
}

func TestIgnoresEmptyRequests(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatText)

	engine.HandleCommandOutput(nil)
	engine.HandleCommandOutput(chatter())

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestMessageBlockWritesToStderrInJSONMode(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatJSON)

	engine.HandleCommandOutput(chatter(messageBlock("working", proto.MessageLevel_INFO)))

	// stdout is reserved for the JSON envelope.
	require.Empty(t, stdout.String())
	require.Equal(t, "working\n", stderr.String())
}

func TestProgressBlockRendersStepsAndSpinners(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Message: "step one", Type: proto.ProgressType_STEP,
	})))
	// Non-terminal writers get plain lines instead of an animation.
	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "uploading", Type: proto.ProgressType_SPINNER_START,
	})))
	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "still uploading", Type: proto.ProgressType_SPINNER_UPDATE,
	})))
	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "uploaded", Type: proto.ProgressType_SPINNER_STOP, Success: true,
	})))

	require.Equal(t, "✔ step one\nuploading\nstill uploading\n✔ uploaded\n", stdout.String())
}

func TestProgressBlockFailedSpinner(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "uploading", Type: proto.ProgressType_SPINNER_START,
	})))
	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "upload failed", Type: proto.ProgressType_SPINNER_STOP, Success: false,
	})))

	require.Equal(t, "uploading\n✗ upload failed\n", stdout.String())
}

func TestProgressBlockStopWithoutMessageReusesStartMessage(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "uploading", Type: proto.ProgressType_SPINNER_START,
	})))
	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Type: proto.ProgressType_SPINNER_STOP, Success: true,
	})))

	require.Equal(t, "uploading\n✔ uploading\n", stdout.String())
}

func TestProgressBlockStopWithoutStartStillReportsOutcome(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "unknown", Message: "done anyway", Type: proto.ProgressType_SPINNER_STOP, Success: true,
	})))

	require.Equal(t, "✔ done anyway\n", stdout.String())
}

func TestIgnoresNilBlocks(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(nil))

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestHandleCommandOutputPreservesBlockOrder(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Command: "apps upload",
		Blocks: []*proto.OutputBlock{
			dataBlock("data", `{"app_id":"app_123","version":"1.0.0","dashboard_url":"https://example.test/app_123"}`),
			dataBlock("warning", `{"code":"PERMANENT_ID","message":"Your app ID is permanent"}`),
			dataBlock("nextstep", `{"code":"NAVIGATE_TO_APP","description":"View status","command":"https://example.test/app_123"}`),
		},
	})

	require.Equal(t, strings.Join([]string{
		"  app_id: app_123",
		"  version: 1.0.0",
		"  dashboard_url: https://example.test/app_123",
		"⚠ Your app ID is permanent",
		"→ View status: https://example.test/app_123",
		"",
	}, "\n"), stdout.String())
}

// Field order comes from the payload, not from Go map iteration, so repeated
// renders of the same payload are byte-identical.
func TestHandleCommandOutputDataFieldOrderIsDeterministic(t *testing.T) {
	payload := `{"zeta":1,"alpha":2,"mid":3,"beta":4,"omega":5}`

	var first string
	for i := 0; i < 20; i++ {
		engine, stdout, _ := newTestEngine(FormatText)
		engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
			Blocks: []*proto.OutputBlock{dataBlock("data", payload)},
		})
		if i == 0 {
			first = stdout.String()
			continue
		}
		require.Equal(t, first, stdout.String())
	}

	require.Equal(t, "  zeta: 1\n  alpha: 2\n  mid: 3\n  beta: 4\n  omega: 5\n", first)
}

func TestHandleCommandOutputRendersNestedAndTypedValues(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{dataBlock("data", `{
			"name": "ticket-manager",
			"count": 3,
			"live": false,
			"note": null,
			"tags": ["a", "b"],
			"nested": {"k": "v"}
		}`)},
	})

	require.Equal(t, strings.Join([]string{
		"  name: ticket-manager",
		"  count: 3",
		"  live: false",
		// A JSON null renders as an empty value, not the literal "null".
		"  note: ",
		`  tags: ["a","b"]`,
		`  nested: {"k":"v"}`,
		"",
	}, "\n"), stdout.String())
}

// Large integers must survive rendering; decoding through float64 would turn
// this into 1.2345678901234568e+18.
func TestHandleCommandOutputPreservesLargeNumbers(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{dataBlock("data", `{"id":1234567890123456789}`)},
	})

	require.Equal(t, "  id: 1234567890123456789\n", stdout.String())
}

func TestHandleCommandOutputMessageAndProgressBlocks(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{
			messageBlock("all set", proto.MessageLevel_SUCCESS),
			{Block: &proto.OutputBlock_Progress{Progress: &proto.ProgressBlock{
				Message: "built files", Type: proto.ProgressType_STEP,
			}}},
		},
	})

	require.Equal(t, "✔ all set\n✔ built files\n", stdout.String())
}

func TestHandleCommandOutputErrorBlockGoesToStderr(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatText)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{dataBlock("error", `{"code":"UPLOAD_FAILED","message":"could not upload"}`)},
	})

	require.Empty(t, stdout.String())
	require.Equal(t, "✗ could not upload\n", stderr.String())
}

func TestHandleCommandOutputIgnoresEmptyRequests(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatText)

	engine.HandleCommandOutput(nil)
	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{})
	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{Blocks: []*proto.OutputBlock{nil}})

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

// Malformed payloads are shown verbatim rather than silently dropped.
func TestHandleCommandOutputRendersInvalidPayloadsVerbatim(t *testing.T) {
	tests := []struct {
		name      string
		blockType string
		payload   string
		want      string
	}{
		{"data not an object", "data", `["a","b"]`, "[\"a\",\"b\"]\n"},
		{"data truncated", "data", `{"app_id":`, "{\"app_id\":\n"},
		{"warning not json", "warning", `oops`, "oops\n"},
		{"warning missing message", "warning", `{"code":"X"}`, "{\"code\":\"X\"}\n"},
		{"nextstep empty fields", "nextstep", `{}`, "{}\n"},
		{"unknown block type", "somethingnew", `{"a":1}`, "{\"a\":1}\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine, stdout, _ := newTestEngine(FormatText)
			engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
				Blocks: []*proto.OutputBlock{dataBlock(tc.blockType, tc.payload)},
			})
			require.Equal(t, tc.want, stdout.String())
		})
	}
}

func TestHandleCommandOutputStopsSpinnersFirst(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "uploading", Type: proto.ProgressType_SPINNER_START,
	})))
	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Final:  true,
		Blocks: []*proto.OutputBlock{dataBlock("data", `{"app_id":"app_123"}`)},
	})

	require.Equal(t, "uploading\n  app_id: app_123\n", stdout.String())
	require.Empty(t, engine.spinners, "spinners must be cleaned up before the result is rendered")
}

func TestHandleCommandOutputJSONEnvelope(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatJSON)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Command: "apps upload",
		Final:   true,
		Blocks: []*proto.OutputBlock{
			dataBlock("data", `{"app_id":"app_123"}`),
			dataBlock("warning", `{"code":"C","message":"m"}`),
			// Non-data blocks are transient and excluded from the envelope.
			messageBlock("chatter", proto.MessageLevel_INFO),
		},
	})

	var envelope JSONEnvelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, "apps upload", envelope.Command)
	require.Len(t, envelope.Data, 2)
	require.Equal(t, "data", envelope.Data[0].Type)
	require.JSONEq(t, `{"app_id":"app_123"}`, string(envelope.Data[0].Payload))
	require.Equal(t, "warning", envelope.Data[1].Type)
	// The message block is not envelope data; it goes to stderr so it cannot
	// corrupt the document on stdout.
	require.Equal(t, "chatter\n", stderr.String())
}

// A malformed payload must not produce an unparseable envelope.
func TestHandleCommandOutputJSONEnvelopeQuotesInvalidPayloads(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatJSON)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Command: "apps upload",
		Final:   true,
		Blocks:  []*proto.OutputBlock{dataBlock("data", `{"app_id":`)},
	})

	require.True(t, json.Valid(stdout.Bytes()), "envelope must be valid JSON: %s", stdout.String())

	var envelope JSONEnvelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Len(t, envelope.Data, 1)
	require.Equal(t, `"{\"app_id\":"`, string(envelope.Data[0].Payload))
}

// The engine renders synchronously under a lock, so a plugin sending from
// several goroutines can't interleave partial lines.
func TestConcurrentSendsProduceWholeLines(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	const senders = 8
	const perSender = 25

	var wg sync.WaitGroup
	for i := 0; i < senders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < perSender; j++ {
				engine.HandleCommandOutput(chatter(
					messageBlock(fmt.Sprintf("sender-%d-line-%d", i, j), proto.MessageLevel_INFO),
				))
			}
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	require.Len(t, lines, senders*perSender)
	for _, line := range lines {
		require.Regexp(t, `^sender-\d+-line-\d+$`, line)
	}
}

// Payloads far larger than gRPC's 4MB default must render intact. Before
// centralized output, output this size hit the GRPCStdio truncation bug.
func TestHandleCommandOutputLargePayload(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	const size = 8 << 20 // 8MB
	value := strings.Repeat("x", size)
	payload, err := json.Marshal(map[string]string{"blob": value})
	require.NoError(t, err)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{dataBlock("data", string(payload))},
	})

	require.Equal(t, "  blob: "+value+"\n", stdout.String())
}

func TestHandlePrompt(t *testing.T) {
	tests := []struct {
		name      string
		format    Format
		req       *proto.PromptRequest
		wantValue string
		wantText  string
	}{
		{
			name:      "text default",
			format:    FormatText,
			req:       &proto.PromptRequest{Message: "Name", DefaultValue: "app"},
			wantValue: "app",
			wantText:  "? Name [app]: \n",
		},
		{
			name:      "confirm",
			format:    FormatText,
			req:       &proto.PromptRequest{Message: "Proceed", Type: proto.PromptType_CONFIRM, DefaultValue: "y"},
			wantValue: "y",
			wantText:  "? Proceed (y/N) [y]: \n",
		},
		{
			name:      "select falls back to first option",
			format:    FormatText,
			req:       &proto.PromptRequest{Message: "Pick", Type: proto.PromptType_SELECT, Options: []string{"a", "b"}},
			wantValue: "a",
			wantText:  "? Pick\n  1. a\n  2. b\n",
		},
		{
			name:      "json mode does not prompt",
			format:    FormatJSON,
			req:       &proto.PromptRequest{Message: "Pick", Type: proto.PromptType_SELECT, Options: []string{"a", "b"}},
			wantValue: "a",
			wantText:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine, stdout, stderr := newTestEngine(tc.format)
			resp := engine.HandlePrompt(tc.req)
			require.Equal(t, tc.wantValue, resp.GetValue())
			if tc.format == FormatJSON {
				require.Equal(t, tc.wantText, stderr.String())
				require.Empty(t, stdout.String())
				return
			}
			require.Equal(t, tc.wantText, stdout.String())
		})
	}
}

func TestHandlePromptNilRequest(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)
	require.Empty(t, engine.HandlePrompt(nil).GetValue())
	require.Empty(t, stdout.String())
}

// --- one-RPC semantics ---

// Chatter arrives on the same RPC as the result, so it must not tear down a
// spinner the command is still using.
func TestNonFinalRequestLeavesSpinnersRunning(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatText)

	engine.HandleCommandOutput(chatter(progressBlock(&proto.ProgressBlock{
		Id: "s1", Message: "uploading", Type: proto.ProgressType_SPINNER_START,
	})))
	engine.HandleCommandOutput(chatter(messageBlock("still going", proto.MessageLevel_INFO)))

	require.Len(t, engine.spinners, 1, "a status message must not stop the command's spinner")
	require.Equal(t, "uploading\nstill going\n", stdout.String())
}

// One command must produce exactly one JSON document however many times it
// sends, or anything parsing our stdout breaks.
func TestJSONEnvelopeIsEmittedOnceAcrossManySends(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatJSON)

	engine.HandleCommandOutput(chatter(messageBlock("working", proto.MessageLevel_INFO)))
	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{dataBlock("data", `{"app_id":"app_123"}`)},
	})
	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Command: "apps upload",
		Final:   true,
		Blocks:  []*proto.OutputBlock{dataBlock("warning", `{"code":"C","message":"m"}`)},
	})

	require.True(t, json.Valid(stdout.Bytes()), "envelope must be valid JSON: %s", stdout.String())

	var envelope JSONEnvelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, "apps upload", envelope.Command)
	require.Len(t, envelope.Data, 2, "data from every send belongs to the one envelope")
	require.Equal(t, "data", envelope.Data[0].Type)
	require.Equal(t, "warning", envelope.Data[1].Type)
	// Chatter cannot go to stdout without corrupting the envelope.
	require.Equal(t, "working\n", stderr.String())
}

// Nothing is written until the command says it is done.
func TestJSONEnvelopeWaitsForFinal(t *testing.T) {
	engine, stdout, _ := newTestEngine(FormatJSON)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Command: "apps upload",
		Blocks:  []*proto.OutputBlock{dataBlock("data", `{"app_id":"app_123"}`)},
	})

	require.Empty(t, stdout.String())
}

// A block kind this build does not know decodes to an empty variant and the RPC
// still succeeds, so the plugin cannot detect it. Saying something is the only
// way the user learns output went missing.
func TestUnknownBlockVariantIsReportedNotDropped(t *testing.T) {
	engine, stdout, stderr := newTestEngine(FormatText)

	engine.HandleCommandOutput(&proto.SendCommandOutputRequest{
		Final: true,
		Blocks: []*proto.OutputBlock{
			dataBlock("data", `{"app_id":"app_123"}`),
			{}, // a variant from a newer plugin
			{},
		},
	})

	require.Equal(t, "  app_id: app_123\n", stdout.String(), "known blocks still render")
	require.Contains(t, stderr.String(), "2 output block(s) require a newer Stripe CLI")
}
