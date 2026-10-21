package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loaderFunc func(context.Context, string) (Catalog, error)

func (f loaderFunc) Load(c context.Context, p string) (Catalog, error) { return f(c, p) }

type discoveryFunc func(context.Context, string) (Discovery, error)

func (f discoveryFunc) Discover(c context.Context, p string) (Discovery, error) { return f(c, p) }

type configFunc func(context.Context, FlowContext) ([]Configuration, error)

func (f configFunc) Configurations(c context.Context, p FlowContext) ([]Configuration, error) {
	return f(c, p)
}

func testServices(t *testing.T) Services {
	t.Helper()
	return Services{Predicates: MockLoader{}, Discovery: discoveryFunc(func(context.Context, string) (Discovery, error) {
		return Discovery{Snapshot: "snapshot", Flows: []SourceFlow{{ID: "flow_a", Method: "POST", Path: "/v1/payment_intents", Pinned: Methods{Status: "known", Values: []string{"card", "klarna"}}, Automatic: "absent", Context: FlowContext{Account: "profile", Mode: "test", Configuration: "configuration_a"}}}}, nil
	}), Configurations: func(context.Context) (ConfigurationClient, error) {
		return configFunc(func(context.Context, FlowContext) ([]Configuration, error) {
			return []Configuration{configuration("ideal")}, nil
		}), nil
	}}
}

func TestOfflineNeverConstructsClient(t *testing.T) {
	s := testServices(t)
	s.Configurations = func(context.Context) (ConfigurationClient, error) {
		t.Fatal("offline constructed network or credential client")
		return nil, nil
	}
	r, err := s.Analyze(context.Background(), Options{Offline: true})
	require.NoError(t, err)
	require.Len(t, r.Results, 1)
	assert.Nil(t, r.Results[0].Configuration)
	assert.Equal(t, "needs_discovery", r.Results[0].Status)
	assert.Contains(t, r.Input, "mocked")
	assert.Equal(t, "offline_configuration_unknown", r.Results[0].Diagnostics[0].Code)
}

func TestNewDeclarationsDoNotFallback(t *testing.T) {
	s := testServices(t)
	c, err := ParseCatalog(mockPredicates)
	require.NoError(t, err)
	c.Remediations[0].Version = 99
	c.Fixes = []LegacyFix{{ID: "old", Kind: "removeParam", Param: "payment_method_types"}}
	c.Rules[0].FixIDs = []string{"old"}
	s.Predicates = loaderFunc(func(context.Context, string) (Catalog, error) { return c, nil })
	s.Configurations = func(context.Context) (ConfigurationClient, error) {
		t.Fatal("unsupported handler attempted account lookup")
		return nil, nil
	}
	r, err := s.Analyze(context.Background(), Options{})
	require.NoError(t, err)
	require.Len(t, r.Results, 1)
	assert.Equal(t, "unsupported", r.Results[0].Status)
	assert.Equal(t, 99, r.Results[0].Provenance.Version)
	assert.Equal(t, "payment_methods.adopt_dynamic", r.Results[0].Provenance.Kind)
	c.Rules[0].RemediationIDs = nil
	r, err = s.Analyze(context.Background(), Options{})
	require.NoError(t, err)
	require.Len(t, r.Results, 1)
	assert.Equal(t, "legacy_path", r.Results[0].Diagnostics[0].Code)
	assert.Equal(t, "removeParam", r.Results[0].Legacy.Kind)
	assert.Equal(t, "payment_method_types", r.Results[0].Legacy.Param)
}

func TestSeparateDeclarationsAndEndpointSelection(t *testing.T) {
	s := testServices(t)
	c, err := ParseCatalog(mockPredicates)
	require.NoError(t, err)
	other := c.Remediations[0]
	other.ID = "second"
	other.Version = 2
	c.Remediations = append(c.Remediations, other)
	c.Rules[0].RemediationIDs = append(c.Rules[0].RemediationIDs, "second")
	s.Predicates = loaderFunc(func(context.Context, string) (Catalog, error) { return c, nil })
	r, err := s.Analyze(context.Background(), Options{})
	require.NoError(t, err)
	require.Len(t, r.Results, 2)
	assert.Equal(t, "needs_review", r.Results[0].Status)
	assert.Equal(t, "unsupported", r.Results[1].Status)
	assert.NotEqual(t, r.Results[0].Provenance.Declaration, r.Results[1].Provenance.Declaration)
	assert.Equal(t, "intent_pin", r.Results[0].Provenance.Finding)
}

