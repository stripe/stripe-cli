package rpcservice

import (
	"context"
	"net"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	srv := New(&Config{})

	rpc.RegisterStripeCLIServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestVersionReturnsCLIVersion(t *testing.T) {
	ctx := context.Background()
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
