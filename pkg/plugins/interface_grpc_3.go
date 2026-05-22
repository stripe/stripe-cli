package plugins

import (
	"context"
	"errors"
	"sync"
	"time"

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
		broker: grpcBrokerAdapter{broker: broker},
	}, nil
}

// GRPCClientV3 is an implementation of the gRPC client that talks over gRPC.
type GRPCClientV3 struct {
	client proto.MainClient
	broker grpcBrokerClient
}

type grpcBrokerServer interface {
	AcceptAndServe(id uint32, newGRPCServer func([]grpc.ServerOption) *grpc.Server)
}

type grpcBrokerClient interface {
	grpcBrokerServer
	nextID() uint32
}

type grpcBrokerAdapter struct {
	broker *hcplugin.GRPCBroker
}

func (b grpcBrokerAdapter) nextID() uint32 {
	return b.broker.NextId()
}

func (b grpcBrokerAdapter) AcceptAndServe(id uint32, newGRPCServer func([]grpc.ServerOption) *grpc.Server) {
	b.broker.AcceptAndServe(id, newGRPCServer)
}

var errCoreCLIHelperBrokerServerStart = errors.New("failed to start CoreCLIHelper broker server")
var coreCLIHelperBrokerPublishDelay = 25 * time.Millisecond
var coreCLIHelperBrokerServerStartTimeout = 5 * time.Second

func startCoreCLIHelperBrokerServer(broker grpcBrokerServer, brokerID uint32, coreCLIHelper CoreCLIHelper) (func(), error) {
	startedCh := make(chan struct{})
	doneCh := make(chan struct{})

	var server *grpc.Server
	go func() {
		defer close(doneCh)

		broker.AcceptAndServe(brokerID, func(opts []grpc.ServerOption) *grpc.Server {
			server = grpc.NewServer(opts...)
			proto.RegisterCoreCLIHelperServer(server, &CoreCLIHelperServer{Impl: coreCLIHelper})
			close(startedCh)
			return server
		})
	}()

	select {
	case <-startedCh:
	case <-doneCh:
		return nil, errCoreCLIHelperBrokerServerStart
	case <-time.After(coreCLIHelperBrokerServerStartTimeout):
		return nil, errCoreCLIHelperBrokerServerStart
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if server != nil {
				server.Stop()
			}
		})
	}

	return cleanup, nil
}

// RunCommand calls the RPC.
func (m *GRPCClientV3) RunCommand(additionalInfo *proto.AdditionalInfo, args []string, coreCLIHelper CoreCLIHelper) error {
	brokerID := m.broker.nextID()
	errCh := make(chan error, 1)
	go func() {
		_, err := m.client.RunCommand(context.Background(), &proto.RunCommandRequest{
			AdditionalInfo:  additionalInfo,
			Args:            args,
			CoreCliHelperId: brokerID,
		})
		errCh <- err
	}()

	// Non-Go plugins can drop unsolicited broker connection info if it arrives
	// before they register a pending dial for this service ID.
	time.Sleep(coreCLIHelperBrokerPublishDelay)

	cleanup, err := startCoreCLIHelperBrokerServer(m.broker, brokerID, coreCLIHelper)
	if err != nil {
		return err
	}
	defer cleanup()

	err = <-errCh
	if err != nil {
		return err
	}

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
