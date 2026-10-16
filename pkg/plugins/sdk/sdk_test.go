package sdk

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
	"github.com/stripe/stripe-cli/pkg/plugins/rendering"
)

// fakeHelper records the requests the SDK builds and can fail with a chosen error.
//
// Every kind of output arrives on the one SendCommandOutput RPC, so requests are
// recorded in a single slice and the accessors below pick out the blocks a test
// cares about.
type fakeHelper struct {
	plugins.CoreCLIHelper

	requests []*proto.SendCommandOutputRequest
	prompts  []*proto.PromptRequest

	err error
}

func (h *fakeHelper) SendCommandOutput(req *proto.SendCommandOutputRequest) error {
	h.requests = append(h.requests, req)
	return h.err
}

func (h *fakeHelper) Prompt(req *proto.PromptRequest) (*proto.PromptResponse, error) {
	h.prompts = append(h.prompts, req)
	if h.err != nil {
		return nil, h.err
	}
	return &proto.PromptResponse{Value: "chosen"}, nil
}

// messages returns every MessageBlock sent, in order.
func (h *fakeHelper) messages() []*proto.MessageBlock {
	var out []*proto.MessageBlock
	for _, req := range h.requests {
		for _, b := range req.GetBlocks() {
			if m := b.GetMessage(); m != nil {
				out = append(out, m)
			}
		}
	}
	return out
}

// progress returns every ProgressBlock sent, in order.
func (h *fakeHelper) progress() []*proto.ProgressBlock {
	var out []*proto.ProgressBlock
	for _, req := range h.requests {
		for _, b := range req.GetBlocks() {
			if p := b.GetProgress(); p != nil {
				out = append(out, p)
			}
		}
	}
	return out
}

// finals returns only the requests that end a command's output, which is what
// the Output tests assert on.
func (h *fakeHelper) finals() []*proto.SendCommandOutputRequest {
	var out []*proto.SendCommandOutputRequest
	for _, req := range h.requests {
		if req.GetFinal() {
			out = append(out, req)
		}
	}
	return out
}

func TestMessagesCarryLevels(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	require.NoError(t, cli.Message("info"))
	require.NoError(t, cli.Success("success"))
	require.NoError(t, cli.Warn("warning"))
	require.NoError(t, cli.Error("error"))

	msgs := helper.messages()
	require.Len(t, msgs, 4)
	require.Equal(t, "info", msgs[0].GetMessage())
	require.Equal(t, proto.MessageLevel_INFO, msgs[0].GetLevel())
	require.Equal(t, proto.MessageLevel_SUCCESS, msgs[1].GetLevel())
	require.Equal(t, proto.MessageLevel_WARNING, msgs[2].GetLevel())
	require.Equal(t, proto.MessageLevel_ERROR, msgs[3].GetLevel())

	// Incremental output must not be marked final, or the core would tear down
	// spinners and close the JSON envelope mid-command.
	require.Empty(t, helper.finals())
}

func TestProgressLifecycle(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	require.NoError(t, cli.Progress("one step"))

	spinner, err := cli.ProgressStart("uploading")
	require.NoError(t, err)
	require.NoError(t, spinner.Update("still uploading"))
	require.NoError(t, spinner.Stop("uploaded", true))

	blocks := helper.progress()
	require.Len(t, blocks, 4)
	require.Equal(t, proto.ProgressType_STEP, blocks[0].GetType())

	start, update, stop := blocks[1], blocks[2], blocks[3]
	require.Equal(t, proto.ProgressType_SPINNER_START, start.GetType())
	require.Equal(t, proto.ProgressType_SPINNER_UPDATE, update.GetType())
	require.Equal(t, proto.ProgressType_SPINNER_STOP, stop.GetType())
	require.Equal(t, "uploaded", stop.GetMessage())
	require.True(t, stop.GetSuccess())

	// All three updates address the same spinner, and its id is distinct from
	// the one-shot step's.
	require.Equal(t, start.GetId(), update.GetId())
	require.Equal(t, start.GetId(), stop.GetId())
	require.NotEqual(t, blocks[0].GetId(), start.GetId())
}

func TestConcurrentSpinnersGetDistinctIDs(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	first, err := cli.ProgressStart("one")
	require.NoError(t, err)
	second, err := cli.ProgressStart("two")
	require.NoError(t, err)

	require.NotEqual(t, first.id, second.id)
}

