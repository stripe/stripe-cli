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
	// PostInstall is called best-effort after the plugin is installed or upgraded.
	PostInstall(additionalInfo *proto.AdditionalInfo, version string, previousVersion string, coreCLIHelper CoreCLIHelper) error
	// PreUninstall is called best-effort before the plugin is uninstalled.
	PreUninstall(additionalInfo *proto.AdditionalInfo, version string, coreCLIHelper CoreCLIHelper) error
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
	brokerID, stop := m.serveCoreCLIHelper(coreCLIHelper)

	_, err := m.client.RunCommand(context.Background(), &proto.RunCommandRequest{
		AdditionalInfo:  additionalInfo,
		Args:            args,
		CoreCliHelperId: brokerID,
	})
	if err != nil {
		return err
	}

	stop()
	return nil
}

// PostInstall calls the RPC.
func (m *GRPCClientV3) PostInstall(additionalInfo *proto.AdditionalInfo, version string, previousVersion string, coreCLIHelper CoreCLIHelper) error {
	brokerID, stop := m.serveCoreCLIHelper(coreCLIHelper)

	_, err := m.client.PostInstall(context.Background(), &proto.PostInstallRequest{
		AdditionalInfo:  additionalInfo,
		Version:         version,
		PreviousVersion: previousVersion,
		CoreCliHelperId: brokerID,
	})
	if err != nil {
		return err
	}

	stop()
	return nil
}

// PreUninstall calls the RPC.
func (m *GRPCClientV3) PreUninstall(additionalInfo *proto.AdditionalInfo, version string, coreCLIHelper CoreCLIHelper) error {
	brokerID, stop := m.serveCoreCLIHelper(coreCLIHelper)

	_, err := m.client.PreUninstall(context.Background(), &proto.PreUninstallRequest{
		AdditionalInfo:  additionalInfo,
		Version:         version,
		CoreCliHelperId: brokerID,
	})
	if err != nil {
		return err
	}

	stop()
	return nil
}

// serveCoreCLIHelper starts a CoreCLIHelper gRPC server on a new broker stream so the
// plugin can call back into the host, returning the broker ID to send with the request
// and a func to stop the server once the call completes.
func (m *GRPCClientV3) serveCoreCLIHelper(coreCLIHelper CoreCLIHelper) (uint32, func()) {
	coreCLIHelperServer := &CoreCLIHelperServer{Impl: coreCLIHelper}

	// serverCh hands off the *grpc.Server once AcceptAndServe's callback constructs it;
	// done closes when AcceptAndServe returns without ever calling the callback (e.g.
	// broker.Accept failed). stop() below selects on both so it never touches a server
	// that was never created, instead of racing a shared variable that's written from
	// this goroutine and read from the caller's.
	serverCh := make(chan *grpc.Server, 1)
	done := make(chan struct{})
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		proto.RegisterCoreCLIHelperServer(s, coreCLIHelperServer)
		serverCh <- s
		return s
	}

	brokerID := m.broker.NextId()
	go func() {
		defer close(done)
		m.broker.AcceptAndServe(brokerID, serverFunc)
	}()

	stop := func() {
		select {
		case s := <-serverCh:
			s.Stop()
		case <-done:
		}
	}

	return brokerID, stop
}

// GRPCServerV3 is the gRPC server that GRPCClientV3 talks to.
type GRPCServerV3 struct {
	proto.MainServer
	Impl   DispatcherV3
	broker *hcplugin.GRPCBroker
}

// RunCommand takes the incoming RPC request and calls the real implementation.
func (m *GRPCServerV3) RunCommand(ctx context.Context, req *proto.RunCommandRequest) (*proto.RunCommandResponse, error) {
	c, closeConn, err := m.dialCoreCLIHelper(req.CoreCliHelperId)
	if err != nil {
		return nil, err
	}
	defer closeConn()

	if err := m.Impl.RunCommand(req.AdditionalInfo, req.Args, c); err != nil {
		return nil, err
	}
	return &proto.RunCommandResponse{}, nil
}

// PostInstall takes the incoming RPC request and calls the real implementation.
func (m *GRPCServerV3) PostInstall(ctx context.Context, req *proto.PostInstallRequest) (*proto.PostInstallResponse, error) {
	c, closeConn, err := m.dialCoreCLIHelper(req.CoreCliHelperId)
	if err != nil {
		return nil, err
	}
	defer closeConn()

	if err := m.Impl.PostInstall(req.AdditionalInfo, req.Version, req.PreviousVersion, c); err != nil {
		return nil, err
	}
	return &proto.PostInstallResponse{}, nil
}

// PreUninstall takes the incoming RPC request and calls the real implementation.
func (m *GRPCServerV3) PreUninstall(ctx context.Context, req *proto.PreUninstallRequest) (*proto.PreUninstallResponse, error) {
	c, closeConn, err := m.dialCoreCLIHelper(req.CoreCliHelperId)
	if err != nil {
		return nil, err
	}
	defer closeConn()

	if err := m.Impl.PreUninstall(req.AdditionalInfo, req.Version, c); err != nil {
		return nil, err
	}
	return &proto.PreUninstallResponse{}, nil
}

// dialCoreCLIHelper connects back to the host's CoreCLIHelper server over the given broker ID.
func (m *GRPCServerV3) dialCoreCLIHelper(brokerID uint32) (*CoreCLIHelperClient, func(), error) {
	conn, err := m.broker.Dial(brokerID)
	if err != nil {
		return nil, nil, err
	}

	return &CoreCLIHelperClient{client: proto.NewCoreCLIHelperClient(conn)}, func() { conn.Close() }, nil
}
