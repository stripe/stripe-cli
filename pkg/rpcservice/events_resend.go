package rpcservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/proxy"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventsResend resends an event given an event ID
func (srv *RPCService) EventsResend(ctx context.Context, req *rpc.EventsResendRequest) (*rpc.EventsResendResponse, error) {
	apiKey, err := srv.cfg.UserCfg.Profile.GetAPIKey(req.Live)
	if err != nil {
		return nil, err
	}

	if req.EventId == "" {
		return nil, status.Error(codes.InvalidArgument, "Event ID is required")
	}

	for _, datum := range req.Data {
		split := strings.SplitN(datum, "=", 2)
		if len(split) < 2 {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid data argument: %s", datum)
		}
	}

	params := requests.RequestParameters{
		Data:          req.Data,
		Expand:        req.Expand,
		Idempotency:   req.Idempotency,
		StripeAccount: req.StripeAccount,
		Version:       req.Version,
	}

	if req.WebhookEndpoint == "" {
		params.AppendData([]string{"for_stripecli=true"})
	} else {
		params.AppendData([]string{fmt.Sprintf("webhook_endpoint=%s", req.WebhookEndpoint)})
	}

	if req.Account != "" {
		params.AppendData([]string{fmt.Sprintf("account=%s", req.Account)})
	}

	path := resource.FormatURL(resource.PathTemplate, []string{req.EventId})

	stripeReq := &requests.Base{
		Method:         strings.ToUpper(http.MethodPost),
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}

	stripeResp, err := stripeReq.MakeRequest(apiKey, path, &params, true)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	var evt proxy.StripeEvent

	err = json.Unmarshal(stripeResp, &evt)
	if err != nil {
		return nil, err
	}

	data, err := structpb.NewStruct(evt.Data)
	if err != nil {
		return nil, err
	}

	return &rpc.EventsResendResponse{
		Account:    evt.Account,
		ApiVersion: evt.APIVersion,
		Created:    int64(evt.Created),
		Data:       data,
		Id:         evt.ID,
		Request: &rpc.EventsResendResponse_Request{
			Id:             evt.Request.ID,
			IdempotencyKey: evt.Request.IdempotencyKey,
		},
		Type:            evt.Type,
		Livemode:        evt.Livemode,
		PendingWebhooks: int64(evt.PendingWebhooks),
	}, nil
}
