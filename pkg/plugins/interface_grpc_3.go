package plugins

import (
	"context"

	hcplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// DispatcherV3 is the interface that's implemented by the plugin and used by the host.
type DispatcherV3 interface {
	RunCommand(additionalInfo *proto.AdditionalInfo, args []string, coreCLIHelper CoreCLIHelper) error
}

// CLIPluginV3 is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type CLIPluginV3 struct {
	hcplugin.Plugin
	Impl DispatcherV3
}

// GRPCServer creates the GRPC server.
func (p *CLIPluginV3) GRPCServer(broker *hcplugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterMainServer(s, &GRPCServerV3{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

// GRPCClient creates the GRPC client.
func (p *CLIPluginV3) GRPCClient(ctx context.Context, broker *hcplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClientV3{
		client: proto.NewMainClient(c),
		broker: broker,
	}, nil
}

// GRPCClientV3 is an implementation of the gRPC client that talks over gRPC.
type GRPCClientV3 struct {
	client proto.MainClient
	broker *hcplugin.GRPCBroker
}

// RunCommand calls the RPC.
func (m *GRPCClientV3) RunCommand(additionalInfo *proto.AdditionalInfo, args []string, coreCLIHelper CoreCLIHelper) error {
	coreCLIHelperServer := &CoreCLIHelperServer{Impl: coreCLIHelper}

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		proto.RegisterCoreCLIHelperServer(s, coreCLIHelperServer)
		return s
	}

	brokerID := m.broker.NextId()
	go m.broker.AcceptAndServe(brokerID, serverFunc)

	_, err := m.client.RunCommand(context.Background(), &proto.RunCommandRequest{
		AdditionalInfo:  additionalInfo,
		Args:            args,
		CoreCliHelperId: brokerID,
	})
	if err != nil {
		return err
	}

	s.Stop()
	return nil
}

// GRPCServerV3 is the gRPC server that GRPCClientV3 talks to.
type GRPCServerV3 struct {
	proto.MainServer
	Impl   DispatcherV3
	broker *hcplugin.GRPCBroker
}

// RunCommand takes the incoming RPC request and calls the real implementation.
func (m *GRPCServerV3) RunCommand(ctx context.Context, req *proto.RunCommandRequest) (*proto.RunCommandResponse, error) {
	conn, err := m.broker.Dial(req.CoreCliHelperId)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := &CoreCLIHelperClient{client: proto.NewCoreCLIHelperClient(conn)}

	err = m.Impl.RunCommand(req.AdditionalInfo, req.Args, c)
	if err != nil {
		return nil, err
	}
	return &proto.RunCommandResponse{}, nil
}
