package resources_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/cmd/resources"
	"github.com/stripe/stripe-cli/pkg/config"
)

// TestSpecVarExistence verifies that well-known OperationSpec vars are present.
func TestSpecVarExistence(t *testing.T) {
	assert.Equal(t, "create", resources.V1AccountLinksCreate.Name)
	assert.Equal(t, "/v1/account_links", resources.V1AccountLinksCreate.Path)
	assert.Equal(t, "POST", resources.V1AccountLinksCreate.Method)
	assert.False(t, resources.V1AccountLinksCreate.IsPreview)
}

// TestAccountParam verifies that a required param is correctly populated.
func TestAccountParam(t *testing.T) {
	spec := resources.V1AccountLinksCreate
	require.NotNil(t, spec.Params)
	accountParam, ok := spec.Params["account"]
	require.True(t, ok, "expected 'account' param in V1AccountLinksCreate")
	assert.Equal(t, "string", accountParam.Type)
	assert.True(t, accountParam.Required, "account param should be required")
}

// TestTypeParam verifies that a required enum param with values is correctly populated.
func TestTypeParam(t *testing.T) {
	spec := resources.V1AccountLinksCreate
	require.NotNil(t, spec.Params)
	typeParam, ok := spec.Params["type"]
	require.True(t, ok, "expected 'type' param in V1AccountLinksCreate")
	assert.Equal(t, "string", typeParam.Type)
	assert.True(t, typeParam.Required, "type param should be required")
	require.Greater(t, len(typeParam.Enum), 0, "type param should have enum values")

	found := false
	for _, ev := range typeParam.Enum {
		if ev.Value == "account_onboarding" {
			found = true
			break
		}
	}
	assert.True(t, found, "account_onboarding should be in type enum values")
}

// TestGeneratedSpecWiresFlags verifies that NewOperationCmd using a generated OperationSpec
// registers cobra flags matching the spec's params.
func TestGeneratedSpecWiresFlags(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	oc := resource.NewOperationCmd(parentCmd, &resources.V1AccountLinksCreate, &config.Config{})

	require.Equal(t, "create", oc.Name)
	require.Equal(t, http.MethodPost, oc.HTTPVerb)

	// Required string param
	_, err := oc.Cmd.Flags().GetString("account")
	require.NoError(t, err, "expected --account flag to be registered")

	// Required enum param
	_, err = oc.Cmd.Flags().GetString("type")
	require.NoError(t, err, "expected --type flag to be registered")

	// Optional string param
	_, err = oc.Cmd.Flags().GetString("return-url")
	require.NoError(t, err, "expected --return-url flag to be registered")
}

// TestGeneratedGetSpecWiresFlags verifies a GET operation spec wires query params as flags.
func TestGeneratedGetSpecWiresFlags(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	oc := resource.NewOperationCmd(parentCmd, &resources.V1CustomersList, &config.Config{})

	require.Equal(t, "list", oc.Name)
	require.Equal(t, http.MethodGet, oc.HTTPVerb)

	_, err := oc.Cmd.Flags().GetString("email")
	require.NoError(t, err, "expected --email flag to be registered")
}

// TestGeneratedV2SpecWiresFlags verifies a v2 operation spec is present and wires correctly.
func TestGeneratedV2SpecWiresFlags(t *testing.T) {
	spec := resources.V2BillingMeterEventsCreate
	assert.Equal(t, "create", spec.Name)
	assert.Equal(t, "POST", spec.Method)
	require.NotNil(t, spec.Params)

	parentCmd := &cobra.Command{Annotations: make(map[string]string)}
	oc := resource.NewOperationCmd(parentCmd, &spec, &config.Config{})
	require.Equal(t, "create", oc.Name)
}

// TestMostCommonSubField_NonRequiredParent verifies that sub-fields of a mostCommon object
// that is NOT top-level required are tagged MostCommon=true but Required=false.
// This ensures they appear in the common usage example but not the required-fields example.
// Concretely: prices create has "recurring" as mostCommon but not required;
// "recurring.interval" is required within recurring and should be MostCommon, not Required.
func TestMostCommonSubField_NonRequiredParent(t *testing.T) {
	spec := resources.V1PricesCreate
	require.NotNil(t, spec.Params)

	interval, ok := spec.Params["recurring.interval"]
	require.True(t, ok, "expected 'recurring.interval' param in V1PricesCreate")
	assert.False(t, interval.Required, "recurring.interval should not be Required (parent 'recurring' is not top-level required)")
	assert.True(t, interval.MostCommon, "recurring.interval should be MostCommon (locally required within a mostCommon parent)")
}

// TestMostCommonSubField_RequiredParent verifies that sub-fields of a mostCommon object
// that IS top-level required are tagged both Required=true and MostCommon=true.
// Concretely: apps/secrets create has "scope" as both top-level required and mostCommon;
// "scope.type" is required within scope.
func TestMostCommonSubField_RequiredParent(t *testing.T) {
	spec := resources.V1AppsSecretsCreate
	require.NotNil(t, spec.Params)

	scopeType, ok := spec.Params["scope.type"]
	require.True(t, ok, "expected 'scope.type' param in V1AppsSecretsCreate")
	assert.True(t, scopeType.Required, "scope.type should be Required (parent 'scope' is top-level required)")
	assert.True(t, scopeType.MostCommon, "scope.type should be MostCommon (locally required within a mostCommon parent)")
}

// TestDeeplyNestedField_NotRequired verifies that a deeply nested field (depth 2+) whose
// ancestor chain includes a non-required object is NOT marked Required, even if it is
// required within its immediate parent. This is the bug fixed by the parentRequired chain.
func TestDeeplyNestedField_NotRequired(t *testing.T) {
	spec := resources.V1PaymentIntentsCreate
	require.NotNil(t, spec.Params)

	// hooks is not top-level required; any nested field should have Required=false
	// regardless of how many levels of local required-ness exist.
	for name, ps := range spec.Params {
		if strings.HasPrefix(name, "hooks.") {
			assert.False(t, ps.Required, "param %q under non-required 'hooks' should not be Required", name)
		}
	}
}

// TestClearableObjectParam verifies that a known clearable-object field is generated
// with Type "clearable_object" so the CLI can offer the right help text and translate
// "{}" to "" when sending the request.
func TestClearableObjectParam(t *testing.T) {
	spec := resources.V1CustomersUpdate
	require.NotNil(t, spec.Params)

	shipping, ok := spec.Params["shipping"]
	require.True(t, ok, "expected 'shipping' param in V1CustomersUpdate")
	assert.Equal(t, "clearable_object", shipping.Type, "shipping should be a clearable_object (anyOf: object | empty string)")
}

// TestAddResourceCmds verifies the coordinator registers commands without panicking.
func TestAddResourceCmds(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:         "stripe",
		Annotations: make(map[string]string),
	}
	require.NotPanics(t, func() {
		resources.AddAllResourcesCmds(rootCmd, &config.Config{})
	})
	require.True(t, rootCmd.HasSubCommands())
}
