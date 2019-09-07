package proxy

import "fmt"

//
// Private types
//

// stripeEvent is a minimal representation of a Stripe `event` object, used
// to extract the event's ID and type for logging purposes.
type stripeEvent struct {
	Account  string `json:"account"`
	ID       string `json:"id"`
	Livemode bool   `json:"livemode"`
	Type     string `json:"type"`
	Created  int    `json:"created"`
}

func (e *stripeEvent) isConnect() bool {
	return e.Account != ""
}

func (e *stripeEvent) urlForEventID() string {
	return fmt.Sprintf("%s/events/%s", baseDashboardURL(e.Livemode, e.Account), e.ID)
}

func (e *stripeEvent) urlForEventType() string {
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
