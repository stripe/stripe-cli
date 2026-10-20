package rpcservice

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"testing"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024
const testDaemonToken = "daemon-authentication-test-token"

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	srv, err := New(&Config{
		UserCfg: &config.Config{
			Profile: config.Profile{
				APIKey:     "sk_test_12345",
				DeviceName: "rpc_test_device_name",
			},
		},
	}, nil)
	if err != nil {
		panic(err)
	}
	srv.token = testDaemonToken

	rpc.RegisterStripeCLIServer(srv.grpcServer, srv)

	go func() {
		if err := srv.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func TestDaemonTokenInitializationFailure(t *testing.T) {
	for _, random := range [][]byte{nil, make([]byte, 31)} {
		srv, err := newRPCService(&Config{}, nil, bytes.NewReader(random))
		require.ErrorContains(t, err, "initialize daemon authentication")
		require.Nil(t, srv)
	}
}

func TestDaemonTokenInitialization(t *testing.T) {
	first, err := New(&Config{}, nil)
	require.NoError(t, err)
	defer first.grpcServer.Stop()
	second, err := New(&Config{}, nil)
	require.NoError(t, err)
	defer second.grpcServer.Stop()
	decoded, err := hex.DecodeString(first.token)
	require.NoError(t, err)
	require.Len(t, decoded, 32)
	require.NotEqual(t, first.token, second.token)
	require.Error(t, first.authorize(context.Background()))
	require.Error(t, first.authorize(metadata.NewIncomingContext(context.Background(), metadata.Pairs(requiredHeader, second.token))))
	require.Error(t, (&RPCService{}).authorize(metadata.NewIncomingContext(context.Background(), metadata.Pairs(requiredHeader, ""))))
}

func TestConfigOutputContainsTokenOnlyOnStdout(t *testing.T) {
	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	logger.SetLevel(logrus.TraceLevel)
	srv, err := New(&Config{Log: logger}, nil)
	require.NoError(t, err)
	defer srv.grpcServer.Stop()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	defer reader.Close()
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	srv.printConfig(ConfigOutput{Host: "::1", Port: 1234, Token: srv.token})
	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	var got ConfigOutput
	require.NoError(t, json.Unmarshal(output, &got))
	require.Equal(t, ConfigOutput{Host: "::1", Port: 1234, Token: srv.token}, got)
	require.NotContains(t, logs.String(), srv.token)
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func withAuth(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{requiredHeader: testDaemonToken})
	return metadata.NewOutgoingContext(ctx, md)
}
