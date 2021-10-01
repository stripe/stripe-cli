package rpcservice

import (
	"context"
	"log"
	"net"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	srv := New(&Config{
		UserCfg: &config.Config{
			Profile: config.Profile{
				APIKey:     "sk_test_12345",
				DeviceName: "rpc_test_device_name",
			},
		},
	}, nil)

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
