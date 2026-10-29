package dpm

// facts.go fetches the two pieces of live account evidence Run needs:
// recent API versions (fetchVersionFacts) and the account's
// payment-method configuration (fetchPMConfigFacts) — the same evidence
// doctor's old generic account-facts fetch used, now owned entirely by
// this topic.

import (
	"sort"

	"github.com/stripe/stripe-cli/pkg/doctor/stripeapi"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// dpmCutoff is the API version at/after which removing payment_method_types
// alone enables dynamic payment methods; before it, automatic_payment_methods
// [enabled]=true must be added or methods are silently lost. Specific to
// this topic's judgment, not a doctor-wide concept.
const dpmCutoff = "2023-08-16"

// versionFacts summarizes the API versions this account's recent traffic
// actually used.
type versionFacts struct {
	Sampled       int
	NoEvents      bool // no recent events: version facts are unknowable
	VersionsOK    bool // every sampled event version >= dpmCutoff
	VersionsMixed bool // some but not all versions >= dpmCutoff
}

// fetchVersionFacts samples the account's most recent events for their
// api_version, the same evidence doctor's old account-facts fetch used.
func fetchVersionFacts(client *stripe.Client, stripeAccount string) (versionFacts, error) {
	var evts struct {
		Data []struct {
			APIVersion string `json:"api_version"`
		} `json:"data"`
	}
	if err := stripeapi.Get(client, stripeAccount, "/v1/events", "limit=20", nil, &evts); err != nil {
		return versionFacts{}, err
	}
	ge, total := 0, 0
	for _, e := range evts.Data {
		if e.APIVersion == "" {
			continue
		}
		total++
		if datePrefix(e.APIVersion) >= dpmCutoff {
			ge++
		}
	}
	return versionFacts{
		Sampled:       total,
		NoEvents:      total == 0,
		VersionsOK:    total > 0 && ge == total,
		VersionsMixed: ge > 0 && ge < total,
	}, nil
}

// datePrefix compares versions by their YYYY-MM-DD prefix, which handles
// both classic ("2023-08-16") and named ("2026-06-24.dahlia") formats.
func datePrefix(v string) string {
	if len(v) >= 10 {
		return v[:10]
	}
	return v
}

// pmConfigFacts summarizes the account's active payment-method
// configuration: which methods are toggled on, and which of those are
// toggled on but not actually available (capability inactive — they will
// NOT render; both are required).
type pmConfigFacts struct {
	ConfigCount           int
	ActiveConfigID        string
	ActiveConfigName      string
	ActiveConfigIsDefault bool
	EnabledMethods        []string
	UnavailableMethods    []string
	ConfiguredOK          bool
}

// fetchPMConfigFacts reads /v1/payment_method_configurations and picks ONE
// governing config — the default, else the first active one — the same
// selection doctor's old account-facts fetch used.
func fetchPMConfigFacts(client *stripe.Client, stripeAccount string) (pmConfigFacts, error) {
	var pmc struct {
		Data []map[string]any `json:"data"`
	}
	if err := stripeapi.Get(client, stripeAccount, "/v1/payment_method_configurations", "", nil, &pmc); err != nil {
		return pmConfigFacts{}, err
	}

	var f pmConfigFacts
	var chosen map[string]any
	for _, cfg := range pmc.Data {
		active, _ := cfg["active"].(bool)
		if !active {
			continue // deactivated demo leftovers shouldn't inflate anything
		}
		f.ConfigCount++
		isDefault, _ := cfg["is_default"].(bool)
		if isDefault || chosen == nil {
			chosen = cfg
			f.ActiveConfigIsDefault = isDefault
		}
	}
	if chosen != nil {
		if name, ok := chosen["name"].(string); ok {
			f.ActiveConfigName = name
		}
		if id, ok := chosen["id"].(string); ok {
			f.ActiveConfigID = id
		}
		for field, v := range chosen {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			dp, ok := m["display_preference"].(map[string]any)
			if !ok || dp["value"] != "on" {
				continue
			}
			f.EnabledMethods = append(f.EnabledMethods, field)
			if avail, ok := m["available"].(bool); ok && !avail {
				f.UnavailableMethods = append(f.UnavailableMethods, field)
			}
		}
		sort.Strings(f.EnabledMethods)
		sort.Strings(f.UnavailableMethods)
	}
	f.ConfiguredOK = f.ConfigCount > 0 && len(f.EnabledMethods) > 0
	return f, nil
}
