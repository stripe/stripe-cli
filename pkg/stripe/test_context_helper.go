// This package exports a common context to be used for testing
package stripe

import (
	"context"
	"fmt"
	"net/http"
)

type TestTelemetry struct{}

func (t *TestTelemetry) SendAPIRequestEvent(ctx context.Context, requestID string, livemode bool) (*http.Response, error) {
	// Do nothing
	fmt.Printf("Calling test SendAPIRequestEvent with %v, %v", requestID, livemode)
	return nil, nil
}

func (t *TestTelemetry) SendEvent(ctx context.Context, eventName string, eventValue string) (*http.Response, error) {
	// Do nothing
	fmt.Printf("Calling test SendEvent with %v, %v", eventName, eventValue)
	return nil, nil

}

func GetTestContext() context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, TelemetryClientKey{}, &TestTelemetry{})
}
