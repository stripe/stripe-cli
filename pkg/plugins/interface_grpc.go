package plugins

import (
	"context"

	hcplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// DispatcherGRPC is the interface that's implemented by the plugin and used by the host.
type DispatcherGRPC interface {
	RunCommand(additionalInfo *proto.AdditionalInfo, args []string) error
}

// CLIPluginGRPC is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type CLIPluginGRPC struct {
	// GRPCPlugin must still implement the Plugin interface
	hcplugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl DispatcherGRPC
}

// GRPCServer creates the GRPC server.
func (p *CLIPluginGRPC) GRPCServer(broker *hcplugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterMainServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient creates the GRPC client.
func (p *CLIPluginGRPC) GRPCClient(ctx context.Context, broker *hcplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewMainClient(c)}, nil
}

// GRPCClient is an implementation of the gRPC client that talks over gRPC.
type GRPCClient struct {
	client proto.MainClient
}

// RunCommand calls the RPC.
func (m *GRPCClient) RunCommand(additionalInfo *proto.AdditionalInfo, args []string) error {
	_, err := m.client.RunCommand(context.Background(), &proto.RunCommandRequest{
		AdditionalInfo: additionalInfo,
		Args:           args,
	})
	if err != nil {
		return err
	}

	return nil
}

// GRPCServer is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	proto.MainServer
	// This is the real implementation
	Impl DispatcherGRPC
}

// RunCommand takes the incoming RPC request and calls the real implementation.
func (m *GRPCServer) RunCommand(ctx context.Context, req *proto.RunCommandRequest) (*proto.RunCommandResponse, error) {
	err := m.Impl.RunCommand(req.AdditionalInfo, req.Args)
	if err != nil {
		return nil, err
	}
	return &proto.RunCommandResponse{}, nil
}
