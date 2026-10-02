package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAPIKeyReturnsHelpfulErrorForInvalidEnvVar(t *testing.T) {
	t.Setenv("STRIPE_API_KEY", "pk_test_1234567890")

	p := Profile{
		APIKey: "sk_test_1234567890",
	}

	_, err := p.GetAPIKey(false)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid STRIPE_API_KEY environment variable")
	require.ErrorContains(t, err, "takes precedence over the CLI config file")
	require.ErrorContains(t, err, "the CLI only supports using a secret or restricted key")
}

func TestGetAPIKeyUsesEnvVarWhenValid(t *testing.T) {
	t.Setenv("STRIPE_API_KEY", "sk_test_abcdef123456")

	p := Profile{
		APIKey: "sk_test_1234567890",
	}

	key, err := p.GetAPIKey(false)
	require.NoError(t, err)
	require.Equal(t, "sk_test_abcdef123456", key)
}
