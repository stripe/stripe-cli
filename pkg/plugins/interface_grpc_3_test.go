package plugins

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stripe/stripe-cli/pkg/plugins/proto"
)

type stubGRPCBrokerAccepter struct {
	listener    net.Listener
	err         error
	acceptedIDs []uint32
}

func (s *stubGRPCBrokerAccepter) nextID() uint32 {
	return uint32(len(s.acceptedIDs) + 1)
}

func (s *stubGRPCBrokerAccepter) AcceptAndServe(id uint32, newGRPCServer func([]grpc.ServerOption) *grpc.Server) {
	s.acceptedIDs = append(s.acceptedIDs, id)
	if s.err != nil {
		return
	}

	server := newGRPCServer(nil)
	_ = server.Serve(s.listener)
}

type stubCoreCLIHelper struct{}

func (s *stubCoreCLIHelper) Echo(input string) (string, error) {
	return "echo:" + input, nil
}

func (s *stubCoreCLIHelper) SendAnalytics(eventName string, eventValue string) error {
	return nil
}

func (s *stubCoreCLIHelper) KeychainGetPassword(key string) (string, bool, error) {
	return "", false, nil
}

func (s *stubCoreCLIHelper) KeychainSetPassword(key string, value string) error {
	return nil
}

func (s *stubCoreCLIHelper) KeychainDeletePassword(key string) (bool, error) {
	return false, nil
}

func (s *stubCoreCLIHelper) KeychainFindCredentials() ([]string, error) {
	return nil, nil
}

func (s *stubCoreCLIHelper) RunPeerPlugin(pluginName string, args []string, cwd string) error {
	return nil
}

func TestStartCoreCLIHelperBrokerServerServesRPCs(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	broker := &stubGRPCBrokerAccepter{listener: listener}

	cleanup, err := startCoreCLIHelperBrokerServer(broker, 42, &stubCoreCLIHelper{})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	require.Equal(t, []uint32{42}, broker.acceptedIDs)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := proto.NewCoreCLIHelperClient(conn)
	resp, err := client.Echo(ctx, &proto.EchoRequest{Input: "hello"})
	require.NoError(t, err)
	require.Equal(t, "echo:hello", resp.Output)
}

func TestStartCoreCLIHelperBrokerServerReturnsAcceptError(t *testing.T) {
	expectedErr := errors.New("boom")
	broker := &stubGRPCBrokerAccepter{err: expectedErr}

	cleanup, err := startCoreCLIHelperBrokerServer(broker, 7, &stubCoreCLIHelper{})
	require.ErrorIs(t, err, errCoreCLIHelperBrokerServerStart)
	require.Nil(t, cleanup)
	require.Equal(t, []uint32{7}, broker.acceptedIDs)
}

type sequencingGRPCBroker struct {
	nextIDValue   uint32
	acceptedIDs   []uint32
	published     chan struct{}
	publishHelper chan struct{}
	returnServe   chan struct{}
}

func (s *sequencingGRPCBroker) nextID() uint32 {
	return atomic.AddUint32(&s.nextIDValue, 1)
}

func (s *sequencingGRPCBroker) AcceptAndServe(id uint32, newGRPCServer func([]grpc.ServerOption) *grpc.Server) {
	s.acceptedIDs = append(s.acceptedIDs, id)
	<-s.publishHelper
	close(s.published)
	_ = newGRPCServer(nil)
	<-s.returnServe
}

type recordingMainClient struct {
	called chan *proto.RunCommandRequest
}

func (c *recordingMainClient) RunCommand(ctx context.Context, in *proto.RunCommandRequest, opts ...grpc.CallOption) (*proto.RunCommandResponse, error) {
	c.called <- in
	return &proto.RunCommandResponse{}, nil
}

func TestGRPCClientV3RunCommandPublishesHelperAfterPluginRunCommandStarts(t *testing.T) {
	originalDelay := coreCLIHelperBrokerPublishDelay
	coreCLIHelperBrokerPublishDelay = 25 * time.Millisecond
	t.Cleanup(func() {
		coreCLIHelperBrokerPublishDelay = originalDelay
	})

	broker := &sequencingGRPCBroker{
		published:     make(chan struct{}),
		publishHelper: make(chan struct{}),
		returnServe:   make(chan struct{}),
	}
	mainClient := &recordingMainClient{
		called: make(chan *proto.RunCommandRequest, 1),
	}
	grpcClient := &GRPCClientV3{
		client: mainClient,
		broker: broker,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcClient.RunCommand(nil, []string{"search", "magazine"}, &stubCoreCLIHelper{})
	}()

	select {
	case req := <-mainClient.called:
		require.Equal(t, uint32(1), req.CoreCliHelperId)
		require.Equal(t, []string{"search", "magazine"}, req.Args)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for plugin RunCommand before helper broker publication")
	}

	select {
	case <-broker.published:
		t.Fatal("helper broker server published before plugin RunCommand started")
	case <-time.After(10 * time.Millisecond):
	}

	close(broker.publishHelper)

	select {
	case <-broker.published:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for helper broker publication")
	}

	require.Equal(t, []uint32{1}, broker.acceptedIDs)

	close(broker.returnServe)
	require.NoError(t, <-errCh)
}

