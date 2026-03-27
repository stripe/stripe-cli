package resources_test

import (
	"net/http"
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
