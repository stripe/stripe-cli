package proxy

import "fmt"

//
// Private types
//

// stripeEvent is a minimal representation of a Stripe `event` object, used
// to extract the event's ID and type for logging purposes.
type stripeEvent struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Account string `json:"account"`
}

func (e *stripeEvent) isConnect() bool {
	return e.Account != ""
}

func (e *stripeEvent) urlForEventID() string {
	url := ""
	if e.isConnect() {
		url = fmt.Sprintf("https://dashboard.stripe.com/%s/test/events/%s", e.Account, e.ID)
	} else {
		url = fmt.Sprintf("https://dashboard.stripe.com/test/events/%s", e.ID)
	}
	return url
}

func (e *stripeEvent) urlForEventType() string {
	url := ""
	if e.isConnect() {
		url = fmt.Sprintf("https://dashboard.stripe.com/%s/test/events?type=%s", e.Account, e.Type)
	} else {
		url = fmt.Sprintf("https://dashboard.stripe.com/test/events?type=%s", e.Type)
	}
	return url
}
