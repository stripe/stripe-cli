package stripe

import (
	"net/http"
	"strings"
)

// AuthStrategy sets authentication headers on an outgoing request.
type AuthStrategy interface {
	SetAuthHeader(req *http.Request) error
	IsLiveMode() bool
}

// APIKeyAuth authenticates with a static Stripe API key.
type APIKeyAuth struct{ APIKey string }

func (a *APIKeyAuth) SetAuthHeader(req *http.Request) error {
	if a.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.APIKey)
	}
	return nil
}

func (a *APIKeyAuth) IsLiveMode() bool { return strings.Contains(a.APIKey, "live") }
