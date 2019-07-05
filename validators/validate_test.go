package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLegacyAPIKeys(t *testing.T) {
	err := APIKey("sk_123457890abcdef")
	assert.EqualError(t, err, "you are using a legacy-style API key which is unsupported by the CLI. Please generate a new test mode API key")
}

func TestPublishableAPIKey(t *testing.T) {
	err := APIKey("pk_test_12345")
	assert.EqualError(t, err, "the CLI only supports using a secret or restricted key")
}

func TestLivemodeAPIKey(t *testing.T) {
	err := APIKey("sk_live_12345")
	assert.EqualError(t, err, "the CLI only supports using a test mode key")
}

func TestTestmodeAPIKey(t *testing.T) {
	err := APIKey("sk_test_12345")
	assert.Nil(t, err)
}

func TestTestmodeRestrictedAPIKey(t *testing.T) {
	err := APIKey("rk_test_12345")
	assert.Nil(t, err)
}
