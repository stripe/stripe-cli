package rpcservice

import (
	"context"
	"crypto/subtle"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/stripe"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requiredHeader = "sec-x-stripe-cli"

// WrappedServerStream wraps a ServerSteam so that we can pass values through context.
// https://pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware#hdr-Writing_Your_Own
type WrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the context for this stream.
func (w WrappedServerStream) Context() context.Context {
	return w.ctx
}

func newWrappedStream(stream grpc.ServerStream, methodName string, server *RPCService) grpc.ServerStream {
	newCtx := updateContextWithTelemetry(stream.Context(), methodName, server)
	return &WrappedServerStream{stream, newCtx}
}

// authorize requires the secret issued by this daemon instance.
func (srv *RPCService) authorize(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(requiredHeader)
	if len(values) != 1 || values[0] == "" || srv.token == "" ||
		subtle.ConstantTimeCompare([]byte(values[0]), []byte(srv.token)) != 1 {
		return status.Error(codes.Unauthenticated, "Invalid daemon credentials")
	}
	return nil
}

// Populate the context with:
// 1. The telemetry client from the RPC Service
// 2. The event metadata
func updateContextWithTelemetry(ctx context.Context, methodName string, server *RPCService) context.Context {
	// If the context is nil for whatever reason, create an empty one
	if ctx == nil {
		ctx = context.Background()
	}

	// if getting the config errors, don't fail running the command
	merchant, _ := server.cfg.UserCfg.Profile.GetAccountID()
	useragent := getUserAgentFromGrpcMetadata(ctx)

	telemetryMetadata := stripe.NewEventMetadata()
	telemetryMetadata.SetMerchant(merchant)
	telemetryMetadata.SetCommandPath(methodName)
	telemetryMetadata.SetUserAgent(useragent)

	newCtx := stripe.WithEventMetadata(stripe.WithTelemetryClient(ctx, server.TelemetryClient), telemetryMetadata)
	return newCtx
}

// Use the telemetry client in context to send a telemetry event of the method invocation.
func sendCommandInvocationEvent(ctx context.Context) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient != nil {
		go telemetryClient.SendEvent(ctx, "Command Invoked", "gRPC")
	}
}

func getUserAgentFromGrpcMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.Join(md["user-agent"], ",")
}

// Middleware for stream requests
func serverStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	server := srv.(*RPCService)
	if err := server.authorize(stream.Context()); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"prefix": "gRPC",
	}).Debugf("Streaming method invoked: %v", info.FullMethod)
	wrappedStream := newWrappedStream(stream, info.FullMethod, server)
	sendCommandInvocationEvent(wrappedStream.Context())
	return handler(srv, wrappedStream)
}

// Middleware for unary requests
func serverUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	server := info.Server.(*RPCService)
	if err := server.authorize(ctx); err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{
		"prefix": "gRPC",
	}).Debugf("Unary method invoked: %v, req: %v", info.FullMethod, req)
	newCtx := updateContextWithTelemetry(ctx, info.FullMethod, server)
	go sendCommandInvocationEvent(newCtx)
	return handler(newCtx, req)
}
