package stripe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAPIBaseURLWorks(t *testing.T) {
	assert.Nil(t, ValidateAPIBaseURL("https://api.stripe.com"))
	assert.Nil(t, ValidateAPIBaseURL("https://qa-api.stripe.com"))
	assert.Nil(t, ValidateAPIBaseURL("http://foo-api-mydev.dev.stripe.me"))
	assert.Nil(t, ValidateAPIBaseURL("https://foo-lv5r9y--api-mydev.dev.stripe.me/"))
	assert.Nil(t, ValidateAPIBaseURL("http://127.0.0.1"))
	assert.Nil(t, ValidateAPIBaseURL("http://127.0.0.1:1337"))

	assert.ErrorIs(t, ValidateAPIBaseURL("https://example.com"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("https://unknowndomain"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("localhost"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("anything_else"), errInvalidAPIBaseURL)
}

func TestValidateDashboardBaseURLWorks(t *testing.T) {
	assert.Nil(t, ValidateDashboardBaseURL("https://dashboard.stripe.com"))
	assert.Nil(t, ValidateDashboardBaseURL("https://qa-dashboard.stripe.com"))
	assert.Nil(t, ValidateDashboardBaseURL("http://foo-manage-mydev.dev.stripe.me"))
	assert.Nil(t, ValidateDashboardBaseURL("https://foo-lv5r9y--manage-mydev.dev.stripe.me/"))
	assert.Nil(t, ValidateDashboardBaseURL("http://127.0.0.1"))
	assert.Nil(t, ValidateDashboardBaseURL("http://127.0.0.1:1337"))

	assert.ErrorIs(t, ValidateDashboardBaseURL("https://example.com"), errInvalidDashboardBaseURL)
	assert.ErrorIs(t, ValidateDashboardBaseURL("https://unknowndomain"), errInvalidDashboardBaseURL)
	assert.ErrorIs(t, ValidateDashboardBaseURL("localhost"), errInvalidDashboardBaseURL)
	assert.ErrorIs(t, ValidateDashboardBaseURL("anything_else"), errInvalidDashboardBaseURL)
}
