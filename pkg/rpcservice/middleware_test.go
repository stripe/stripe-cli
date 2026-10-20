package rpcservice

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

func TestAllRPCsRequireDaemonAuthentication(t *testing.T) {
	srv, err := New(&Config{}, nil)
	require.NoError(t, err)
	defer srv.grpcServer.Stop()
	listener := bufconn.Listen(bufSize)
	defer listener.Close()
	rpc.RegisterStripeCLIServer(srv.grpcServer, srv)
	go srv.grpcServer.Serve(listener)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	methods := rpc.File_commands_proto.Services().ByName("StripeCLI").Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		for _, tt := range []struct {
			name, token string
		}{
			{"missing", ""},
			{"wrong", "wrong-token"},
			{"legacy", "1"},
		} {
			t.Run(string(method.Name())+"/"+tt.name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if tt.token != "" {
					ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(requiredHeader, tt.token))
				}
				fullMethod := "/rpc.StripeCLI/" + string(method.Name())
				request := dynamicpb.NewMessage(method.Input())
				response := dynamicpb.NewMessage(method.Output())
				if method.IsStreamingServer() {
					stream, err := conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, fullMethod)
					require.NoError(t, err)
					require.NoError(t, stream.SendMsg(request))
					require.NoError(t, stream.CloseSend())
					err = stream.RecvMsg(response)
					require.Equal(t, codes.Unauthenticated, status.Code(err))
					require.Equal(t, "Invalid daemon credentials", status.Convert(err).Message())
				} else {
					err := conn.Invoke(ctx, fullMethod, request, response)
					require.Equal(t, codes.Unauthenticated, status.Code(err))
					require.Equal(t, "Invalid daemon credentials", status.Convert(err).Message())
				}
			})
		}
	}
}

func TestDaemonAuthentication(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	for _, tt := range []struct {
		name   string
		values []string
		code   codes.Code
	}{
		{"correct", []string{testDaemonToken}, codes.OK},
		{"missing", nil, codes.Unauthenticated},
		{"empty", []string{""}, codes.Unauthenticated},
		{"wrong", []string{"wrong-token"}, codes.Unauthenticated},
		{"same length wrong", []string{"Daemon-authentication-test-token"}, codes.Unauthenticated},
		{"legacy", []string{"1"}, codes.Unauthenticated},
		{"multiple", []string{testDaemonToken, testDaemonToken}, codes.Unauthenticated},
		{"correct then wrong", []string{testDaemonToken, "wrong-token"}, codes.Unauthenticated},
		{"wrong then correct", []string{"wrong-token", testDaemonToken}, codes.Unauthenticated},
	} {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.MD{}
			md.Set(requiredHeader, tt.values...)
			ctx := metadata.NewOutgoingContext(context.Background(), md)
			_, err := client.Version(ctx, &rpc.VersionRequest{})
			assert.Equal(t, tt.code, status.Code(err))

			server := &RPCService{token: testDaemonToken}
			called := false
			stream := &WrappedServerStream{ctx: metadata.NewIncomingContext(context.Background(), md)}
			if tt.code == codes.OK {
				server.cfg = &Config{UserCfg: &config.Config{}}
			}
			err = serverStreamInterceptor(server, stream, &grpc.StreamServerInfo{}, func(interface{}, grpc.ServerStream) error {
				called = true
				return nil
			})
			assert.Equal(t, tt.code, status.Code(err))
			assert.Equal(t, tt.code == codes.OK, called)
			if tt.code != codes.OK {
				assert.Equal(t, "Invalid daemon credentials", status.Convert(err).Message())
				liveStream, err := client.Listen(ctx, &rpc.ListenRequest{Live: true})
				require.NoError(t, err)
				resp, err := liveStream.Recv()
				assert.Equal(t, codes.Unauthenticated, status.Code(err))
				assert.Nil(t, resp)
				assert.Equal(t, "Invalid daemon credentials", status.Convert(err).Message())
			}
		})
	}
}

func TestWebhookEndpointCreateRequiresAuthentication(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)
	for _, token := range []string{"", "wrong-token"} {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(requiredHeader, token))
		resp, err := client.WebhookEndpointCreate(ctx, &rpc.WebhookEndpointCreateRequest{Url: "https://example.com/webhook"})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
		require.Nil(t, resp)
	}
}

func TestAuthenticatedWebhookEndpointCreateReachesHandler(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	resp, err := rpc.NewStripeCLIClient(conn).WebhookEndpointCreate(withAuth(context.Background()), &rpc.WebhookEndpointCreateRequest{})
	require.ErrorContains(t, err, "url cannot be empty")
	require.NotEqual(t, codes.Unauthenticated, status.Code(err))
	require.Nil(t, resp)
}

func TestUnaryAuthenticationPrecedesHandlerAndTelemetry(t *testing.T) {
	server := &RPCService{token: testDaemonToken}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(requiredHeader, "wrong-token"))
	called := false
	_, err := serverUnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{Server: server}, func(context.Context, interface{}) (interface{}, error) {
		called = true
		return nil, nil
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.False(t, called)
}

func TestUpdateContextWithTelemetry(t *testing.T) {
	telemetryClient := &stripe.NoOpTelemetryClient{}
	config := &Config{
		UserCfg: &config.Config{
			Profile: config.Profile{
				AccountID: "acct_xxx",
			},
		}}
	rpcService, err := New(config, telemetryClient)
	assert.NoError(t, err)
	defer rpcService.grpcServer.Stop()

	// Add grpc Metadata to context
	md := metadata.Pairs("user-agent", "unit_test")

	ctx := metadata.NewIncomingContext(context.Background(), md)

	newCtx := updateContextWithTelemetry(ctx, "method", rpcService)

	eventMetadata := stripe.GetEventMetadata(newCtx)
	assert.NotNil(t, eventMetadata)
	assert.Equal(t, "acct_xxx", eventMetadata.Merchant)
	assert.Equal(t, "method", eventMetadata.CommandPath)
	assert.Equal(t, "unit_test", eventMetadata.UserAgent)
	assert.Equal(t, stripe.GetTelemetryClient(newCtx), telemetryClient)
}

func TestGetUserAgentFromGRPCMetadata(t *testing.T) {
	// No grpc metadata
	assert.Equal(t, "", getUserAgentFromGrpcMetadata(context.Background()))
}

func TestGetUserAgentFromGRPCMetadataWithNoUserAgent(t *testing.T) {
	// no user-agent key
	md := metadata.Pairs("hello", "world")

	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, "", getUserAgentFromGrpcMetadata(ctx))
}

func TestGetUserAgentFromGRPCMetadataWitMultipleUserAgents(t *testing.T) {
	// no user-agent key
	md := metadata.Pairs("user-agent", "unit_test")
	md.Append("user-agent", "hello")

	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, "unit_test,hello", getUserAgentFromGrpcMetadata(ctx))
}
