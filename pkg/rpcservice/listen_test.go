package rpcservice

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

	expected := &rpc.ListenResponse{
		Content: &rpc.ListenResponse_StripeEvent{
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
		},
	}

	resp, err := listenClient.Recv()
	stripeEventResp := resp.GetStripeEvent()
	expectedStripeEventResp := expected.GetStripeEvent()
	assert.Nil(t, err)
	assert.NotNil(t, stripeEventResp)
	assert.Equal(t, expectedStripeEventResp.Id, stripeEventResp.Id)
	assert.Equal(t, expectedStripeEventResp.ApiVersion, stripeEventResp.ApiVersion)
	assert.True(t, assert.ObjectsAreEqual(expectedStripeEventResp.Data, stripeEventResp.Data))
	assert.Equal(t, expectedStripeEventResp.Request, stripeEventResp.Request)
	assert.Equal(t, expectedStripeEventResp.Type, stripeEventResp.Type)
	assert.Equal(t, expectedStripeEventResp.Created, stripeEventResp.Created)
	assert.Equal(t, expectedStripeEventResp.Livemode, stripeEventResp.Livemode)
	assert.Equal(t, expectedStripeEventResp.PendingWebhooks, stripeEventResp.PendingWebhooks)

	cancel()

	resp, err = listenClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestListenStreamsEndpointResponses(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createProxy = func(cfg *proxy.Config) (IProxy, error) {
		runProxy = func(ctx context.Context) error {
			r := httptest.NewRequest(http.MethodPost, "localhost:4242/webhook", strings.NewReader(""))
			cfg.OutCh <- websocket.DataElement{
				Data: proxy.EndpointResponse{
					Event: &proxy.StripeEvent{
						ID: "evt_12345",
					},
					Resp: &http.Response{
						StatusCode: 200,
						Request:    r,
					},
					RespBody: "foo",
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

	expected := &rpc.ListenResponse{
		Content: &rpc.ListenResponse_EndpointResponse_{
			EndpointResponse: &rpc.ListenResponse_EndpointResponse{
				Content: &rpc.ListenResponse_EndpointResponse_Data_{
					Data: &rpc.ListenResponse_EndpointResponse_Data{
						Status:     200,
						HttpMethod: rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_POST,
						Url:        "localhost:4242/webhook",
						EventId:    "evt_12345",
						Body:       "foo",
					},
				},
			},
		},
	}

	resp, err := listenClient.Recv()
	endpointResponse := resp.GetEndpointResponse().GetData()
	expectedEndpointResponse := expected.GetEndpointResponse().GetData()
	assert.Nil(t, err)
	assert.NotNil(t, endpointResponse)
	assert.Equal(t, expectedEndpointResponse.Body, endpointResponse.Body)
	assert.Equal(t, expectedEndpointResponse.EventId, endpointResponse.EventId)
	assert.Equal(t, expectedEndpointResponse.HttpMethod, endpointResponse.HttpMethod)
	assert.Equal(t, expectedEndpointResponse.Status, endpointResponse.Status)
	assert.Equal(t, expectedEndpointResponse.Url, endpointResponse.Url)

	cancel()

	resp, err = listenClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestListenReturnsEndpointResponseError(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createProxy = func(cfg *proxy.Config) (IProxy, error) {
		runProxy = func(ctx context.Context) error {
			cfg.OutCh <- websocket.ErrorElement{
				Error: proxy.FailedToPostError{Err: errors.New("failed to post")},
			}
			cfg.OutCh <- websocket.ErrorElement{
				Error: proxy.FailedToReadResponseError{Err: errors.New("failed to read response")},
			}
			return nil
		}
		return &mockProxy{
			OutCh: cfg.OutCh,
		}, nil
	}

	listenClient, err := client.Listen(ctx, &rpc.ListenRequest{})
	assert.Nil(t, err)

	for i := 0; i < 2; i++ {
		resp, err := listenClient.Recv()
		assert.Nil(t, err)
		assert.NotNil(t, resp.GetEndpointResponse().GetError())
	}

	cancel()

	resp, err := listenClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}

func TestListenReturnsGenericError(t *testing.T) {
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

func TestListenSucceedsWithAllParams(t *testing.T) {
	ctx, cancel := context.WithCancel(withAuth(context.Background()))

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := rpc.NewStripeCLIClient(conn)

	createProxy = func(cfg *proxy.Config) (IProxy, error) {
		runProxy = func(ctx context.Context) error {
			return nil
		}
		return &mockProxy{
			OutCh: cfg.OutCh,
		}, nil
	}

	listenRequest := rpc.ListenRequest{
		ConnectHeaders:        []string{"foo=bar"},
		Events:                []string{"customer.created", "checkout.session.completed"},
		ForwardConnectTo:      "localhost:4242/webhook/connect",
		ForwardTo:             "localhost:4242/webhook",
		Headers:               []string{"foo=bar"},
		Latest:                true,
		Live:                  true,
		SkipVerify:            true,
		UseConfiguredWebhooks: true,
	}

	listenClient, err := client.Listen(ctx, &listenRequest)
	assert.Nil(t, err)

	cancel()

	resp, err := listenClient.Recv()
	assert.Equal(t, status.Error(codes.Canceled, "context canceled").Error(), err.Error())
	assert.Nil(t, resp)
}
