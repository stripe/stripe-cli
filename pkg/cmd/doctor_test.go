package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/doctor"
)

func TestDoctorRegistered(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"doctor"})
	require.NoError(t, err)
	require.NotNil(t, command)
	assert.Equal(t, "doctor", command.Name())
	assert.NotNil(t, command.Flags().Lookup("offline"))
}

func TestOfflineDoctorSkipsRootNetworkSetup(t *testing.T) {
	command := doctor.NewCommand(doctor.Services{})
	require.NoError(t, command.Flags().Set("offline", "true"))
	oldBase := rootAccessBaseURL
	rootAccessBaseURL = "invalid"
	t.Cleanup(func() { rootAccessBaseURL = oldBase })
	// Invalid access configuration would fail the normal pre-run before any lookup.
	require.NoError(t, rootCmd.PersistentPreRunE(command, nil))
}
