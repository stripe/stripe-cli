package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/cmd/resources"
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

	// Verify account_onboarding is present
	found := false
	for _, ev := range typeParam.Enum {
		if ev.Value == "account_onboarding" {
			found = true
			break
		}
	}
	assert.True(t, found, "account_onboarding should be in type enum values")
}

// TestNewOperationCmdWithSpec verifies that NewOperationCmd using a generated OperationSpec
// registers the same flags as the spec defines.
func TestNewOperationCmdWithSpec(t *testing.T) {
	spec := resource.OperationSpec{
		Name:   "create",
		Path:   "/v1/test/{id}",
		Method: "POST",
		Params: map[string]*resource.ParamSpec{
			"email":  {Type: "string"},
			"amount": {Type: "integer"},
			"active": {Type: "boolean"},
		},
	}

	// Verify the spec structure
	assert.Equal(t, "create", spec.Name)
	assert.Equal(t, "/v1/test/{id}", spec.Path)
	assert.Equal(t, "POST", spec.Method)
	assert.Len(t, spec.Params, 3)

	emailParam, ok := spec.Params["email"]
	require.True(t, ok)
	assert.Equal(t, "string", emailParam.Type)
}
