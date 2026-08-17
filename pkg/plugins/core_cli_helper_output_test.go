package plugins

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
	"github.com/stripe/stripe-cli/pkg/plugins/rendering"
)

// recordingHelper is a CoreCLIHelper that records the centralized-output calls
// it receives and can be made to fail.
type recordingHelper struct {
	CoreCLIHelper

	messages []*proto.SendMessageRequest
	progress []*proto.SendProgressRequest
	outputs  []*proto.SendCommandOutputRequest
	prompts  []*proto.PromptRequest

	err error
}

func (h *recordingHelper) SendMessage(req *proto.SendMessageRequest) error {
	h.messages = append(h.messages, req)
	return h.err
}

func (h *recordingHelper) SendProgress(req *proto.SendProgressRequest) error {
	h.progress = append(h.progress, req)
	return h.err
}

func (h *recordingHelper) SendCommandOutput(req *proto.SendCommandOutputRequest) error {
	h.outputs = append(h.outputs, req)
	return h.err
}

func (h *recordingHelper) Prompt(req *proto.PromptRequest) (*proto.PromptResponse, error) {
	h.prompts = append(h.prompts, req)
	if h.err != nil {
		return nil, h.err
	}
	return &proto.PromptResponse{Value: req.GetDefaultValue()}, nil
}

func TestCoreCLIHelperServerForwardsOutputRPCs(t *testing.T) {
	impl := &recordingHelper{}
	server := &CoreCLIHelperServer{Impl: impl}
	ctx := context.Background()

	msgReq := &proto.SendMessageRequest{Message: &proto.MessageBlock{Message: "hi", Level: proto.MessageLevel_SUCCESS}}
	msgResp, err := server.SendMessage(ctx, msgReq)
	require.NoError(t, err)
	require.NotNil(t, msgResp)

	progressReq := &proto.SendProgressRequest{Progress: &proto.ProgressBlock{Id: "s1", Message: "working", Type: proto.ProgressType_SPINNER_START}}
	progressResp, err := server.SendProgress(ctx, progressReq)
	require.NoError(t, err)
	require.NotNil(t, progressResp)

	outputReq := &proto.SendCommandOutputRequest{Command: "apps upload"}
	outputResp, err := server.SendCommandOutput(ctx, outputReq)
	require.NoError(t, err)
	require.NotNil(t, outputResp)

	promptResp, err := server.Prompt(ctx, &proto.PromptRequest{Message: "Proceed", DefaultValue: "y"})
	require.NoError(t, err)
	require.Equal(t, "y", promptResp.GetValue())

	require.Equal(t, []*proto.SendMessageRequest{msgReq}, impl.messages)
	require.Equal(t, []*proto.SendProgressRequest{progressReq}, impl.progress)
	require.Equal(t, []*proto.SendCommandOutputRequest{outputReq}, impl.outputs)
	require.Len(t, impl.prompts, 1)
}

func TestCoreCLIHelperServerPropagatesErrors(t *testing.T) {
	wantErr := errors.New("boom")
	server := &CoreCLIHelperServer{Impl: &recordingHelper{err: wantErr}}
	ctx := context.Background()

	_, err := server.SendMessage(ctx, &proto.SendMessageRequest{})
	require.ErrorIs(t, err, wantErr)

	_, err = server.SendProgress(ctx, &proto.SendProgressRequest{})
	require.ErrorIs(t, err, wantErr)

	_, err = server.SendCommandOutput(ctx, &proto.SendCommandOutputRequest{})
	require.ErrorIs(t, err, wantErr)

	_, err = server.Prompt(ctx, &proto.PromptRequest{})
	require.ErrorIs(t, err, wantErr)
}

func TestCoreCLIHelperRendersToConfiguredWriters(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	helper := NewCoreCLIHelperWithWriters(context.Background(), nil, afero.NewMemMapFs(), rendering.FormatText, stdout, stderr)

	require.NoError(t, helper.SendMessage(&proto.SendMessageRequest{
		Message: &proto.MessageBlock{Message: "starting", Level: proto.MessageLevel_INFO},
	}))
	require.NoError(t, helper.SendProgress(&proto.SendProgressRequest{
		Progress: &proto.ProgressBlock{Message: "built files", Type: proto.ProgressType_STEP},
	}))
	require.NoError(t, helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Command: "apps upload",
		Blocks: []*proto.OutputBlock{{Block: &proto.OutputBlock_Data{Data: &proto.DataBlock{
			Type:    "data",
			Payload: `{"app_id":"app_123"}`,
		}}}},
	}))

	require.Equal(t, "starting\n✔ built files\n  app_id: app_123\n", stdout.String())
	require.Empty(t, stderr.String())
}

// Output is written by the host, so nothing should reach the process's own
// stdio when writers are supplied.
func TestCoreCLIHelperLargeCommandOutputRendersIntact(t *testing.T) {
	stdout := &bytes.Buffer{}
	helper := NewCoreCLIHelperWithWriters(context.Background(), nil, afero.NewMemMapFs(), rendering.FormatText, stdout, &bytes.Buffer{})

	const size = 8 << 20 // 8MB: well past gRPC's 4MB default message limit
	blob := strings.Repeat("y", size)

	require.NoError(t, helper.SendCommandOutput(&proto.SendCommandOutputRequest{
		Blocks: []*proto.OutputBlock{{Block: &proto.OutputBlock_Data{Data: &proto.DataBlock{
			Type:    "data",
			Payload: `{"blob":"` + blob + `"}`,
		}}}},
	}))

	require.Equal(t, "  blob: "+blob+"\n", stdout.String())
}

func TestCoreCLIHelperPromptReturnsDefault(t *testing.T) {
	stdout := &bytes.Buffer{}
	helper := NewCoreCLIHelperWithWriters(context.Background(), nil, afero.NewMemMapFs(), rendering.FormatText, stdout, &bytes.Buffer{})

	resp, err := helper.Prompt(&proto.PromptRequest{Message: "Proceed", Type: proto.PromptType_CONFIRM, DefaultValue: "y"})
	require.NoError(t, err)
	require.Equal(t, "y", resp.GetValue())
	require.Contains(t, stdout.String(), "? Proceed")
}