type raceAwareGRPCBroker struct {
	nextIDValue           uint32
	acceptedIDs           []uint32
	pendingDialRegistered <-chan struct{}
	publishedAfterPending chan struct{}
	returnServe           chan struct{}
}

func (s *raceAwareGRPCBroker) nextID() uint32 {
	return atomic.AddUint32(&s.nextIDValue, 1)
}

func (s *raceAwareGRPCBroker) AcceptAndServe(id uint32, newGRPCServer func([]grpc.ServerOption) *grpc.Server) {
	s.acceptedIDs = append(s.acceptedIDs, id)

	select {
	case <-s.pendingDialRegistered:
		close(s.publishedAfterPending)
	default:
	}

	_ = newGRPCServer(nil)
	<-s.returnServe
}

type pendingDialMainClient struct {
	pendingDialRegistered chan struct{}
	publishedAfterPending <-chan struct{}
}

func (c *pendingDialMainClient) RunCommand(ctx context.Context, in *proto.RunCommandRequest, opts ...grpc.CallOption) (*proto.RunCommandResponse, error) {
	close(c.pendingDialRegistered)

	select {
	case <-c.publishedAfterPending:
		return &proto.RunCommandResponse{}, nil
	case <-time.After(250 * time.Millisecond):
		return nil, errors.New("broker publication dropped before pending dial was registered")
	}
}

func runCommandWithHelperPublishedFirst(client proto.MainClient, broker grpcBrokerClient, additionalInfo *proto.AdditionalInfo, args []string, coreCLIHelper CoreCLIHelper) error {
	brokerID := broker.nextID()

	cleanup, err := startCoreCLIHelperBrokerServer(broker, brokerID, coreCLIHelper)
	if err != nil {
		return err
	}
	defer cleanup()

	_, err = client.RunCommand(context.Background(), &proto.RunCommandRequest{
		AdditionalInfo:  additionalInfo,
		Args:            args,
		CoreCliHelperId: brokerID,
	})
	return err
}

func TestLegacyHelperBrokerPublicationOrderCanDropConnectionInfo(t *testing.T) {
	pendingDialRegistered := make(chan struct{})
	publishedAfterPending := make(chan struct{})
	broker := &raceAwareGRPCBroker{
		pendingDialRegistered: pendingDialRegistered,
		publishedAfterPending: publishedAfterPending,
		returnServe:           make(chan struct{}),
	}
	mainClient := &pendingDialMainClient{
		pendingDialRegistered: pendingDialRegistered,
		publishedAfterPending: publishedAfterPending,
	}

	err := runCommandWithHelperPublishedFirst(mainClient, broker, nil, []string{"search", "myproject"}, &stubCoreCLIHelper{})
	close(broker.returnServe)

	require.EqualError(t, err, "broker publication dropped before pending dial was registered")
	require.Equal(t, []uint32{1}, broker.acceptedIDs)
}

func TestGRPCClientV3RunCommandAvoidsDroppedHelperBrokerPublicationRace(t *testing.T) {
	originalDelay := coreCLIHelperBrokerPublishDelay
	coreCLIHelperBrokerPublishDelay = 25 * time.Millisecond
	t.Cleanup(func() {
		coreCLIHelperBrokerPublishDelay = originalDelay
	})

	pendingDialRegistered := make(chan struct{})
	publishedAfterPending := make(chan struct{})
	broker := &raceAwareGRPCBroker{
		pendingDialRegistered: pendingDialRegistered,
		publishedAfterPending: publishedAfterPending,
		returnServe:           make(chan struct{}),
	}
	mainClient := &pendingDialMainClient{
		pendingDialRegistered: pendingDialRegistered,
		publishedAfterPending: publishedAfterPending,
	}
	grpcClient := &GRPCClientV3{
		client: mainClient,
		broker: broker,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcClient.RunCommand(nil, []string{"search", "myproject"}, &stubCoreCLIHelper{})
	}()

	select {
	case <-publishedAfterPending:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for helper broker publication after pending dial registration")
	}

	require.Equal(t, []uint32{1}, broker.acceptedIDs)

	close(broker.returnServe)
	require.NoError(t, <-errCh)
}
