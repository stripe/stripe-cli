package plugins

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

// CoreCLIHelper is the interface that's implemented by the host and called by the plugin.
type CoreCLIHelper interface {
	Echo(input string) (string, error)
}

type CoreCLIHelperClient struct {
	client proto.CoreCLIHelperClient
}

func (c *CoreCLIHelperClient) Echo(input string) (string, error) {
	resp, err := c.client.Echo(context.Background(), &proto.EchoRequest{Input: input})
	if err != nil {
		return "", err
	}
	return resp.Output, nil
}

type CoreCLIHelperServer struct {
	proto.CoreCLIHelperServer
	Impl CoreCLIHelper
}

func (s *CoreCLIHelperServer) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	output, err := s.Impl.Echo(req.Input)
	if err != nil {
		return nil, err
	}
	return &proto.EchoResponse{Output: output}, nil
}

// coreCLIHelper is the real implementation of the CoreCLIHelper interface.
type coreCLIHelper struct{}

var _ CoreCLIHelper = &coreCLIHelper{}

// Echo echoes the input string.
func (h *coreCLIHelper) Echo(input string) (string, error) {
	fmt.Printf("[ECHO] %s\n", input)
	return input, nil
}