func TestLiveContextAndLookupFailures(t *testing.T) {
	for _, flowContext := range []FlowContext{{Account: "unknown", Mode: "unknown"}, {Account: "profile", Mode: "test"}, {Account: "connect", Mode: "test", Configuration: "configuration_a"}} {
		s := testServices(t)
		discover := s.Discovery
		s.Discovery = discoveryFunc(func(ctx context.Context, p string) (Discovery, error) {
			d, err := discover.Discover(ctx, p)
			d.Flows[0].Context = flowContext
			return d, err
		})
		s.Configurations = func(context.Context) (ConfigurationClient, error) {
			t.Fatal("unestablished context fetched configuration")
			return nil, nil
		}
		r, err := s.Analyze(context.Background(), Options{})
		require.NoError(t, err)
		assert.Equal(t, "needs_discovery", r.Results[0].Status)
	}
	s := testServices(t)
	s.Configurations = func(context.Context) (ConfigurationClient, error) { return nil, errors.New("private") }
	r, err := s.Analyze(context.Background(), Options{})
	require.NoError(t, err)
	assert.Equal(t, "credentials_unavailable", r.Results[0].Diagnostics[0].Code)
}

func TestCommandJSONAndExitBehavior(t *testing.T) {
	s := testServices(t)
	cmd := NewCommand(s)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"dpm", ".", "--offline", "--json"})
	err := cmd.Execute()
	require.ErrorIs(t, err, ErrIncomplete)
	var r Report
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &r))
	assert.Empty(t, stderr.String())
	assert.Equal(t, "needs_discovery", r.Status)
	stdout.Reset()
	s.Predicates = loaderFunc(func(context.Context, string) (Catalog, error) { return Catalog{}, errors.New("predicate invalid") })
	cmd = NewCommand(s)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--offline"})
	require.Error(t, cmd.Execute())
	var output map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	assert.Equal(t, "invalid", output["status"])
}

func TestConditionalAndMatchingSetsNeverAuthorizeEdits(t *testing.T) {
	c, err := ParseCatalog(mockPredicates)
	require.NoError(t, err)
	f := SourceFlow{Pinned: Methods{Status: "conditional", Alternatives: [][]string{{"card"}, {"ideal"}}}}
	r := newResult(c, c.Rules[0], c.Remediations[0], f)
	evaluate(&r, configuration("card"), "unknown", Coverage{Truncated: true})
	assert.Equal(t, "needs_discovery", r.Status)
	f.Pinned = Methods{Status: "known", Values: []string{"card"}}
	r = newResult(c, c.Rules[0], c.Remediations[0], f)
	evaluate(&r, configuration("card"), "yes", Coverage{})
	assert.Equal(t, "needs_discovery", r.Status, "matching sets do not establish API/frontend prerequisites")
	f.Path = "/v1/payment_intents"
	f.Context.APIVersion = "2022-01-01"
	task := paymentMethodProposal(f)
	assert.Contains(t, task.Instruction, "pin removal alone is insufficient")
}

func TestOfflineStartupAndSanitization(t *testing.T) {
	assert.True(t, IsOfflineInvocation([]string{"doctor", "dpm", ".", "--offline", "--json"}))
	assert.True(t, IsOfflineInvocation([]string{"--project-name", "example", "doctor", "--offline=true"}))
	assert.True(t, IsOfflineInvocation([]string{"doctor", "--offline", "--offline=false"}))
	assert.False(t, IsOfflineInvocation([]string{"get", "--offline"}))
	cmd := NewCommand(testServices(t))
	var out bytes.Buffer
	cmd.SetOut(&out)
	secret := "sk" + "_test_" + "examplevalue"
	require.NoError(t, writeJSON(cmd, map[string]string{"value": secret + "\x1b[31m<script>"}))
	assert.NotContains(t, out.String(), secret)
	assert.NotContains(t, out.String(), "\\u001b")
	assert.Contains(t, out.String(), "[redacted]")
	assert.NotContains(t, out.String(), "<script>")
}
