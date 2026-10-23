package judge

// Live fetch of the dpm rule from the not-yet-public doctor_rules API.
// catalog.ParseAPIRule does the actual parsing; this file only resolves
// credentials and performs the GET, using the shared stripeapi helpers.

import (
	"encoding/json"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
	"github.com/stripe/stripe-cli/pkg/doctor/stripeapi"
)

// doctorRulesAPIVersion pins the unreleased Stripe-Version the
// integ_hlth/doctor_rules endpoints require while they aren't publicly
// shipped. Remove once the API is GA and this header is no longer needed.
const doctorRulesAPIVersion = "2026-10-12-16.internal"

// FetchDPMRule fetches the dpm migration rule from the live doctor_rules
// API. Uses live-mode credentials only (see stripeapi.LoadCredentials) —
// no --live flag needed, but the active context must actually be in live
// mode.
func FetchDPMRule(cfg *config.Config, stripeAccount string) (catalog.Rule, error) {
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
	if err := stripeapi.Get(client, stripeAccount, "/v2/integ_hlth/doctor_rules/dpm", "", headers, &resp); err != nil {
		return catalog.Rule{}, err
	}
	return catalog.ParseAPIRule(resp.DoctorRule)
}
