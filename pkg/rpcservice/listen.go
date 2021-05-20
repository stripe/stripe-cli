package rpcservice

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/websocket"
	"github.com/stripe/stripe-cli/rpc"
)

const webhooksWebSocketFeature = "webhooks"

// IProxy enables mocking a proxy object in tests
type IProxy interface {
	Run(context.Context) error
}

var createProxy = func(cfg *proxy.Config) (IProxy, error) {
	return proxy.Init(cfg)
}

// Listen returns a stream of webhook events and forwards them to a local endpoint
func (srv *RPCService) Listen(req *rpc.ListenRequest, stream rpc.StripeCLI_ListenServer) error {
	deviceName, err := srv.cfg.UserCfg.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	key, err := srv.cfg.UserCfg.Profile.GetAPIKey(req.Live)
	if err != nil {
		return err
	}

	logger := log.StandardLogger()
	proxyVisitor := createProxyVisitor(&stream)
	proxyOutCh := make(chan websocket.IElement)

	p, err := createProxy(&proxy.Config{
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
				(*stream).Send(&rpc.ListenResponse{
					Content: &rpc.ListenResponse_EndpointResponse_{
						EndpointResponse: &rpc.ListenResponse_EndpointResponse{
							Content: &rpc.ListenResponse_EndpointResponse_Error{
								Error: ee.Error.Error(),
							},
						},
					},
				})
				return nil
			default:
				return ee.Error
			}
		},
		VisitData: func(de websocket.DataElement) error {
			switch data := de.Data.(type) {
			case proxy.StripeEvent:
				eventData, err := structpb.NewStruct(data.Data)
				if err != nil {
					return err
				}
				request := rpc.StripeEvent_Request{
					Id:             data.Request.ID,
					IdempotencyKey: data.Request.IdempotencyKey,
				}
				(*stream).Send(&rpc.ListenResponse{
					Content: &rpc.ListenResponse_StripeEvent{
						StripeEvent: &rpc.StripeEvent{
							Account:         data.Account,
							ApiVersion:      data.APIVersion,
							Created:         int64(data.Created),
							Data:            eventData,
							Id:              data.ID,
							Type:            data.Type,
							Livemode:        data.Livemode,
							PendingWebhooks: int64(data.PendingWebhooks),
							Request:         &request,
						},
					},
				})
				return nil
			case proxy.EndpointResponse:
				var httpMethodResponse rpc.ListenResponse_EndpointResponse_Data_HttpMethod
				switch data.Resp.Request.Method {
				case http.MethodDelete:
					httpMethodResponse = rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_DELETE
				case http.MethodGet:
					httpMethodResponse = rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_GET
				case http.MethodPost:
					httpMethodResponse = rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_POST
				default:
					httpMethodResponse = rpc.ListenResponse_EndpointResponse_Data_HTTP_METHOD_UNSPECIFIED
				}

				(*stream).Send(&rpc.ListenResponse{
					Content: &rpc.ListenResponse_EndpointResponse_{
						EndpointResponse: &rpc.ListenResponse_EndpointResponse{
							Content: &rpc.ListenResponse_EndpointResponse_Data_{
								Data: &rpc.ListenResponse_EndpointResponse_Data{
									Body:       data.RespBody,
									EventId:    data.Event.ID,
									HttpMethod: httpMethodResponse,
									Status:     int64(data.Resp.StatusCode),
									Url:        data.Resp.Request.URL.String(),
								},
							},
						},
					},
				})
				return nil
			default:
				return fmt.Errorf("VisitData received unexpected type for DataElement, got %T", de)
			}
		},
		VisitStatus: func(se websocket.StateElement) error {
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
			(*stream).Send(&rpc.ListenResponse{
				Content: &rpc.ListenResponse_State_{
					State: stateResponse,
				},
			})
			return nil
		},
	}
}
