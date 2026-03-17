package plugins

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// CoreCLIHelper is the interface that's implemented by the host and called by the plugin.
type CoreCLIHelper interface {
	Echo(input string) (string, error)
	SendAnalytics(eventName string, eventValue string) error
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

func (c *CoreCLIHelperClient) SendAnalytics(eventName string, eventValue string) error {
	_, err := c.client.SendAnalytics(context.Background(), &proto.SendAnalyticsRequest{
		EventName:  eventName,
		EventValue: eventValue,
	})
	return err
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

func (s *CoreCLIHelperServer) SendAnalytics(ctx context.Context, req *proto.SendAnalyticsRequest) (*proto.SendAnalyticsResponse, error) {
	err := s.Impl.SendAnalytics(req.EventName, req.EventValue)
	if err != nil {
		return nil, err
	}
	return &proto.SendAnalyticsResponse{}, nil
}

// coreCLIHelper is the real implementation of the CoreCLIHelper interface.
type coreCLIHelper struct {
	ctx context.Context
}

var _ CoreCLIHelper = &coreCLIHelper{}

// NewCoreCLIHelper creates a new CoreCLIHelper with the given context.
func NewCoreCLIHelper(ctx context.Context) CoreCLIHelper {
	return &coreCLIHelper{ctx: ctx}
}

// Echo echoes the input string.
func (h *coreCLIHelper) Echo(input string) (string, error) {
	fmt.Printf("[ECHO] %s\n", input)
	return input, nil
}

// SendAnalytics sends a telemetry event to the analytics service.
func (h *coreCLIHelper) SendAnalytics(eventName string, eventValue string) error {
	// Get the telemetry client from the context
	telemetryClient := stripe.GetTelemetryClient(h.ctx)
	if telemetryClient == nil {
		// If no telemetry client is available, silently skip
		return nil
	}

	// Send the event via the telemetry client
	telemetryClient.SendEvent(h.ctx, eventName, eventValue)
	return nil
}
