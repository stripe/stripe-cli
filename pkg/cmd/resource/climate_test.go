package resource

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func buildClimateTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "stripe", Annotations: make(map[string]string)}
	climateCmd := &cobra.Command{Use: "climate", Annotations: make(map[string]string)}
	root.AddCommand(climateCmd)
	root.Annotations["climate"] = "namespace"
	return root
}

func setupCommitmentsCmd(t *testing.T) *cobra.Command {
	t.Helper()
	root := buildClimateTestRoot()
	AddClimateCommitmentsSubCmds(root, &config.Config{})
	for _, c := range root.Commands()[0].Commands() {
		if c.Name() == "commitment" {
			return c
		}
	}
	t.Fatal("commitment command not found")
	return nil
}

func TestAddClimateCommitmentSubCmds_IsHidden(t *testing.T) {
	assert.True(t, setupCommitmentsCmd(t).Hidden)
}

func TestAddClimateCommitmentSubCmds_HasExpectedOperations(t *testing.T) {
	ops := make(map[string]bool)
	for _, c := range setupCommitmentsCmd(t).Commands() {
		ops[c.Name()] = true
	}
	assert.True(t, ops["enable"])
	assert.True(t, ops["show"])
	assert.True(t, ops["disable"])
	assert.Len(t, ops, 3)
}

func TestAddClimateCommitmentSubCmds_EnableTakesPositionalRateArg(t *testing.T) {
	var enableCmd *cobra.Command
	for _, op := range setupCommitmentsCmd(t).Commands() {
		if op.Name() == "enable" {
			enableCmd = op
			break
		}
	}
	require.NotNil(t, enableCmd)

	assert.NoError(t, enableCmd.Args(enableCmd, []string{"1.5"}))
	assert.Error(t, enableCmd.Args(enableCmd, []string{}))
	assert.Error(t, enableCmd.Args(enableCmd, []string{"1.5", "extra"}))
	assert.Nil(t, enableCmd.Flags().Lookup("rate"), "--rate flag should not exist; rate is a positional arg")
}

func TestAddClimateCommitmentSubCmds_ShowAndDisableHaveNoParams(t *testing.T) {
	commitmentCmd := setupCommitmentsCmd(t)
	for _, op := range commitmentCmd.Commands() {
		if op.Name() == "show" || op.Name() == "disable" {
			op.Flags().VisitAll(func(f *pflag.Flag) {
				_, isRequestFlag := f.Annotations["request"]
				assert.False(t, isRequestFlag,
					"%s should have no request params, but found --%s", op.Name(), f.Name)
			})
		}
	}
}

func TestAddClimateCommitmentSubCmds_NilRootDoesNotPanic(t *testing.T) {
	root := &cobra.Command{Use: "stripe", Annotations: make(map[string]string)}
	require.NotPanics(t, func() {
		AddClimateCommitmentsSubCmds(root, &config.Config{})
	})
}

func TestAddClimateCommitmentSubCmds_DoesNotDisplaceExistingClimateCommands(t *testing.T) {
	root := buildClimateTestRoot()
	root.Commands()[0].AddCommand(&cobra.Command{Use: "orders", Annotations: make(map[string]string)})
	AddClimateCommitmentsSubCmds(root, &config.Config{})

	found := false
	for _, c := range root.Commands()[0].Commands() {
		if c.Name() == "orders" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestTransformClimateProgram(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "renames contribution fields",
			input: map[string]interface{}{
				"id":                        "climprog_1TZeA6DJ5fwQRCtH9D8hfkgL",
				"contribution_method":       "rate",
				"contribution_rate_percent": 4.0,
				"status":                    "enabled",
			},
			expected: map[string]interface{}{
				"id":     "climprog_1TZeA6DJ5fwQRCtH9D8hfkgL",
				"type":   "rate",
				"rate":   4.0,
				"status": "enabled",
			},
		},
		{
			name: "drops flat and date fields",
			input: map[string]interface{}{
				"contribution_method":        "rate",
				"contribution_rate_percent":  4.0,
				"contribution_flat_amount":   nil,
				"contribution_flat_currency": nil,
				"initial_start_date":         1779398955.4014118,
				"status":                     "enabled",
			},
			expected: map[string]interface{}{
				"type":   "rate",
				"rate":   4.0,
				"status": "enabled",
			},
		},
		{
			name: "flat program renames amount and currency, drops rate",
			input: map[string]interface{}{
				"id":                         "climprog_flat123",
				"contribution_method":        "flat",
				"contribution_flat_amount":   5000.0,
				"contribution_flat_currency": "usd",
				"contribution_rate_percent":  nil,
				"initial_start_date":         1779398955.4014118,
				"status":                     "enabled",
			},
			expected: map[string]interface{}{
				"id":       "climprog_flat123",
				"type":     "flat",
				"amount":   5000.0,
				"currency": "usd",
				"status":   "enabled",
			},
		},
		{
			name:     "passes through unrelated fields unchanged",
			input:    map[string]interface{}{"object": "climate.program", "livemode": false},
			expected: map[string]interface{}{"object": "climate.program", "livemode": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := json.Marshal(tt.input)
			require.NoError(t, err)

			got, err := transformClimateProgram(input)
			require.NoError(t, err)

			var result map[string]interface{}
			require.NoError(t, json.Unmarshal(got, &result))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformClimateProgram_InvalidJSON(t *testing.T) {
	input := []byte(`not json`)
	got, err := transformClimateProgram(input)
	assert.NoError(t, err)
	assert.Equal(t, input, got)
}
