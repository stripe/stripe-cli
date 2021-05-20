package rpcservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/websocket"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var runProxy func(ctx context.Context) error

type mockProxy struct {
	OutCh chan websocket.IElement
}

func (mp *mockProxy) Run(ctx context.Context) error {
	return runProxy(ctx)
}

func TestListenStreamsState(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createProxy = func(cfg *proxy.Config) (IProxy, error) {
		runProxy = func(ctx context.Context) error {
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Loading,
			}
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Reconnecting,
			}
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Ready,
			}
			cfg.OutCh <- websocket.StateElement{
				State: websocket.Done,
			}
			return nil
		}
		return &mockProxy{
			OutCh: cfg.OutCh,
		}, nil
	}

	listenClient, err := client.Listen(ctx, &rpc.ListenRequest{})
	assert.Nil(t, err)

	expectedStates := []rpc.ListenResponse_State{
		rpc.ListenResponse_STATE_LOADING,
		rpc.ListenResponse_STATE_RECONNECTING,
		rpc.ListenResponse_STATE_READY,
		rpc.ListenResponse_STATE_DONE,
	}

	for _, s := range expectedStates {
		resp, err := listenClient.Recv()
		assert.Nil(t, err)
		assert.Equal(t, s, resp.GetState())
	}

	cancel()

	resp, err := listenClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestListenStreamsEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createProxy = func(cfg *proxy.Config) (IProxy, error) {
		runProxy = func(ctx context.Context) error {
			cfg.OutCh <- websocket.DataElement{
				Data: proxy.StripeEvent{
					Account:         "acct_12345",
					APIVersion:      "2020-08-27",
					Created:         12345,
					ID:              "evt_12345",
					Livemode:        false,
					PendingWebhooks: 2,
					Type:            "checkout.session.completed",
					Data: map[string]interface{}{
						"object": map[string]interface{}{
							"id": "cs_test_12345",
						},
					},
					Request: proxy.StripeRequestData{
						ID:             "req_12345",
						IdempotencyKey: "foo",
					},
				},
			}
			return nil
		}
		return &mockProxy{
			OutCh: cfg.OutCh,
		}, nil
	}

	listenClient, err := client.Listen(ctx, &rpc.ListenRequest{})
	assert.Nil(t, err)

	expectedData, err := structpb.NewStruct(map[string]interface{}{
		"object": map[string]interface{}{
			"id": "cs_test_12345",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create expected event data")
	}

	expected := &rpc.EventsResendResponse{
		StripeEvent: &rpc.StripeEvent{
			Id:              "evt_12345",
			ApiVersion:      "2020-08-27",
			Data:            expectedData,
			Type:            "checkout.session.completed",
			Created:         12345,
			Livemode:        false,
			PendingWebhooks: 2,
			Request: &rpc.StripeEvent_Request{
				Id:             "req_12345",
				IdempotencyKey: "foo",
			},
		},
	}

	resp, err := listenClient.Recv()
	stripeEventResp := resp.GetStripeEvent()
	assert.Nil(t, err)
	assert.NotNil(t, stripeEventResp)
	assert.Equal(t, expected.StripeEvent.Id, stripeEventResp.Id)
	assert.Equal(t, expected.StripeEvent.ApiVersion, stripeEventResp.ApiVersion)
	assert.True(t, assert.ObjectsAreEqual(expected.StripeEvent.Data, stripeEventResp.Data))
	assert.Equal(t, expected.StripeEvent.Request, stripeEventResp.Request)
	assert.Equal(t, expected.StripeEvent.Type, stripeEventResp.Type)
	assert.Equal(t, expected.StripeEvent.Created, stripeEventResp.Created)
	assert.Equal(t, expected.StripeEvent.Livemode, stripeEventResp.Livemode)
	assert.Equal(t, expected.StripeEvent.PendingWebhooks, stripeEventResp.PendingWebhooks)

	cancel()

	resp, err = listenClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestListenReturnsError(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createProxy = func(cfg *proxy.Config) (IProxy, error) {
		runProxy = func(ctx context.Context) error {
			myErr := errors.New("my error")
			cfg.OutCh <- websocket.ErrorElement{
				Error: myErr,
			}
			return myErr
		}
		return &mockProxy{
			OutCh: cfg.OutCh,
		}, nil
	}

	listenClient, err := client.Listen(ctx, &rpc.ListenRequest{})
	assert.Nil(t, err)

	resp, err := listenClient.Recv()
	assert.Equal(t, status.Error(codes.Unknown, "my error").Error(), err.Error())
	assert.Nil(t, resp)
}
