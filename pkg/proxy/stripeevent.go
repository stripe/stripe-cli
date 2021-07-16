package proxy

import "fmt"

// StripeEvent is a representation of a Stripe `event` object
// we define RequestData as an interface for backwards compatibility
// Request will hold the deserialized request data
type StripeEvent struct {
	Account         string                 `json:"account"`
	APIVersion      string                 `json:"api_version"`
	Created         int                    `json:"created"`
	Data            map[string]interface{} `json:"data"`
	ID              string                 `json:"id"`
	Livemode        bool                   `json:"livemode"`
	PendingWebhooks int                    `json:"pending_webhooks"`
	Type            string                 `json:"type"`
	RequestData     interface{}            `json:"request"`
	Request         StripeRequest
}

// StripeRequest is a representation of the Request field in a Stripe `event` object
type StripeRequest struct {
	ID             string
	IdempotencyKey string
}

// IsConnect return true or false if *StripeEvent is connect or not.
func (e *StripeEvent) IsConnect() bool {
	return e.Account != ""
}

// URLForEventID builds a full URL from a StripeEvent ID.
func (e *StripeEvent) URLForEventID() string {
	return fmt.Sprintf("%s/events/%s", baseDashboardURL(e.Livemode, e.Account), e.ID)
}

// URLForEventType builds a full URL from a StripeEvent Type.
func (e *StripeEvent) URLForEventType() string {
	return fmt.Sprintf("%s/events?type=%s", baseDashboardURL(e.Livemode, e.Account), e.Type)
}

func baseDashboardURL(livemode bool, account string) string {
	maybeTest := ""
	if !livemode {
		maybeTest = "/test"
	}

	maybeAccount := ""
	if account != "" {
		maybeAccount = fmt.Sprintf("/%s", account)
	}

	return fmt.Sprintf("https://dashboard.stripe.com%s%s", maybeAccount, maybeTest)
}
