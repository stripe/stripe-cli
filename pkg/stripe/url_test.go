package stripe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAPIBaseURLWorks(t *testing.T) {
	assert.Nil(t, ValidateAPIBaseUrl("https://api.stripe.com"))
	assert.Nil(t, ValidateAPIBaseUrl("https://qa-api.stripe.com"))
	assert.Nil(t, ValidateAPIBaseUrl("http://foo-api-mydev.dev.stripe.me"))
	assert.Nil(t, ValidateAPIBaseUrl("https://foo-lv5r9y--api-mydev.dev.stripe.me/"))

	assert.Error(t, ValidateAPIBaseUrl("https://example.com"))
	assert.Error(t, ValidateAPIBaseUrl("https://unknowndomain"))
	assert.Error(t, ValidateAPIBaseUrl("http://127.0.0.1:1337 "))
	assert.Error(t, ValidateAPIBaseUrl("localhost"))
	assert.Error(t, ValidateAPIBaseUrl("anything_else"))
}

func TestValidateDashboardBaseURLWorks(t *testing.T) {
	assert.Nil(t, ValidateDashboardBaseURL("https://dashboard.stripe.com"))
	assert.Nil(t, ValidateDashboardBaseURL("https://qa-dashboard.stripe.com"))
	assert.Nil(t, ValidateDashboardBaseURL("http://foo-manage-mydev.dev.stripe.me"))
	assert.Nil(t, ValidateDashboardBaseURL("https://foo-lv5r9y--manage-mydev.dev.stripe.me/"))

	assert.Error(t, ValidateDashboardBaseURL("https://example.com"))
	assert.Error(t, ValidateDashboardBaseURL("https://unknowndomain"))
	assert.Error(t, ValidateDashboardBaseURL("http://127.0.0.1:1337 "))
	assert.Error(t, ValidateDashboardBaseURL("localhost"))
	assert.Error(t, ValidateDashboardBaseURL("anything_else"))
}
