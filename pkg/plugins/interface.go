package plugins

import (
	"net/rpc"

	hcplugin "github.com/hashicorp/go-plugin"
)

// Server -----------------------------------------------

// Dispatcher is the interface that we're exposing as a plugin.
// It is named so because it is able to dispatch a command from the main CLI to the plugin
type Dispatcher interface {
	RunCommand(args []string) (string, error)
}

// DispatcherRPCServer is the RPC server that a plugin talks to, conforming to
// the requirements of net/rpc
type DispatcherRPCServer struct {
	// This is the real implementation
	Impl Dispatcher
}

// RunCommand is the main entry command that can be invoked remotely by the Stripe CLI
// it is defined here on the plugin's RPC server
// then we call the internal RunCommand method
// finally, we then return the response back via the DispatcherRPC interface that the CLI is interacting with
func (s *DispatcherRPCServer) RunCommand(args []string, resp *string) error {
	var err error
	*resp, err = s.Impl.RunCommand(args)
	return err
}

// CLI Client ---------------------------------------------------

// PluginClient is an implementation that talks over RPC
type PluginClient struct {
	client *rpc.Client
}

// RunCommand is the main plugin command that can be invoked remotely by the Stripe CLI
// we expose the command here for the CLI to call, which then calls the method directly on the RPCServer
func (g *PluginClient) RunCommand(args []string) (string, error) {
	var resp string
	err := g.client.Call("Plugin.RunCommand", args, &resp)
	if err != nil {
		return "", err
	}

	return resp, nil
}

// Plugin --------------------------------------------------------

// CLIPluginV1 is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a DispatcherRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return a PluginClient for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on a
// plugin connection and is a more advanced use case.
type CLIPluginV1 struct {
	// Impl Injection
	Impl Dispatcher
}

// Server returns the rpc server
func (p *CLIPluginV1) Server(*hcplugin.MuxBroker) (interface{}, error) {
	return &DispatcherRPCServer{Impl: p.Impl}, nil
}

// Client returns the rpc client
func (CLIPluginV1) Client(b *hcplugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginClient{client: c}, nil
}