func TestOutputBuildsOrderedDataBlocks(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	err := cli.Output("apps upload",
		Data(map[string]string{"app_id": "app_123"}),
		WarningWithCode("PERMANENT_ID", "Your app ID is permanent"),
		NextStepWithCode("NAVIGATE_TO_APP", "View status", "https://example.test/app_123"),
	)
	require.NoError(t, err)

	finals := helper.finals()
	require.Len(t, finals, 1)
	req := finals[0]
	require.Equal(t, "apps upload", req.GetCommand())
	require.Len(t, req.GetBlocks(), 3)

	require.Equal(t, "data", req.GetBlocks()[0].GetData().GetType())
	require.JSONEq(t, `{"app_id":"app_123"}`, req.GetBlocks()[0].GetData().GetPayload())

	require.Equal(t, "warning", req.GetBlocks()[1].GetData().GetType())
	require.JSONEq(t, `{"code":"PERMANENT_ID","message":"Your app ID is permanent"}`, req.GetBlocks()[1].GetData().GetPayload())

	require.Equal(t, "nextstep", req.GetBlocks()[2].GetData().GetType())
	require.JSONEq(t,
		`{"code":"NAVIGATE_TO_APP","description":"View status","command":"https://example.test/app_123"}`,
		req.GetBlocks()[2].GetData().GetPayload(),
	)
}

func TestOutputRejectsUnencodablePayload(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	err := cli.Output("apps upload", Data(func() {}))
	require.Error(t, err)
	require.False(t, Unsupported(err), "an encoding bug is not a reason to fall back")
	require.Empty(t, helper.requests, "nothing should be sent when a block cannot be encoded")
}

func TestPromptReturnsResponse(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	value, err := cli.Prompt(PromptOpts{Message: "Pick", Type: PromptSelect, Options: []string{"a", "b"}, Default: "a"})
	require.NoError(t, err)
	require.Equal(t, "chosen", value)

	require.Len(t, helper.prompts, 1)
	require.Equal(t, proto.PromptType_SELECT, helper.prompts[0].GetType())
	require.Equal(t, []string{"a", "b"}, helper.prompts[0].GetOptions())
	require.Equal(t, "a", helper.prompts[0].GetDefaultValue())
}

func TestNilHelperReportsUnavailable(t *testing.T) {
	cli := New(nil)
	require.False(t, cli.Available())

	// Every entry point must report ErrNoHelper rather than panicking, so a
	// plugin running standalone or against a v1/v2 core takes the local path.
	spinner, err := cli.ProgressStart("uploading")
	require.ErrorIs(t, err, ErrNoHelper)
	require.NotNil(t, spinner, "the handle must be usable so callers can defer Stop")

	_, promptErr := cli.Prompt(PromptOpts{Message: "Pick"})

	for name, err := range map[string]error{
		"Message":       cli.Message("m"),
		"Success":       cli.Success("m"),
		"Warn":          cli.Warn("m"),
		"Error":         cli.Error("m"),
		"Progress":      cli.Progress("m"),
		"ProgressStart": err,
		"SpinnerUpdate": spinner.Update("m"),
		"SpinnerStop":   spinner.Stop("m", true),
		"Output":        cli.Output("apps upload", Data(map[string]string{"a": "b"})),
		"Prompt":        promptErr,
	} {
		require.ErrorIs(t, err, ErrNoHelper, name)
		require.True(t, Unsupported(err), name)
	}
}

func TestUnsupportedClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"no helper", ErrNoHelper, true},
		{"error whose text merely mentions no helper", errors.New("send failed: " + ErrNoHelper.Error()), false},
		{"unimplemented", status.Error(codes.Unimplemented, "unknown method SendCommandOutput"), true},
		// Anything else may have been rendered already; falling back would
		// duplicate output.
		{"unavailable", status.Error(codes.Unavailable, "connection closed"), false},
		{"internal", status.Error(codes.Internal, "boom"), false},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, "too slow"), false},
		{"plain error", errors.New("boom"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Unsupported(tc.err))
		})
	}
}

