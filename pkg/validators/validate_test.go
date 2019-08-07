package validators

import (
	"fmt"
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

func TestHTTPMethod(t *testing.T) {
	err := HTTPMethod("GET")
	assert.Nil(t, err)
}

func TestHTTPMethodInvalid(t *testing.T) {
	err := HTTPMethod("invalid")
	assert.Equal(t, "invalid is not an acceptable HTTP method (GET, POST, DELETE)", fmt.Sprintf("%s", err))
}

func TestHTTPMethodLowercase(t *testing.T) {
	err := HTTPMethod("post")
	assert.Nil(t, err)
}

func TestRequestSourceAPI(t *testing.T) {
	err := RequestSource("API")
	assert.Nil(t, err)
}

func TestRequestSourceDashboard(t *testing.T) {
	err := RequestSource("dashboard")
	assert.Nil(t, err)
}

func TestRequestSourceInvalid(t *testing.T) {
	err := RequestSource("invalid")
	assert.Equal(t, "invalid is not an acceptable source (API, DASHBOARD)", fmt.Sprintf("%s", err))
}

func TestStatusCode(t *testing.T) {
	err := StatusCode("200")
	assert.Nil(t, err)
}

func TestStatusCodeUnusedInStripe(t *testing.T) {
	err := StatusCode("300")
	assert.Equal(t, "Provided status code 300 is not in the range of acceptable status codes (200's, 400's, 500's)", fmt.Sprintf("%s", err))
}

func TestStatusCodeType(t *testing.T) {
	err := StatusCodeType("200")
	assert.Nil(t, err)
}

func TestStatusCodeTypeUnusedInStripe(t *testing.T) {
	err := StatusCodeType("300")
	assert.Equal(t, "Provided status code type 300 is not a valid type (200, 400, 500)", fmt.Sprintf("%s", err))
}

func TestStatusCodeNotEvenHundred(t *testing.T) {
	err := StatusCodeType("201")
	assert.Equal(t, "Provided status code type 201 is not a valid type (200, 400, 500)", fmt.Sprintf("%s", err))
}
