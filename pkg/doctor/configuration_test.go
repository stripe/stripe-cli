package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func configuration(methods ...string) Configuration {
	c := Configuration{ID: "configuration_a", Active: true, Complete: true, Methods: map[string]MethodPreference{}, FetchedAt: time.Now().UTC()}
	for _, m := range methods {
		c.Methods[m] = MethodPreference{Available: true, Stored: "on", Effective: "on"}
	}
	return c
}

func TestPerFlowConfigurationSelection(t *testing.T) {
	a, b := configuration("card"), configuration("ideal")
	b.ID = "configuration_b"
	flow := FlowContext{Account: "profile", Mode: "test", Configuration: a.ID}
	selected, err := selectConfiguration(flow, []Configuration{b, a})
	require.NoError(t, err)
	assert.Equal(t, a.ID, selected.ID)
	flow.Configuration = ""
	_, err = selectConfiguration(flow, []Configuration{a, b})
	assert.Error(t, err)
	flow.DefaultEstablished = true
	_, err = selectConfiguration(flow, []Configuration{a, b})
	assert.Error(t, err, "must not choose first active")
	b.Default = true
	selected, err = selectConfiguration(flow, []Configuration{a, b})
	require.NoError(t, err)
	assert.Equal(t, b.ID, selected.ID)
	a.Default = true
	_, err = selectConfiguration(flow, []Configuration{a, b})
	assert.Error(t, err)
	for _, mutate := range []func(*Configuration){func(c *Configuration) { c.Active = false }, func(c *Configuration) { c.Complete = false }, func(c *Configuration) { c.Parent = "parent_configuration" }, func(c *Configuration) { c.Livemode = true }, func(c *Configuration) { c.FetchedAt = time.Now().Add(-10 * time.Minute) }} {
		c := configuration("card")
		c.Default = true
		mutate(&c)
		_, err := selectConfiguration(flow, []Configuration{c})
		assert.Error(t, err)
	}
	flow.Account = "connect"
	_, err = selectConfiguration(flow, []Configuration{b})
	assert.Error(t, err)
}

func TestPreserveExistingBothDirections(t *testing.T) {
	cfg := configuration("ideal")
	lost, added := differences([]string{"card", "klarna"}, cfg)
	assert.Equal(t, []string{"card", "klarna"}, lost)
	assert.Equal(t, []string{"ideal"}, added)
	cfg = configuration("card", "ideal", "klarna")
	cfg.Methods["card"] = MethodPreference{Available: false, Stored: "on", Effective: "on"}
	cfg.Methods["klarna"] = MethodPreference{Available: true, Stored: "none", Effective: "off"}
	lost, added = differences([]string{"card", "klarna"}, cfg)
	assert.Equal(t, []string{"card", "klarna"}, lost)
	assert.Equal(t, []string{"ideal"}, added)
	cfg.Methods["klarna"] = MethodPreference{Available: true, Stored: "none", Effective: "on"}
	lost, _ = differences([]string{"klarna"}, cfg)
	assert.Empty(t, lost, "effective preference is separate from stored inheritance preference")
}

type performer func(context.Context, string, string, string, func(*http.Request) error) (*http.Response, error)

func (p performer) PerformRequest(c context.Context, m, path, params string, configure func(*http.Request) error) (*http.Response, error) {
	return p(c, m, path, params, configure)
}

func wireConfig(id string) string {
	return `{"id":"` + id + `","active":true,"is_default":true,"livemode":false,"card":{"available":true,"display_preference":{"preference":"none","value":"on"}}}`
}

func TestReadOnlyPagination(t *testing.T) {
	calls := 0
	api := APIConfigurations{Client: performer(func(ctx context.Context, method, path, params string, configure func(*http.Request) error) (*http.Response, error) {
		calls++
		assert.Equal(t, "GET", method)
		assert.Equal(t, "/v1/payment_method_configurations", path)
		assert.NotContains(t, path, "events")
		req, _ := http.NewRequest(method, "https://api.stripe.com"+path, nil)
		require.NoError(t, configure(req))
		assert.Empty(t, req.Header.Get("Stripe-Version"))
		body := `{"data":[` + wireConfig("configuration_a") + `],"has_more":true}`
		if calls == 2 {
			assert.Contains(t, params, "starting_after=configuration_a")
			body = `{"data":[` + wireConfig("configuration_b") + `],"has_more":false}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	configs, err := api.Configurations(context.Background(), FlowContext{Account: "profile", Mode: "test"})
	require.NoError(t, err)
	assert.Len(t, configs, 2)
	assert.Equal(t, 2, calls)
	assert.Equal(t, MethodPreference{Available: true, Stored: "none", Effective: "on"}, configs[0].Methods["card"])
}

func TestConfigurationFailures(t *testing.T) {
	for name, body := range map[string]string{"missingHasMore": `{"data":[]}`, "emptyPage": `{"data":[],"has_more":true}`, "missingMeta": `{"data":[{"id":"configuration_a"}],"has_more":false}`, "badMethod": `{"data":[` + strings.Replace(wireConfig("configuration_a"), `"available":true,`, "", 1) + `],"has_more":false}`} {
		t.Run(name, func(t *testing.T) {
			api := APIConfigurations{Client: performer(func(context.Context, string, string, string, func(*http.Request) error) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			_, err := api.Configurations(context.Background(), FlowContext{Account: "profile", Mode: "test"})
			require.Error(t, err)
		})
	}
	for _, status := range []int{401, 403, 500} {
		api := APIConfigurations{Client: performer(func(context.Context, string, string, string, func(*http.Request) error) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("private response"))}, nil
		})}
		_, err := api.Configurations(context.Background(), FlowContext{Account: "profile", Mode: "test"})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "private response")
	}
	api := APIConfigurations{Client: performer(func(context.Context, string, string, string, func(*http.Request) error) (*http.Response, error) {
		return nil, errors.New("private transport details")
	})}
	_, err := api.Configurations(context.Background(), FlowContext{Account: "profile", Mode: "test"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "private")
	_, err = api.Configurations(context.Background(), FlowContext{Account: "profile", Mode: "live"})
	assert.Error(t, err)
}

func TestRequiredRegressionResult(t *testing.T) {
	c, err := ParseCatalog(mockPredicates)
	require.NoError(t, err)
	f := SourceFlow{Pinned: Methods{Status: "known", Values: []string{"card", "klarna"}}}
	r := newResult(c, c.Rules[0], c.Remediations[0], f)
	evaluate(&r, configuration("ideal"), "yes", Coverage{})
	assert.Equal(t, "needs_review", r.Status)
	assert.Equal(t, []string{"card", "klarna"}, r.Lost)
	assert.Equal(t, []string{"ideal"}, r.Added)
	data, err := json.Marshal(r.Tasks)
	require.NoError(t, err)
	for _, forbidden := range []string{"remove payment_method_types", "replace payment_method_types", "enable ideal"} {
		assert.NotContains(t, string(data), forbidden)
	}
}

var _ stripe.RequestPerformer = performer(nil)
