package judge

import (
	"encoding/json"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/internal/stripeapi"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
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

// RulesAPIAvailable reports whether the doctor_rules API exists for the
// active credentials. doctor's rollout is gated on this endpoint: a 404
// means the command isn't available yet (ok=false, err=nil), not an
// error.
func RulesAPIAvailable(cfg *config.Config, stripeAccount string) (bool, error) {
	creds, err := stripeapi.LoadCredentials(cfg)
	if err != nil {
		return false, err
	}
	client, err := stripeapi.NewClient(creds)
	if err != nil {
		return false, err
	}
	headers := map[string]string{"Stripe-Version": doctorRulesAPIVersion}
	status, err := stripeapi.StatusCode(client, stripeAccount, "/v2/integ_hlth/doctor_rules", headers)
	if err != nil {
		return false, err
	}
	switch status {
	case 404:
		return false, nil
	case 200:
		return true, nil
	case 401:
		return false, errorcategory.Errorf(errorcategory.Auth, "credentials rejected (401) — key may be expired; run `stripe login`")
	default:
		return false, errorcategory.Errorf(errorcategory.API, "GET /v2/integ_hlth/doctor_rules: HTTP %d", status)
	}
}
