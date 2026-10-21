package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

type MethodPreference struct {
	Available bool   `json:"available"`
	Stored    string `json:"stored"`
	Effective string `json:"effective"`
}

type Configuration struct {
	ID        string                      `json:"id"`
	Active    bool                        `json:"active"`
	Default   bool                        `json:"default"`
	Livemode  bool                        `json:"livemode"`
	Parent    string                      `json:"parent,omitempty"`
	Methods   map[string]MethodPreference `json:"methods"`
	FetchedAt time.Time                   `json:"fetchedAt"`
	Complete  bool                        `json:"complete"`
}

type ConfigurationClient interface {
	Configurations(context.Context, FlowContext) ([]Configuration, error)
}

// APIConfigurations uses only published read-only configuration endpoints.
// Credentials are resolved lazily, so offline execution cannot refresh credentials.
type APIConfigurations struct {
	Client      stripe.RequestPerformer
	Credentials stripe.Credentials
}

func (a APIConfigurations) Configurations(ctx context.Context, flow FlowContext) ([]Configuration, error) {
	if flow.Account != "profile" || flow.Mode == "unknown" || (flow.Mode == "live") != a.Credentials.Livemode() {
		return nil, errors.New("account or mode not established for these credentials")
	}
	result := []Configuration{}
	cursor := ""
	seen := map[string]bool{}
	for page := 0; page < 10; page++ {
		query := url.Values{"limit": []string{"100"}}
		if cursor != "" {
			query.Set("starting_after", cursor)
		}
		resp, err := a.Client.PerformRequest(ctx, http.MethodGet, "/v1/payment_method_configurations", query.Encode(), func(req *http.Request) error {
			a.Credentials.ApplyAccountContextHeaders(req.Header, "", "")
			return nil
		})
		if err != nil {
			return nil, errors.New("configuration request failed")
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxInput+1))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || readErr != nil || len(data) > maxInput {
			return nil, errors.New("configuration permission or response failure")
		}
		var list struct {
			Data    []json.RawMessage `json:"data"`
			HasMore *bool             `json:"has_more"`
		}
		if json.Unmarshal(data, &list) != nil || list.HasMore == nil || list.Data == nil {
			return nil, errors.New("incomplete configuration response")
		}
		for _, raw := range list.Data {
			c, err := decodeConfiguration(raw)
			if err != nil {
				return nil, err
			}
			if seen[c.ID] {
				return nil, errors.New("duplicate pagination result")
			}
			seen[c.ID] = true
			result = append(result, c)
			cursor = c.ID
		}
		if !*list.HasMore {
			return result, nil
		}
		if len(list.Data) == 0 {
			return nil, errors.New("empty pagination page")
		}
	}
	return nil, errors.New("configuration pagination budget exhausted")
}

func decodeConfiguration(raw []byte) (Configuration, error) {
	var meta struct {
		ID      string `json:"id"`
		Active  *bool  `json:"active"`
		Default *bool  `json:"is_default"`
		Live    *bool  `json:"livemode"`
		Parent  string `json:"parent"`
	}
	if json.Unmarshal(raw, &meta) != nil || meta.ID == "" || len(meta.ID) > 128 || meta.Active == nil || meta.Default == nil || meta.Live == nil {
		return Configuration{}, errors.New("incomplete configuration metadata")
	}
	c := Configuration{ID: meta.ID, Active: *meta.Active, Default: *meta.Default, Livemode: *meta.Live, Parent: meta.Parent, Methods: map[string]MethodPreference{}, FetchedAt: time.Now().UTC(), Complete: true}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return c, errors.New("invalid configuration")
	}
	for key, value := range fields {
		var field map[string]json.RawMessage
		if json.Unmarshal(value, &field) != nil || field["display_preference"] == nil {
			continue
		}
		var method struct {
			Available *bool `json:"available"`
			Display   struct {
				Preference string `json:"preference"`
				Value      string `json:"value"`
			} `json:"display_preference"`
		}
		if json.Unmarshal(value, &method) != nil || method.Available == nil || !identifier.MatchString(key) || (method.Display.Value != "on" && method.Display.Value != "off") || (method.Display.Preference != "on" && method.Display.Preference != "off" && method.Display.Preference != "none") {
			return c, errors.New("incomplete payment method preferences")
		}
		c.Methods[key] = MethodPreference{Available: *method.Available, Stored: method.Display.Preference, Effective: method.Display.Value}
	}
	if len(c.Methods) == 0 {
		return c, errors.New("no payment method evidence")
	}
	return c, nil
}

// selectConfiguration never guesses the first active configuration.
func selectConfiguration(flow FlowContext, configs []Configuration) (*Configuration, error) {
	if flow.Account != "profile" || (flow.Mode != "test" && flow.Mode != "live") {
		return nil, errors.New("Connect, inheritance or account/mode context requires discovery")
	}
	if flow.Configuration == "" && !flow.DefaultEstablished {
		return nil, errors.New("applicable default is not established for this flow")
	}
	var chosen *Configuration
	for _, c := range configs {
		matches := flow.Configuration != "" && c.ID == flow.Configuration || flow.Configuration == "" && c.Default
		if !matches {
			continue
		}
		if chosen != nil {
			return nil, errors.New("ambiguous configuration selection")
		}
		copy := c
		chosen = &copy
	}
	if chosen == nil || !chosen.Active || !chosen.Complete || chosen.Livemode != (flow.Mode == "live") || chosen.Parent != "" || len(chosen.Methods) == 0 || len(chosen.Methods) > 256 || chosen.FetchedAt.IsZero() || time.Since(chosen.FetchedAt) > 5*time.Minute || time.Until(chosen.FetchedAt) > time.Minute {
		return nil, errors.New("missing, inactive, incomplete, inherited or mode-mismatched configuration")
	}
	return chosen, nil
}

func differences(pinned []string, config Configuration) (lost, added []string) {
	on := map[string]bool{}
	pins := map[string]bool{}
	for name, m := range config.Methods {
		if m.Available && m.Effective == "on" {
			on[name] = true
		}
	}
	lost, added = []string{}, []string{}
	for _, m := range pinned {
		pins[m] = true
		if !on[m] {
			lost = append(lost, m)
		}
	}
	for m := range on {
		if !pins[m] {
			added = append(added, m)
		}
	}
	sort.Strings(lost)
	sort.Strings(added)
	return
}
