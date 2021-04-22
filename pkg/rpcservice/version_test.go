package rpcservice

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	srv := New(&Config{})

	rpc.RegisterStripeCLIServer(srv.grpcServer, srv)

	go func() {
		if err := srv.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func withAuth(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{requiredHeader: "1"})
	return metadata.NewOutgoingContext(ctx, md)
}

func TestVersionReturnsCLIVersion(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	resp, err := client.Version(ctx, &rpc.VersionRequest{})
	if err != nil {
		t.Fatalf("Version failed: %v", err)
	}

	expected := rpc.VersionResponse{
		Version: "master",
	}

	assert.Equal(t, expected.Version, resp.Version)
}
