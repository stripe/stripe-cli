package rpcservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"syscall"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/stripe/stripe-cli/rpc"
)

// Config provides the configuration for the RPC service
type Config struct {
	// Port is the port number to listen to on localhost
	Port int

	// Info, error, etc. logger. Unrelated to API request logs.
	Log *log.Logger
}

// RPCService is the gRPC server and implements the protobuf service
type RPCService struct {
	cfg *Config
}

// ConfigOutput is the server config written to stderr for clients to read
type ConfigOutput struct {
	// Address is the localhost address of the gRPC server
	Address string `json:"address"`
}

// New creates a new RPC service
func New(cfg *Config) *RPCService {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	return &RPCService{
		cfg: cfg,
	}
}

// Run starts a gRPC server on localhost
func (srv *RPCService) Run(ctx context.Context) {
	address := "127.0.0.1:"
	if srv.cfg.Port != 0 {
		address = fmt.Sprintf("%s%d", address, srv.cfg.Port)
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		// Invalid port, such as "Foo". This case should be handled by cobra and never be reached.
		var dnsError *net.DNSError
		if errors.As(err, &dnsError) {
			srv.cfg.Log.Fatalf("Failed to listen on %s. %s is an unknown name.", address, dnsError.Name)
		}

		// Invalid port number, such as -1
		var addrError *net.AddrError
		if errors.As(err, &addrError) {
			srv.cfg.Log.Fatalf("Failed to listen on %s. %s is an invalid port.", address, addrError.Addr)
		}

		// Port is already in use
		var syscallErr *os.SyscallError
		if errors.As(err, &syscallErr) && errors.Is(syscallErr.Err, syscall.EADDRINUSE) {
			srv.cfg.Log.Fatalf("Failed to listen on %s. Port is already in use.", address)
		}
		srv.cfg.Log.Fatalf("Failed to listen on %s. Unexpected error: %v", address, err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(serverUnaryInterceptor),
		grpc.StreamInterceptor(serverStreamInterceptor),
	)

	rpc.RegisterStripeCLIServer(grpcServer, srv)

	configOutput, err := json.Marshal(ConfigOutput{
		Address: lis.Addr().String(),
	})
	if err != nil {
		srv.cfg.Log.Fatalf("Failed to write server config to stderr: %v", err)
	}

	fmt.Fprintln(os.Stderr, string(configOutput))

	if err := grpcServer.Serve(lis); err != nil {
		srv.cfg.Log.Fatalf("Failed to serve gRPC server on %s: %v", lis.Addr().String(), err)
	}
}