func TestTransportFailuresAreSurfacedNotSwallowed(t *testing.T) {
	transportErr := status.Error(codes.Unavailable, "connection closed")
	helper := &fakeHelper{err: transportErr}
	cli := New(helper)

	err := cli.Output("apps upload", Data(map[string]string{"app_id": "app_123"}))
	require.ErrorIs(t, err, transportErr)
	require.False(t, Unsupported(err))
}

// --- end-to-end over gRPC ---

// newHelperOverGRPC dials a CoreCLIHelper served over an in-memory connection,
// mirroring how a plugin talks to the core CLI. When oldCore is true the server
// only registers the RPCs that existed before centralized output, so
// SendCommandOutput returns Unimplemented exactly as an older core CLI would.
func newHelperOverGRPC(t *testing.T, oldCore bool, stdout, stderr *bytes.Buffer) plugins.CoreCLIHelper {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(64*1024*1024),
		grpc.MaxSendMsgSize(64*1024*1024),
	)

	impl := &plugins.CoreCLIHelperServer{
		Impl: plugins.NewCoreCLIHelperWithWriters(
			context.Background(), nil, afero.NewMemMapFs(), rendering.FormatText, stdout, stderr,
		),
	}

	desc := proto.CoreCLIHelper_ServiceDesc
	if oldCore {
		desc = withoutMethods(desc, "SendCommandOutput")
	}
	server.RegisterService(&desc, impl)

	go func() {
		// Serve returns when the listener closes during cleanup.
		_ = server.Serve(listener)
	}()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		conn.Close()
		server.Stop()
		listener.Close()
	})

	return plugins.NewCoreCLIHelperClient(proto.NewCoreCLIHelperClient(conn))
}

// withoutMethods returns a copy of desc with the named methods removed.
func withoutMethods(desc grpc.ServiceDesc, names ...string) grpc.ServiceDesc {
	removed := make(map[string]bool, len(names))
	for _, name := range names {
		removed[name] = true
	}

	methods := make([]grpc.MethodDesc, 0, len(desc.Methods))
	for _, m := range desc.Methods {
		if !removed[m.MethodName] {
			methods = append(methods, m)
		}
	}
	desc.Methods = methods
	return desc
}

func TestEndToEndAgainstNewCore(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cli := New(newHelperOverGRPC(t, false, stdout, stderr))

	require.NoError(t, cli.Message("starting upload"))

	spinner, err := cli.ProgressStart("uploading")
	require.NoError(t, err)
	require.NoError(t, spinner.Stop("uploaded", true))

	require.NoError(t, cli.Output("apps upload",
		Data(map[string]string{"app_id": "app_123"}),
		Warning("Your app ID is permanent"),
		NextStep("View status", "https://example.test/app_123"),
	))

	require.Equal(t, strings.Join([]string{
		"starting upload",
		"uploading",
		"✔ uploaded",
		"  app_id: app_123",
		"⚠ Your app ID is permanent",
		"→ View status: https://example.test/app_123",
		"",
	}, "\n"), stdout.String())
	require.Empty(t, stderr.String())
}

// A payload larger than gRPC's 4MB default must cross the helper channel intact.
func TestEndToEndLargeCommandOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	cli := New(newHelperOverGRPC(t, false, stdout, &bytes.Buffer{}))

	blob := strings.Repeat("z", 6<<20) // 6MB
	require.NoError(t, cli.Output("apps upload", Data(map[string]string{"blob": blob})))

	require.Equal(t, "  blob: "+blob+"\n", stdout.String())
}

// An older core CLI has no SendCommandOutput, so every output call returns
// Unimplemented and the plugin knows to render locally instead. Because all
// output rides one RPC, that answer is a single, consistent capability signal:
// there is no host that can render some kinds of output but not others.
func TestEndToEndAgainstOldCore(t *testing.T) {
	stdout := &bytes.Buffer{}
	cli := New(newHelperOverGRPC(t, true, stdout, &bytes.Buffer{}))

	for name, err := range map[string]error{
		"Message":  cli.Message("starting upload"),
		"Progress": cli.Progress("uploading"),
		"Output":   cli.Output("apps upload", Data(map[string]string{"app_id": "app_123"})),
	} {
		require.Error(t, err, name)
		require.Equal(t, codes.Unimplemented, status.Code(err), name)
		require.True(t, Unsupported(err), name)
	}

	// The core rendered nothing, which is what makes local rendering safe.
	require.Empty(t, stdout.String())
}
