package plugins

import (
	"context"
	"fmt"

	"github.com/99designs/keyring"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins/proto"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// CoreCLIHelper is the interface that's implemented by the host and called by the plugin.
type CoreCLIHelper interface {
	Echo(input string) (string, error)
	SendAnalytics(eventName string, eventValue string) error
	KeychainGetPassword(key string) (string, bool, error)
	KeychainSetPassword(key string, value string) error
	KeychainDeletePassword(key string) (bool, error)
	KeychainFindCredentials() ([]string, error)
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

func (c *CoreCLIHelperClient) KeychainGetPassword(key string) (string, bool, error) {
	resp, err := c.client.KeychainGetPassword(context.Background(), &proto.KeychainGetPasswordRequest{Key: key})
	if err != nil {
		return "", false, err
	}
	return resp.Value, resp.Found, nil
}

func (c *CoreCLIHelperClient) KeychainSetPassword(key string, value string) error {
	_, err := c.client.KeychainSetPassword(context.Background(), &proto.KeychainSetPasswordRequest{
		Key:   key,
		Value: value,
	})
	return err
}

func (c *CoreCLIHelperClient) KeychainDeletePassword(key string) (bool, error) {
	resp, err := c.client.KeychainDeletePassword(context.Background(), &proto.KeychainDeletePasswordRequest{Key: key})
	if err != nil {
		return false, err
	}
	return resp.Deleted, nil
}

func (c *CoreCLIHelperClient) KeychainFindCredentials() ([]string, error) {
	resp, err := c.client.KeychainFindCredentials(context.Background(), &proto.KeychainFindCredentialsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Keys, nil
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

func (s *CoreCLIHelperServer) KeychainGetPassword(ctx context.Context, req *proto.KeychainGetPasswordRequest) (*proto.KeychainGetPasswordResponse, error) {
	value, found, err := s.Impl.KeychainGetPassword(req.Key)
	if err != nil {
		return nil, err
	}
	return &proto.KeychainGetPasswordResponse{Value: value, Found: found}, nil
}

func (s *CoreCLIHelperServer) KeychainSetPassword(ctx context.Context, req *proto.KeychainSetPasswordRequest) (*proto.KeychainSetPasswordResponse, error) {
	err := s.Impl.KeychainSetPassword(req.Key, req.Value)
	if err != nil {
		return nil, err
	}
	return &proto.KeychainSetPasswordResponse{}, nil
}

func (s *CoreCLIHelperServer) KeychainDeletePassword(ctx context.Context, req *proto.KeychainDeletePasswordRequest) (*proto.KeychainDeletePasswordResponse, error) {
	deleted, err := s.Impl.KeychainDeletePassword(req.Key)
	if err != nil {
		return nil, err
	}
	return &proto.KeychainDeletePasswordResponse{Deleted: deleted}, nil
}

func (s *CoreCLIHelperServer) KeychainFindCredentials(ctx context.Context, req *proto.KeychainFindCredentialsRequest) (*proto.KeychainFindCredentialsResponse, error) {
	keys, err := s.Impl.KeychainFindCredentials()
	if err != nil {
		return nil, err
	}
	return &proto.KeychainFindCredentialsResponse{Keys: keys}, nil
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

// KeychainGetPassword retrieves a password from the system keychain.
func (h *coreCLIHelper) KeychainGetPassword(key string) (string, bool, error) {
	item, err := config.KeyRing.Get(key)
	if err == keyring.ErrKeyNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(item.Data), true, nil
}

// KeychainSetPassword stores a password in the system keychain.
func (h *coreCLIHelper) KeychainSetPassword(key string, value string) error {
	return config.KeyRing.Set(keyring.Item{
		Key:   key,
		Data:  []byte(value),
		Label: key,
	})
}

// KeychainDeletePassword removes a password from the system keychain.
func (h *coreCLIHelper) KeychainDeletePassword(key string) (bool, error) {
	existingKeys, err := config.KeyRing.Keys()
	if err != nil {
		return false, err
	}
	for _, k := range existingKeys {
		if k == key {
			if err := config.KeyRing.Remove(key); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

// KeychainFindCredentials lists all keys stored in the keychain for this service.
func (h *coreCLIHelper) KeychainFindCredentials() ([]string, error) {
	return config.KeyRing.Keys()
}
