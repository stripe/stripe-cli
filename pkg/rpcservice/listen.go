package rpcservice

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	// validate forward-urls args
	if req.UseConfiguredWebhooks && len(req.ForwardTo) > 0 {
		if strings.HasPrefix(req.ForwardTo, "/") {
			return status.Error(codes.InvalidArgument, "forward_to cannot be a relative path when loading webhook endpoints from the API")
		}
		if strings.HasPrefix(req.ForwardConnectTo, "/") {
			return status.Error(codes.InvalidArgument, "forward_connect_to cannot be a relative path when loading webhook endpoints from the API")
		}
	} else if req.UseConfiguredWebhooks && len(req.ForwardTo) == 0 {
		return status.Error(codes.InvalidArgument, "load_from_webhooks_api requires a location to forward to with forward_to")
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
			return ee.Error
		},
		VisitData: func(de websocket.DataElement) error {
			stripeEvent, ok := de.Data.(proxy.StripeEvent)
			if !ok {
				return fmt.Errorf("VisitData received unexpected type for DataElement, got %T expected %T", de, proxy.StripeEvent{})
			}

			data, err := structpb.NewStruct(stripeEvent.Data)
			if err != nil {
				return err
			}

			request := rpc.StripeEvent_Request{
				Id:             stripeEvent.Request.ID,
				IdempotencyKey: stripeEvent.Request.IdempotencyKey,
			}

			(*stream).Send(&rpc.ListenResponse{
				Content: &rpc.ListenResponse_StripeEvent{
					StripeEvent: &rpc.StripeEvent{
						Account:         stripeEvent.Account,
						ApiVersion:      stripeEvent.APIVersion,
						Created:         int64(stripeEvent.Created),
						Data:            data,
						Id:              stripeEvent.ID,
						Type:            stripeEvent.Type,
						Livemode:        stripeEvent.Livemode,
						PendingWebhooks: int64(stripeEvent.PendingWebhooks),
						Request:         &request,
					},
				},
			})
			return nil
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
