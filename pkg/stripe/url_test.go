package stripe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAPIBaseURLWorks(t *testing.T) {
	assert.Nil(t, ValidateAPIBaseURL("https://api.stripe.com"))
	assert.Nil(t, ValidateAPIBaseURL("https://api.stripe.com/v1"))
	assert.Nil(t, ValidateAPIBaseURL("https://api.stripe.com/v2"))
	assert.Nil(t, ValidateAPIBaseURL("https://api.stripe.com/v100"))
	assert.Nil(t, ValidateAPIBaseURL("https://qa-api.stripe.com"))
	assert.Nil(t, ValidateAPIBaseURL("http://foo-api-mydev.dev.stripe.me"))
	assert.Nil(t, ValidateAPIBaseURL("https://foo-lv5r9y--api-mydev.dev.stripe.me/"))
	assert.Nil(t, ValidateAPIBaseURL("https://foo-lv5r9y--api-iso.dev.stripe.me"))
	assert.Nil(t, ValidateAPIBaseURL("http://127.0.0.1"))
	assert.Nil(t, ValidateAPIBaseURL("http://127.0.0.1:1337"))
	assert.Nil(t, ValidateAPIBaseURL("https://files.stripe.com/"))

	assert.ErrorIs(t, ValidateAPIBaseURL("https://example.com"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("https://unknowndomain"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("localhost"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("anything_else"), errInvalidAPIBaseURL)
	assert.ErrorIs(t, ValidateAPIBaseURL("https://api.stripe.com/v1.1"), errInvalidAPIBaseURL)
}

func TestValidateDashboardBaseURLWorks(t *testing.T) {
	assert.Nil(t, ValidateDashboardBaseURL("https://dashboard.stripe.com"))
	assert.Nil(t, ValidateDashboardBaseURL("https://qa-dashboard.stripe.com"))
	assert.Nil(t, ValidateDashboardBaseURL("http://foo-manage-mydev.dev.stripe.me"))
	assert.Nil(t, ValidateDashboardBaseURL("https://foo-lv5r9y--manage-mydev.dev.stripe.me/"))
	assert.Nil(t, ValidateDashboardBaseURL("https://foo-0-lv5r9y--manage-dashboard-proxy-mydev.dev.stripe.me/"))
	assert.Nil(t, ValidateDashboardBaseURL("https://foo-0-lv5r9y--dashboard-dev.dev.stripe.me/"))
	assert.Nil(t, ValidateDashboardBaseURL("https://foo-0-lv5r9y--dashboard-iso.dev.stripe.me/"))
	assert.Nil(t, ValidateDashboardBaseURL("http://127.0.0.1"))
	assert.Nil(t, ValidateDashboardBaseURL("http://127.0.0.1:1337"))

	assert.ErrorIs(t, ValidateDashboardBaseURL("https://example.com"), errInvalidDashboardBaseURL)
	assert.ErrorIs(t, ValidateDashboardBaseURL("https://unknowndomain"), errInvalidDashboardBaseURL)
	assert.ErrorIs(t, ValidateDashboardBaseURL("localhost"), errInvalidDashboardBaseURL)
	assert.ErrorIs(t, ValidateDashboardBaseURL("anything_else"), errInvalidDashboardBaseURL)
}

func TestDashboardBaseURLForAPIBaseURLWorks(t *testing.T) {
	assert.Equal(t, "https://dashboard.stripe.com", DashboardBaseURLForAPIBaseURL(""))
	assert.Equal(t, "https://dashboard.stripe.com", DashboardBaseURLForAPIBaseURL("https://api.stripe.com"))
	assert.Equal(t, "https://dashboard.stripe.com", DashboardBaseURLForAPIBaseURL("https://api.stripe.com/v1"))
	assert.Equal(t, "https://qa-dashboard.stripe.com", DashboardBaseURLForAPIBaseURL("https://qa-api.stripe.com"))
	assert.Equal(t, "http://foo-manage-mydev.dev.stripe.me", DashboardBaseURLForAPIBaseURL("http://foo-api-mydev.dev.stripe.me"))
	assert.Equal(t, "https://foo-lv5r9y--manage-mydev.dev.stripe.me", DashboardBaseURLForAPIBaseURL("https://foo-lv5r9y--api-mydev.dev.stripe.me/"))
	assert.Equal(t, "https://foo-lv5r9y--dashboard-iso.dev.stripe.me", DashboardBaseURLForAPIBaseURL("https://foo-lv5r9y--api-iso.dev.stripe.me"))
	assert.Equal(t, "https://foo-0-lv5r9y--dashboard-dev.dev.stripe.me", DashboardBaseURLForAPIBaseURL("https://foo-0-lv5r9y--api-dev.dev.stripe.me"))
	assert.Equal(t, "https://foo-0-lv5r9y--dashboard-iso.dev.stripe.me", DashboardBaseURLForAPIBaseURL("https://foo-0-lv5r9y--api-iso.dev.stripe.me"))
	assert.Equal(t, "http://127.0.0.1:1337", DashboardBaseURLForAPIBaseURL("http://127.0.0.1:1337"))
}
