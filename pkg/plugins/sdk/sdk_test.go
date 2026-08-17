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
type fakeHelper struct {
	plugins.CoreCLIHelper

	messages []*proto.SendMessageRequest
	progress []*proto.SendProgressRequest
	outputs  []*proto.SendCommandOutputRequest
	prompts  []*proto.PromptRequest

	err error
}

func (h *fakeHelper) SendMessage(req *proto.SendMessageRequest) error {
	h.messages = append(h.messages, req)
	return h.err
}

func (h *fakeHelper) SendProgress(req *proto.SendProgressRequest) error {
	h.progress = append(h.progress, req)
	return h.err
}

func (h *fakeHelper) SendCommandOutput(req *proto.SendCommandOutputRequest) error {
	h.outputs = append(h.outputs, req)
	return h.err
}

func (h *fakeHelper) Prompt(req *proto.PromptRequest) (*proto.PromptResponse, error) {
	h.prompts = append(h.prompts, req)
	if h.err != nil {
		return nil, h.err
	}
	return &proto.PromptResponse{Value: "chosen"}, nil
}

func TestMessagesCarryLevels(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	require.NoError(t, cli.Message("info"))
	require.NoError(t, cli.Success("success"))
	require.NoError(t, cli.Warn("warning"))
	require.NoError(t, cli.Error("error"))

	require.Len(t, helper.messages, 4)
	require.Equal(t, "info", helper.messages[0].GetMessage().GetMessage())
	require.Equal(t, proto.MessageLevel_INFO, helper.messages[0].GetMessage().GetLevel())
	require.Equal(t, proto.MessageLevel_SUCCESS, helper.messages[1].GetMessage().GetLevel())
	require.Equal(t, proto.MessageLevel_WARNING, helper.messages[2].GetMessage().GetLevel())
	require.Equal(t, proto.MessageLevel_ERROR, helper.messages[3].GetMessage().GetLevel())
}

func TestProgressLifecycle(t *testing.T) {
	helper := &fakeHelper{}
	cli := New(helper)

	require.NoError(t, cli.Progress("one step"))

	spinner, err := cli.ProgressStart("uploading")
	require.NoError(t, err)
	require.NoError(t, spinner.Update("still uploading"))
	require.NoError(t, spinner.Stop("uploaded", true))

	require.Len(t, helper.progress, 4)
	require.Equal(t, proto.ProgressType_STEP, helper.progress[0].GetProgress().GetType())

	start, update, stop := helper.progress[1].GetProgress(), helper.progress[2].GetProgress(), helper.progress[3].GetProgress()
	require.Equal(t, proto.ProgressType_SPINNER_START, start.GetType())
	require.Equal(t, proto.ProgressType_SPINNER_UPDATE, update.GetType())
	require.Equal(t, proto.ProgressType_SPINNER_STOP, stop.GetType())
	require.Equal(t, "uploaded", stop.GetMessage())
	require.True(t, stop.GetSuccess())

	// All three updates address the same spinner, and its id is distinct from
	// the one-shot step's.
	require.Equal(t, start.GetId(), update.GetId())
	require.Equal(t, start.GetId(), stop.GetId())
	require.NotEqual(t, helper.progress[0].GetProgress().GetId(), start.GetId())
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

	require.Len(t, helper.outputs, 1)
	req := helper.outputs[0]
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
	require.Empty(t, helper.outputs, "nothing should be sent when a block cannot be encoded")
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
		{"unimplemented", status.Error(codes.Unimplemented, "unknown method SendMessage"), true},
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
// only registers the RPCs that existed before centralized output, so the new
// RPCs return Unimplemented exactly as an older core CLI would.
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
		desc = withoutMethods(desc, "SendMessage", "SendProgress")
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

// An older core CLI has no SendMessage/SendProgress, so those calls return
// Unimplemented and the plugin renders them locally. SendCommandOutput is
// unaffected, since it predates this change.
func TestEndToEndAgainstOldCore(t *testing.T) {
	stdout := &bytes.Buffer{}
	cli := New(newHelperOverGRPC(t, true, stdout, &bytes.Buffer{}))

	err := cli.Message("starting upload")
	require.Error(t, err)
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.True(t, Unsupported(err))

	_, err = cli.ProgressStart("uploading")
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.True(t, Unsupported(err))

	require.NoError(t, cli.Output("apps upload", Data(map[string]string{"app_id": "app_123"})))
	require.Equal(t, "  app_id: app_123\n", stdout.String())
}
