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

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"
)

// Config provides the configuration for the RPC service.
type Config struct {
	// Port is the port number to listen to on localhost
	Port int

	// Info, error, etc. logger. Unrelated to API request logs.
	Log *log.Logger

	// UserCfg is the Stripe CLI config of the user
	UserCfg *config.Config
}

// RPCService implements the gRPC interface and starts the gRPC server.
type RPCService struct {
	cfg *Config

	grpcServer *grpc.Server

	// TelemetryClient to use for sending telemetry events
	TelemetryClient stripe.TelemetryClient
}

// ConfigOutput is the config that clients will need to connect to the gRPC server. This is printed
// out for clients to parse.
type ConfigOutput struct {
	// Host is the IP address of the gRPC server
	Host string `json:"host"`

	// Port is port number of the gRPC server
	Port int `json:"port"`
}

// New creates a new RPC service
func New(
	cfg *Config,
	telemetryClient stripe.TelemetryClient,
) *RPCService {
	if cfg.Log == nil {
		cfg.Log = &log.Logger{Out: ioutil.Discard}
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(serverUnaryInterceptor),
		grpc.StreamInterceptor(serverStreamInterceptor),
	)

	return &RPCService{
		cfg:             cfg,
		grpcServer:      grpcServer,
		TelemetryClient: telemetryClient,
	}
}

// Run starts a gRPC server on localhost
func (srv *RPCService) Run(ctx context.Context) {
	lis := srv.createListener()

	addr, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		srv.cfg.Log.Fatalf("Failed to get the TCP address of the gRPC server")
	}
	srv.printConfig(ConfigOutput{
		Host: addr.IP.String(),
		Port: addr.Port,
	})

	rpc.RegisterStripeCLIServer(srv.grpcServer, srv)

	if err := srv.grpcServer.Serve(lis); err != nil {
		srv.cfg.Log.Fatalf("Failed to serve gRPC server on %s: %v", lis.Addr().String(), err)
	}
}

func (srv *RPCService) createListener() net.Listener {
	// if port is 0, an available port is automatically chosen
	address := fmt.Sprintf("[%s]:%d", net.IPv6loopback.String(), srv.cfg.Port)

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
	return lis
}

func (srv *RPCService) printConfig(configOutput ConfigOutput) {
	if configOutputMarshalled, err := json.Marshal(configOutput); err != nil {
		srv.cfg.Log.Fatalf("Failed to write server config to stderr: %v", err)
	} else {
		fmt.Println(string(configOutputMarshalled))
	}
}
