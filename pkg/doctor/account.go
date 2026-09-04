package doctor

// Read-only account facts: profile credentials, payment-method
// configurations (with capability availability), and the events census.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// dpmCutoff is the API version at/after which removing payment_method_types
// alone enables dynamic payment methods; before it, automatic_payment_methods
// [enabled]=true must be added or methods are silently lost.
const dpmCutoff = "2023-08-16"

type accountFacts struct {
	AccountID   string
	DisplayName string

	// EventVersions maps api_version -> count over the sampled events.
	EventVersions map[string]int
	OldestVersion string

	// PMConfigs summarizes /v1/payment_method_configurations.
	ConfigCount  int
	ActiveConfig string
	MethodsOn    int
	MethodsOff   int
	// EnabledMethods are toggled ON in the chosen config;
	// UnavailableMethods is the subset whose `available` is false (toggle on
	// but capability inactive — they will NOT render; both are required).
	EnabledMethods     []string
	UnavailableMethods []string
	ConfiguredOK       bool
	NoEvents           bool // no recent events: version facts are unknowable
	VersionsOK         bool // every sampled event version >= dpmCutoff
	VersionsMixed      bool // some but not all versions >= dpmCutoff
}

// loadTestCredentials resolves sandbox/test-mode credentials through the
// CLI's own profile machinery: the user access token (UAT) from `stripe
// login` when present, falling back to an API key (which already honors
// STRIPE_API_KEY). Live-mode API keys are refused — every account call here
// is read-only and test-mode by design. UAT credentials need no such check:
// ResolveCredentials(false) already scopes them to test mode via the
// Stripe-Livemode header, erroring out if the active context is live.
func loadTestCredentials(cfg *config.Config) (stripe.Credentials, error) {
	if cfg == nil {
		return stripe.Credentials{}, fmt.Errorf("no CLI configuration available")
	}
	creds, err := cfg.GetProfile().ResolveCredentials(false)
	if err != nil {
		return stripe.Credentials{}, err
	}
	if creds.OAKLivemode == nil && !strings.HasPrefix(creds.Token, "sk_test_") && !strings.HasPrefix(creds.Token, "rk_test_") {
		return stripe.Credentials{}, fmt.Errorf("resolved key is not a test-mode key (sk_test_/rk_test_); refusing")
	}
	return creds, nil
}

// newAccountClient builds the shared stripe.Client used for account.go's
// read-only GETs, so they inherit the CLI's normal request path — unix
// socket / proxy support, standard headers — instead of a bare http.Client.
func newAccountClient(creds stripe.Credentials) (*stripe.Client, error) {
	base, err := url.Parse(stripe.DefaultAPIBaseURL)
	if err != nil {
		return nil, err
	}
	return &stripe.Client{BaseURL: base, Credentials: creds}, nil
}

// stripeGET issues a read-only GET through client, decoding the JSON
// response into out. query is the raw (already-encoded) query string, e.g.
// "limit=20"; it is applied via PerformRequest rather than appended to path
// since PerformRequest overwrites any query already present on path.
func stripeGET(client *stripe.Client, stripeAccount, path, query string, out any) error {
	configure := func(req *http.Request) error {
		client.Credentials.ApplyAccountContextHeaders(req.Header, stripeAccount, "")
		return nil
	}
	resp, err := client.PerformRequest(context.Background(), http.MethodGet, path, query, configure)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("credentials rejected (401) — key may be expired; run `stripe login`")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GET %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchAccountFacts pulls the read-only account evidence. stripeAccount, when
// non-empty, is sent as the Stripe-Account header so Connect direct-charge
// integrations resolve the CONNECTED account's payment-method configuration
// (the platform's own config does not govern those charges).
func fetchAccountFacts(creds stripe.Credentials, stripeAccount string) (*accountFacts, error) {
	client, err := newAccountClient(creds)
	if err != nil {
		return nil, err
	}
	f := &accountFacts{EventVersions: map[string]int{}}

	var acct struct {
		ID       string `json:"id"`
		Settings struct {
			Dashboard struct {
				DisplayName string `json:"display_name"`
			} `json:"dashboard"`
		} `json:"settings"`
	}
	if err := stripeGET(client, stripeAccount, "/v1/account", "", &acct); err != nil {
		return nil, err
	}
	f.AccountID, f.DisplayName = acct.ID, acct.Settings.Dashboard.DisplayName

	// Payment method configurations: each payment-method field is an object
	// with display_preference.value on|off plus `available` (enabled AND the
	// capability is active — what will actually render). Parse generically.
	var pmc struct {
		Data []map[string]any `json:"data"`
	}
	if err := stripeGET(client, stripeAccount, "/v1/payment_method_configurations", "", &pmc); err != nil {
		return nil, err
	}
	// Choose ONE governing config — strictly the default (the one the API
	// uses when payment_method_configuration is not specified), else the
	// first active — and count/collect methods from it alone.
	var chosen map[string]any
	for _, cfg := range pmc.Data {
		active, _ := cfg["active"].(bool)
		if !active {
			continue // deactivated demo leftovers shouldn't inflate anything
		}
		f.ConfigCount++
		isDefault, _ := cfg["is_default"].(bool)
		if isDefault {
			chosen = cfg
		} else if chosen == nil {
			chosen = cfg
		}
	}
	if chosen != nil {
		if name, ok := chosen["name"].(string); ok {
			f.ActiveConfig = name
		}
		for field, v := range chosen {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			dp, ok := m["display_preference"].(map[string]any)
			if !ok {
				continue
			}
			switch dp["value"] {
			case "on":
				f.MethodsOn++
				f.EnabledMethods = append(f.EnabledMethods, field)
				// Toggled on but capability inactive → will NOT render;
				// per DPM guidance both are required.
				if avail, ok := m["available"].(bool); ok && !avail {
					f.UnavailableMethods = append(f.UnavailableMethods, field)
				}
			case "off":
				f.MethodsOff++
			}
		}
		sort.Strings(f.EnabledMethods)
		sort.Strings(f.UnavailableMethods)
	}
	f.ConfiguredOK = f.ConfigCount > 0 && f.MethodsOn > 0

	// Recent events: the API versions live traffic actually uses.
	var evts struct {
		Data []struct {
			APIVersion string `json:"api_version"`
		} `json:"data"`
	}
	if err := stripeGET(client, stripeAccount, "/v1/events", "limit=20", &evts); err != nil {
		return nil, err
	}
	ge := 0
	for _, e := range evts.Data {
		if e.APIVersion == "" {
			continue
		}
		f.EventVersions[e.APIVersion]++
		if f.OldestVersion == "" || datePrefix(e.APIVersion) < datePrefix(f.OldestVersion) {
			f.OldestVersion = e.APIVersion
		}
		if datePrefix(e.APIVersion) >= dpmCutoff {
			ge++
		}
	}
	total := 0
	for _, n := range f.EventVersions {
		total += n
	}
	f.NoEvents = total == 0
	f.VersionsOK = total > 0 && ge == total
	f.VersionsMixed = ge > 0 && ge < total
	return f, nil
}

// datePrefix compares versions by their YYYY-MM-DD prefix, which handles both
// classic ("2023-08-16") and named ("2026-06-24.dahlia") formats.
func datePrefix(v string) string {
	if len(v) >= 10 {
		return v[:10]
	}
	return v
}

// ---------- intent classification ----------
