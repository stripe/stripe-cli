package rpcservice

import (
	"context"
	"fmt"

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

// Only allow requests from clients that have the required header. This helps prevent malicious
// websites from making requests. See https://fetch.spec.whatwg.org/#forbidden-header-name
func authorize(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Retrieving metadata failed")
	}

	if _, ok := md[requiredHeader]; !ok {
		return status.Errorf(codes.Unauthenticated, fmt.Sprintf("%s header is not supplied", requiredHeader))
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

	telemetryMetadata := stripe.NewEventMetadata()
	telemetryMetadata.SetMerchant(merchant)
	telemetryMetadata.CommandPath = methodName

	newCtx := stripe.WithEventMetadata(stripe.WithTelemetryClient(ctx, server.TelemetryClient), telemetryMetadata)
	return newCtx
}

// Middleware for stream requests
func serverStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.WithFields(log.Fields{
		"prefix": "gRPC",
	}).Debugf("Streaming method invoked: %v", info.FullMethod)
	wrappedStream := newWrappedStream(stream, info.FullMethod, srv.(*RPCService))
	if err := authorize(wrappedStream.Context()); err != nil {
		return err
	}
	return handler(srv, wrappedStream)
}

// Middleware for unary requests
func serverUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log.WithFields(log.Fields{
		"prefix": "gRPC",
	}).Debugf("Unary method invoked: %v, req: %v", info.FullMethod, req)
	newCtx := updateContextWithTelemetry(ctx, info.FullMethod, info.Server.(*RPCService))
	if err := authorize(newCtx); err != nil {
		return nil, err
	}
	return handler(newCtx, req)
}
