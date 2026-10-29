package judge

import (
	"encoding/json"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/stripeapi"
)

// doctorRulesAPIVersion pins the unreleased Stripe-Version the
// integ_hlth/doctor_rules endpoints require while they aren't publicly
// shipped. Remove once the API is GA and this header is no longer needed.
const doctorRulesAPIVersion = "2026-10-12-16.internal"

// FetchRule fetches topic's migration rule from the live doctor_rules
// API. Uses live-mode credentials only (see stripeapi.LoadCredentials) —
// the active context must actually be in live mode.
func FetchRule(cfg *config.Config, topic, stripeAccount string) (catalog.Rule, error) {
	creds, err := stripeapi.LoadCredentials(cfg)
	if err != nil {
		return catalog.Rule{}, err
	}
	client, err := stripeapi.NewClient(creds)
	if err != nil {
		return catalog.Rule{}, err
	}
	var resp struct {
		DoctorRule json.RawMessage `json:"doctor_rule"`
	}
	headers := map[string]string{"Stripe-Version": doctorRulesAPIVersion}
	path := "/v2/integ_hlth/doctor_rules/" + url.PathEscape(topic)
	if err := stripeapi.Get(client, stripeAccount, path, "", headers, &resp); err != nil {
		return catalog.Rule{}, err
	}
	return catalog.ParseAPIRule(resp.DoctorRule)
}
