package judge

// Live fetch of the dpm rule from the not-yet-public doctor_rules API.
// catalog.ParseAPIRule does the actual parsing; this file only resolves
// credentials and performs the GET, same as account.go's facts.

import (
	"encoding/json"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/doctor/catalog"
)

// doctorRulesAPIVersion pins the unreleased Stripe-Version the
// integ_hlth/doctor_rules endpoints require while they aren't publicly
// shipped. Remove once the API is GA and this header is no longer needed.
const doctorRulesAPIVersion = "2026-10-12-16.internal"

// FetchDPMRule fetches the dpm migration rule from the live doctor_rules
// API, in whichever mode (test or live) the active context is actually
// logged into — same as account facts, no --live flag needed.
func FetchDPMRule(cfg *config.Config, stripeAccount string) (catalog.Rule, error) {
	creds, err := loadCredentials(cfg)
	if err != nil {
		return catalog.Rule{}, err
	}
	client, err := newAccountClient(creds)
	if err != nil {
		return catalog.Rule{}, err
	}
	var resp struct {
		DoctorRule json.RawMessage `json:"doctor_rule"`
	}
	headers := map[string]string{"Stripe-Version": doctorRulesAPIVersion}
	if err := stripeGET(client, stripeAccount, "/v2/integ_hlth/doctor_rules/dpm", "", headers, &resp); err != nil {
		return catalog.Rule{}, err
	}
	return catalog.ParseAPIRule(resp.DoctorRule)
}
