package rpcservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IStripeReq enables mocking for tests
type IStripeReq interface {
	MakeRequest(apiKey string, path string, params *requests.RequestParameters, errOnStatus bool) ([]byte, error)
}

var getStripeReq = func() IStripeReq {
	stripeReq := &requests.Base{
		Method:         strings.ToUpper(http.MethodPost),
		SuppressOutput: true,
		APIBaseURL:     stripe.DefaultAPIBaseURL,
	}
	return stripeReq
}

// EventsResend resends an event given an event ID
func (srv *RPCService) EventsResend(ctx context.Context, req *rpc.EventsResendRequest) (*rpc.EventsResendResponse, error) {
	apiKey, err := srv.cfg.UserCfg.Profile.GetAPIKey(req.Live)
	if err != nil {
		return nil, err
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
		Version:       req.Version,
		StripeAccount: req.StripeAccount,
	}

	if req.WebhookEndpoint == "" {
		params.AppendData([]string{"for_stripecli=true"})
	} else {
		params.AppendData([]string{fmt.Sprintf("webhook_endpoint=%s", req.WebhookEndpoint)})
	}

	if req.Account == "" {
		params.AppendData([]string{"account=%s", req.Account})
	}

	path := resource.FormatURL(resource.PathTemplate, []string{req.EventId})

	stripeReq := getStripeReq()

	stripeResp, err := stripeReq.MakeRequest(apiKey, path, &params, true)
	if err != nil {
		return nil, err
	}

	return &rpc.EventsResendResponse{
		Payload: string(stripeResp),
	}, nil
}
