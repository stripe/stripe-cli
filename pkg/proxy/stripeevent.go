package proxy

import "fmt"

type stripeRequestData struct {
	ID             string `json:"id"`
	IdempotencyKey string `json:"idempotency_key"`
}

// StripeEvent is a  representation of a Stripe `event` object, used
// to extract the event's ID and type for logging purposes.
type StripeEvent struct {
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

func (e *StripeEvent) isConnect() bool {
	return e.Account != ""
}

func (e *StripeEvent) urlForEventID() string {
	return fmt.Sprintf("%s/events/%s", baseDashboardURL(e.Livemode, e.Account), e.ID)
}

func (e *StripeEvent) urlForEventType() string {
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
