package rpcservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type stripeRequestData struct {
	ID             string `json:"id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type stripeEvent struct {
	Account         string                 `json:"account"`
	APIVersion      string                 `json:"api_version"`
	Created         int                    `json:"created"`
	Data            map[string]interface{} `json:"data"`
	ID              string                 `json:"id"`
	Livemode        bool                   `json:"livemode"`
	Request         stripeRequestData      `json:"request"`
	PendingWebhooks int                    `json:"pending_webhooks"`
	Type            string                 `json:"type"`
}

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

	path := resource.FormatURL(resource.PathTemplate, []string{req.EventId})

	stripeReq := &requests.Base{
		Method:         strings.ToUpper(http.MethodPost),
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}

	params := getParamsFromReq(req)

	stripeResp, err := stripeReq.MakeRequest(apiKey, path, params, true)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	var evt stripeEvent

	err = json.Unmarshal(stripeResp, &evt)
	if err != nil {
		return nil, err
	}

	data, err := structpb.NewStruct(evt.Data)
	if err != nil {
		return nil, err
	}

	request := rpc.EventsResendResponse_Request{
		Id:             evt.Request.ID,
		IdempotencyKey: evt.Request.IdempotencyKey,
	}

	return &rpc.EventsResendResponse{
		Account:         evt.Account,
		ApiVersion:      evt.APIVersion,
		Created:         int64(evt.Created),
		Data:            data,
		Id:              evt.ID,
		Type:            evt.Type,
		Livemode:        evt.Livemode,
		PendingWebhooks: int64(evt.PendingWebhooks),
		Request:         &request,
	}, nil
}

func getParamsFromReq(req *rpc.EventsResendRequest) *requests.RequestParameters {
	params := requests.RequestParameters{}

	if len(req.Data) > 0 {
		params.AppendData(req.Data)
	}

	if len(req.Expand) > 0 {
		params.AppendExpand(req.Expand)
	}

	if req.Idempotency != "" {
		params.SetIdempotency(req.Idempotency)
	}

	if req.StripeAccount != "" {
		params.SetStripeAccount(req.StripeAccount)
	}

	if req.Version != "" {
		params.SetVersion(req.Version)
	}

	if req.WebhookEndpoint == "" {
		params.AppendData([]string{"for_stripecli=true"})
	} else {
		params.AppendData([]string{fmt.Sprintf("webhook_endpoint=%s", req.WebhookEndpoint)})
	}

	if req.Account != "" {
		params.AppendData([]string{fmt.Sprintf("account=%s", req.Account)})
	}

	return &params
}
