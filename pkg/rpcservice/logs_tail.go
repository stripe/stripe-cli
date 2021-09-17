package rpcservice

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stripe/stripe-cli/pkg/logtailing"
	"github.com/stripe/stripe-cli/pkg/websocket"
	"github.com/stripe/stripe-cli/rpc"
)

// ITailer enables mocking a tailer object in tests
type ITailer interface {
	Run(context.Context) error
}

var createTailer = func(cfg *logtailing.Config) ITailer {
	return logtailing.New(cfg)
}

// LogsTail returns a stream of API logs
func (srv *RPCService) LogsTail(req *rpc.LogsTailRequest, stream rpc.StripeCLI_LogsTailServer) error {
	deviceName, err := srv.cfg.UserCfg.Profile.GetDeviceName()
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	key, err := srv.cfg.UserCfg.Profile.GetAPIKey(false)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	filters := getFiltersFromReq(req)

	logtailingVisitor := createVisitor(&stream)

	logtailingOutCh := make(chan websocket.IElement)

	logger := log.StandardLogger()

	tailer := createTailer(&logtailing.Config{
		DeviceName: deviceName,
		Filters:    filters,
		Key:        key,
		Log:        logger,
		OutCh:      logtailingOutCh,

		// Hidden for debugging
		APIBaseURL: "",
		NoWSS:      false,
	})

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go tailer.Run(ctx)

	for {
		select {
		case e := <-logtailingOutCh:
			err := e.Accept(logtailingVisitor)
			if err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func createVisitor(stream *rpc.StripeCLI_LogsTailServer) *websocket.Visitor {
	return &websocket.Visitor{
		VisitError: func(ee websocket.ErrorElement) error {
			return ee.Error
		},
		VisitData: func(de websocket.DataElement) error {
			log, ok := de.Data.(logtailing.EventPayload)
			if !ok {
				return fmt.Errorf("VisitData received unexpected type for DataElement, got %T expected %T", de, logtailing.EventPayload{})
			}

			logResponse := rpc.LogsTailResponse_Log{
				Livemode:  log.Livemode,
				Method:    log.Method,
				Url:       log.URL,
				Status:    int64(log.Status),
				RequestId: log.RequestID,
				CreatedAt: int64(log.CreatedAt),
			}

			// error struct is not empty
			hasError := log.Error != logtailing.RedactedError{}

			if hasError {
				logResponse.Error = &rpc.LogsTailResponse_Log_Error{
					Type:        log.Error.Type,
					Charge:      log.Error.Charge,
					Code:        log.Error.Code,
					DeclineCode: log.Error.DeclineCode,
					Message:     log.Error.Message,
					Param:       log.Error.Param,
				}
			}

			(*stream).Send(&rpc.LogsTailResponse{
				Content: &rpc.LogsTailResponse_Log_{
					Log: &logResponse,
				},
			})
			return nil
		},
		VisitStatus: func(se websocket.StateElement) error {
			var stateResponse rpc.LogsTailResponse_State
			switch se.State {
			case websocket.Loading:
				stateResponse = rpc.LogsTailResponse_STATE_LOADING
			case websocket.Reconnecting:
				stateResponse = rpc.LogsTailResponse_STATE_RECONNECTING
			case websocket.Ready:
				stateResponse = rpc.LogsTailResponse_STATE_READY
			case websocket.Done:
				stateResponse = rpc.LogsTailResponse_STATE_DONE
			}
			(*stream).Send(&rpc.LogsTailResponse{
				Content: &rpc.LogsTailResponse_State_{
					State: stateResponse,
				},
			})
			return nil
		},
	}
}

func getFiltersFromReq(req *rpc.LogsTailRequest) *logtailing.LogFilters {
	if req == nil {
		return nil
	}

	filterAccountRaw := req.FilterAccounts
	filterAccount := make([]string, len(filterAccountRaw))
	for i, v := range filterAccountRaw {
		switch v {
		case rpc.LogsTailRequest_ACCOUNT_CONNECT_IN:
			filterAccount[i] = "connect_in"
		case rpc.LogsTailRequest_ACCOUNT_CONNECT_OUT:
			filterAccount[i] = "connect_out"
		case rpc.LogsTailRequest_ACCOUNT_SELF:
			filterAccount[i] = "self"
		}
	}

	filterHTTPMethodRaw := req.FilterHttpMethods
	filterHTTPMethod := make([]string, len(filterHTTPMethodRaw))
	for i, v := range filterHTTPMethodRaw {
		switch v {
		case rpc.LogsTailRequest_HTTP_METHOD_DELETE:
			filterHTTPMethod[i] = "DELETE"
		case rpc.LogsTailRequest_HTTP_METHOD_GET:
			filterHTTPMethod[i] = "GET"
		case rpc.LogsTailRequest_HTTP_METHOD_POST:
			filterHTTPMethod[i] = "POST"
		}
	}

	filterRequestStatusRaw := req.FilterRequestStatuses
	filterRequestStatus := make([]string, len(filterRequestStatusRaw))
	for i, v := range filterRequestStatusRaw {
		switch v {
		case rpc.LogsTailRequest_REQUEST_STATUS_FAILED:
			filterRequestStatus[i] = "SUCCEEDED"
		case rpc.LogsTailRequest_REQUEST_STATUS_SUCCEEDED:
			filterRequestStatus[i] = "FAILED"
		}
	}

	filterSourceRaw := req.FilterSources
	filterSource := make([]string, len(filterSourceRaw))
	for i, v := range filterSourceRaw {
		switch v {
		case rpc.LogsTailRequest_SOURCE_API:
			filterSource[i] = "API"
		case rpc.LogsTailRequest_SOURCE_DASHBOARD:
			filterSource[i] = "DASHBOARD"
		}
	}

	filterStatusCodeTypeRaw := req.FilterStatusCodeTypes
	filterStatusCodeType := make([]string, len(filterStatusCodeTypeRaw))
	for i, v := range filterStatusCodeTypeRaw {
		switch v {
		case rpc.LogsTailRequest_STATUS_CODE_TYPE_2XX:
			filterStatusCodeType[i] = "2XX"
		case rpc.LogsTailRequest_STATUS_CODE_TYPE_4XX:
			filterStatusCodeType[i] = "4XX"
		case rpc.LogsTailRequest_STATUS_CODE_TYPE_5XX:
			filterStatusCodeType[i] = "5XX"
		}
	}

	return &logtailing.LogFilters{
		FilterAccount:        filterAccount,
		FilterHTTPMethod:     filterHTTPMethod,
		FilterIPAddress:      req.FilterIpAddresses,
		FilterRequestPath:    req.FilterRequestPaths,
		FilterRequestStatus:  filterRequestStatus,
		FilterSource:         filterSource,
		FilterStatusCode:     req.FilterStatusCodes,
		FilterStatusCodeType: filterStatusCodeType,
	}
}
