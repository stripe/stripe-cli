package rpcservice

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
)

func TestAuthorizeAllowsIfHeaderPresent(t *testing.T) {
	md := make(metadata.MD)
	md["sec-x-stripe-cli"] = []string{}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	assert.Equal(t, nil, authorize(ctx))
}

func TestAuthorizeRejectsIfHeaderAbsent(t *testing.T) {
	md := make(metadata.MD)
	md["foo-bar"] = []string{}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	expected := status.Errorf(codes.Unauthenticated, fmt.Sprintf("%s header is not supplied", requiredHeader))
	assert.Equal(t, expected, authorize(ctx))
}
