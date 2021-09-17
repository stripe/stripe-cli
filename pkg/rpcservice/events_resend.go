package rpcservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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
		return nil, status.Error(codes.Unauthenticated, err.Error())
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

	path := formatURL("/v1/events/{event}/retry", []string{req.EventId})

	stripeReq := &requests.Base{
		Method:         strings.ToUpper(http.MethodPost),
		SuppressOutput: true,
		APIBaseURL:     baseURL,
	}

	params := getParamsFromReq(req)

	stripeResp, err := stripeReq.MakeRequest(ctx, apiKey, path, params, true)
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

	reqData, err := proxy.ExtractRequestData(evt.RequestData)

	if err != nil {
		return nil, err
	}

	request := rpc.StripeEvent_Request{
		Id:             reqData.ID,
		IdempotencyKey: reqData.IdempotencyKey,
	}

	return &rpc.EventsResendResponse{
		StripeEvent: &rpc.StripeEvent{
			Account:         evt.Account,
			ApiVersion:      evt.APIVersion,
			Created:         int64(evt.Created),
			Data:            data,
			Id:              evt.ID,
			Type:            evt.Type,
			Livemode:        evt.Livemode,
			PendingWebhooks: int64(evt.PendingWebhooks),
			Request:         &request,
		},
	}, nil
}

func formatURL(path string, urlParams []string) string {
	s := make([]interface{}, len(urlParams))
	for i, v := range urlParams {
		s[i] = v
	}

	re := regexp.MustCompile(`{\w+}`)
	format := re.ReplaceAllString(path, "%s")

	return fmt.Sprintf(format, s...)
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
