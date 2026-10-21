package doctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockLoader(t *testing.T) {
	c, err := (MockLoader{}).Load(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, c.Rules, 5)
	assert.Len(t, c.Revision, 64)
	assert.NoError(t, contract(c.Remediations[0]))
	p := filepath.Join(t.TempDir(), "predicate.json")
	require.NoError(t, os.WriteFile(p, mockPredicates, 0600))
	local, err := (MockLoader{}).Load(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, c, local)
}

func TestRejectMalformedCatalog(t *testing.T) {
	for name, data := range map[string]string{
		"nullRefs":  strings.Replace(string(mockPredicates), `"doctorRemediationIds": ["adopt_dynamic_payment_methods"]`, `"doctorRemediationIds": null`, 1),
		"malformed": "{", "trailing": string(mockPredicates) + "{}", "oversize": strings.Repeat(" ", maxInput+1),
		"mixed":        strings.Replace(string(mockPredicates), "doctorRemediations", "doctor_remediations", 1),
		"duplicateKey": strings.Replace(string(mockPredicates), `"topic": "dpm"`, `"topic":"dpm","topic":"dpm"`, 1),
		"dangling":     strings.Replace(string(mockPredicates), `"doctorRemediationIds": ["adopt_dynamic_payment_methods"]`, `"doctorRemediationIds": ["missing"]`, 1),
		"method":       strings.Replace(string(mockPredicates), `"method": "POST"`, `"method": "GET"`, 1),
		"operator":     strings.Replace(string(mockPredicates), `"operator": "exists"`, `"operator": "evaluate"`, 1),
		"path":         strings.Replace(string(mockPredicates), `"path": "/v1/payment_intents"`, `"path": "/v1/payment_intents/confirm"`, 1),
		"emptyRefs":    strings.Replace(string(mockPredicates), `"doctorRemediationIds": ["adopt_dynamic_payment_methods"]`, `"doctorRemediationIds": []`, 1),
	} {
		t.Run(name, func(t *testing.T) { _, err := ParseCatalog([]byte(data)); require.Error(t, err) })
	}
	c, err := ParseCatalog(mockPredicates)
	require.NoError(t, err)
	c.Remediations = append(c.Remediations, c.Remediations[0])
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	_, err = ParseCatalog(raw)
	require.Error(t, err)
}

func TestExactContract(t *testing.T) {
	for _, d := range []Declaration{
		{Kind: "payment_methods.adopt_dynamic", Version: 2, Options: json.RawMessage(`{"offeringPolicy":"preserve_existing"}`)},
		{Kind: "payment_methods.other", Version: 1, Options: json.RawMessage(`{"offeringPolicy":"preserve_existing"}`)},
		{Kind: "payment_methods.adopt_dynamic", Version: 1, Options: json.RawMessage(`{"offeringPolicy":"expand"}`)},
		{Kind: "payment_methods.adopt_dynamic", Version: 1, Options: json.RawMessage(`{"offeringPolicy":"preserve_existing","execute":"arbitrary"}`)},
	} {
		require.Error(t, contract(d))
	}
}

func TestRequestDistinctions(t *testing.T) {
	c, err := ParseCatalog(mockPredicates)
	require.NoError(t, err)
	flow := SourceFlow{Pinned: Methods{Status: "known", Values: []string{"card"}}, Automatic: "absent", Context: FlowContext{APIVersion: "2022-01-01"}}
	assert.Equal(t, "yes", match(c.Rules[0].Conditions, flow))
	assert.Equal(t, "no", match(c.Rules[3].Conditions, flow))
	flow.Pinned.Status = "absent"
	flow.Pinned.Values = nil
	assert.Equal(t, "no", match(c.Rules[0].Conditions, flow))
	assert.Equal(t, "yes", match(c.Rules[3].Conditions, flow))
	flow.Automatic = "enabled"
	assert.Equal(t, "no", match(c.Rules[3].Conditions, flow))
	flow.Automatic = "disabled"
	flow.Context.APIVersion = ""
	assert.Equal(t, "unknown", match(c.Rules[3].Conditions, flow))
	flow.Context.APIVersion = "2024-01-01"
	assert.Equal(t, "no", match(c.Rules[3].Conditions, flow))
	flow.Pinned.Status = "conditional"
	assert.Equal(t, "unknown", match(c.Rules[0].Conditions, flow))
}
