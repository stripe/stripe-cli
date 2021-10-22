package rpcservice

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/websocket"
	"github.com/stripe/stripe-cli/rpc"
)

const webhooksWebSocketFeature = "webhooks"

var httpMethodMap = map[string]rpc.ListenResponse_EndpointResponse_Data_HttpMethod{
	http.MethodDelete: rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_DELETE,
	http.MethodGet:    rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_GET,
	http.MethodPost:   rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_POST,
}

// IProxy enables mocking a proxy object in tests
type IProxy interface {
	Run(context.Context) error
}

var createProxy = func(ctx context.Context, cfg *proxy.Config) (IProxy, error) {
	return proxy.Init(ctx, cfg)
}

// Listen returns a stream of webhook events and forwards them to a local endpoint
func (srv *RPCService) Listen(req *rpc.ListenRequest, stream rpc.StripeCLI_ListenServer) error {
	deviceName, err := srv.cfg.UserCfg.Profile.GetDeviceName()
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	key, err := srv.cfg.UserCfg.Profile.GetAPIKey(req.Live)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	logger := log.StandardLogger()
	proxyVisitor := createProxyVisitor(&stream)
	proxyOutCh := make(chan websocket.IElement)

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	p, err := createProxy(ctx, &proxy.Config{
		DeviceName:            deviceName,
		Key:                   key,
		ForwardURL:            req.ForwardTo,
		ForwardHeaders:        req.Headers,
		ForwardConnectURL:     req.ForwardConnectTo,
		ForwardConnectHeaders: req.ConnectHeaders,
		UseConfiguredWebhooks: req.UseConfiguredWebhooks,
		WebSocketFeature:      webhooksWebSocketFeature,
		UseLatestAPIVersion:   req.Latest,
		SkipVerify:            req.SkipVerify,
		Log:                   logger,
		Events:                req.Events,
		OutCh:                 proxyOutCh,

		// Hidden for debugging
		APIBaseURL: "",
		NoWSS:      false,
	})

	if err != nil {
		return err
	}
	go p.Run(ctx)

	for {
		select {
		case e := <-proxyOutCh:
			err := e.Accept(proxyVisitor)
			if err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func createProxyVisitor(stream *rpc.StripeCLI_ListenServer) *websocket.Visitor {
	return &websocket.Visitor{
		VisitError: func(ee websocket.ErrorElement) error {
			switch ee.Error.(type) {
			case proxy.FailedToPostError, proxy.FailedToReadResponseError:
				// These errors shouldn't end the stream
				(*stream).Send(buildEndpointResponseErrorResp(ee.Error))
				return nil
			default:
				return ee.Error
			}
		},
		VisitData: func(de websocket.DataElement) error {
			switch data := de.Data.(type) {
			case proxy.StripeEvent:
				resp, err := buildStripeEventResp(&data)
				if err != nil {
					return err
				}
				(*stream).Send(resp)
				return nil
			case proxy.EndpointResponse:
				resp, err := buildEndpointResponseResp(&data)
				if err != nil {
					return err
				}
				(*stream).Send(resp)
				return nil
			default:
				return fmt.Errorf("VisitData received unexpected type for DataElement, got %T", de)
			}
		},
		VisitStatus: func(se websocket.StateElement) error {
			(*stream).Send(buildStateResponse(se))
			return nil
		},
	}
}

func buildEndpointResponseResp(raw *proxy.EndpointResponse) (*rpc.ListenResponse, error) {
	return &rpc.ListenResponse{
		Content: &rpc.ListenResponse_EndpointResponse_{
			EndpointResponse: &rpc.ListenResponse_EndpointResponse{
				Content: &rpc.ListenResponse_EndpointResponse_Data_{
					Data: &rpc.ListenResponse_EndpointResponse_Data{
						EventId:    raw.Event.ID,
						HttpMethod: getRPCMethodFromRequestMethod(raw.Resp.Request.Method),
						Status:     int64(raw.Resp.StatusCode),
						Url:        raw.Resp.Request.URL.String(),
					},
				},
			},
		},
	}, nil
}

func buildEndpointResponseErrorResp(raw error) *rpc.ListenResponse {
	return &rpc.ListenResponse{
		Content: &rpc.ListenResponse_EndpointResponse_{
			EndpointResponse: &rpc.ListenResponse_EndpointResponse{
				Content: &rpc.ListenResponse_EndpointResponse_Error{
					Error: raw.Error(),
				},
			},
		},
	}
}

func buildStateResponse(se websocket.StateElement) *rpc.ListenResponse {
	var stateResponse rpc.ListenResponse_State
	switch se.State {
	case websocket.Loading:
		stateResponse = rpc.ListenResponse_STATE_LOADING
	case websocket.Reconnecting:
		stateResponse = rpc.ListenResponse_STATE_RECONNECTING
	case websocket.Ready:
		stateResponse = rpc.ListenResponse_STATE_READY
	case websocket.Done:
		stateResponse = rpc.ListenResponse_STATE_DONE
	}
	return &rpc.ListenResponse{
		Content: &rpc.ListenResponse_State_{
			State: stateResponse,
		},
	}
}

func buildStripeEventResp(raw *proxy.StripeEvent) (*rpc.ListenResponse, error) {
	eventData, err := structpb.NewStruct(raw.Data)
	if err != nil {
		return nil, err
	}

	reqData, err := proxy.ExtractRequestData(raw.RequestData)

	if err != nil {
		return nil, err
	}

	request := rpc.StripeEvent_Request{
		Id:             reqData.ID,
		IdempotencyKey: reqData.IdempotencyKey,
	}

	return &rpc.ListenResponse{
		Content: &rpc.ListenResponse_StripeEvent{
			StripeEvent: &rpc.StripeEvent{
				Account:         raw.Account,
				ApiVersion:      raw.APIVersion,
				Created:         int64(raw.Created),
				Data:            eventData,
				Id:              raw.ID,
				Type:            raw.Type,
				Livemode:        raw.Livemode,
				PendingWebhooks: int64(raw.PendingWebhooks),
				Request:         &request,
			},
		},
	}, nil
}

func getRPCMethodFromRequestMethod(raw string) rpc.ListenResponse_EndpointResponse_Data_HttpMethod {
	var httpMethodResponse rpc.ListenResponse_EndpointResponse_Data_HttpMethod
	httpMethodResponse, ok := httpMethodMap[raw]
	if !ok {
		httpMethodResponse = rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_UNSPECIFIED
	}
	return httpMethodResponse
}
