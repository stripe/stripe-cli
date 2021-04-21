package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/stripe/stripe-cli/rpc"
)

// Config provides the configuration for the gRPC server
type Config struct {
	// Port is the port number to listen to on localhost
	Port int

	// Info, error, etc. logger. Unrelated to API request logs.
	Log *log.Logger
}

// RPCServer is the gRPC server
type RPCServer struct {
	cfg *Config
}

// ConfigOutput is the server config written to stderr for clients to read
type ConfigOutput struct {
	// Address is the localhost address of the gRPC server
	Address string `json:"address"`
}

// New creates a new gRPC server
func New(cfg *Config) *RPCServer {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	return &RPCServer{
		cfg: cfg,
	}
}

func withSIGTERMCancel(ctx context.Context, onCancel func()) context.Context {
	// Create a context that will be canceled when Ctrl+C is pressed
	ctx, cancel := context.WithCancel(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interruptCh
		onCancel()
		cancel()
	}()
	return ctx
}

// Run starts the gRPC server
func (srv *RPCServer) Run(ctx context.Context) error {
	ctx = withSIGTERMCancel(ctx, func() {
		log.WithFields(log.Fields{
			"prefix": "logtailing.Tailer.Run",
		}).Debug("Ctrl+C received, cleaning up...")
	})

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
		srv.cfg.Log.Fatalf("Failed write server config to stderr: %v", err)
	}

	fmt.Fprintln(os.Stderr, string(configOutput))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			srv.cfg.Log.Fatalf("Failed to serve gRPC server %s: %v", lis.Addr().String(), err)
		}
	}()

	for range ctx.Done() {
	}

	return nil
}
