package rpcservice

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requiredHeader = "sec-x-stripe-cli"

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
	if err := authorize(stream.Context()); err != nil {
		return err
	}
	return handler(srv, stream)
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
	if err := authorize(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}
