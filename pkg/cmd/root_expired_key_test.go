package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

func TestAPIKeyExpiredMessage_DefaultProfile(t *testing.T) {
	msg := apiKeyExpiredMessage("default")
	require.Contains(t, msg, "default profile")
	require.Contains(t, msg, "`stripe login`")
	require.Contains(t, msg, "`stripe whoami`")
	require.NotContains(t, msg, "--project-name")
}

func TestAPIKeyExpiredMessage_NamedProfile(t *testing.T) {
	msg := apiKeyExpiredMessage("work")
	require.Contains(t, msg, `"work"`)
	require.Contains(t, msg, "--project-name=work")
}

func TestAPIKeyExpiredMessage_EmptyProfileDefaultsToDefault(t *testing.T) {
	msg := apiKeyExpiredMessage("")
	require.Contains(t, msg, "default profile")
	require.NotContains(t, msg, `profile ""`)
}

func TestIsAPIKeyExpiredError(t *testing.T) {
	t.Run("structured request error", func(t *testing.T) {
		err := requests.RequestError{
			StatusCode: 401,
			ErrorCode:  "api_key_expired",
		}
		require.True(t, isAPIKeyExpiredError(err))
	})

	t.Run("wrapped request error", func(t *testing.T) {
		err := fmt.Errorf("wrapper: %w", requests.RequestError{
			StatusCode: 401,
			ErrorCode:  "api_key_expired",
		})
		require.True(t, isAPIKeyExpiredError(err))
	})

	t.Run("plain text plugin error", func(t *testing.T) {
		err := errors.New(`rpc error: code = Unknown desc = {"error":{"code":"api_key_expired","message":"Expired API Key provided: rk_test_***123"}}`)
		require.True(t, isAPIKeyExpiredError(err))
	})

	t.Run("other error", func(t *testing.T) {
		require.False(t, isAPIKeyExpiredError(errors.New("boom")))
	})
}
