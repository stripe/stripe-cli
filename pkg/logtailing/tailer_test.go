package logtailing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonifyFiltersAll(t *testing.T) {
	filters := &LogFilters{
		FilterAccount:        []string{"my-account"},
		FilterIPAddress:      []string{"my-ip-address"},
		FilterHTTPMethod:     []string{"my-http-method"},
		FilterRequestPath:    []string{"my-request-path"},
		FilterRequestStatus:  []string{"my-request-status"},
		FilterSource:         []string{"my-source"},
		FilterStatusCode:     []string{"my-status-code"},
		FilterStatusCodeType: []string{"my-status-code-type"},
	}
	expected := `{"filter_account":["my-account"],"filter_ip_address":["my-ip-address"],"filter_http_method":["my-http-method"],"filter_request_path":["my-request-path"],"filter_request_status":["my-request-status"],"filter_source":["my-source"],"filter_status_code":["my-status-code"],"filter_status_code_type":["my-status-code-type"]}`
	filtersStr, err := jsonifyFilters(filters)
	require.NoError(t, err)
	require.Equal(t, expected, filtersStr)
}

func TestJsonifyFiltersSome(t *testing.T) {
	filters := &LogFilters{
		FilterHTTPMethod: []string{"my-http-method"},
		FilterStatusCode: []string{"my-status-code"},
	}
	expected := `{"filter_http_method":["my-http-method"],"filter_status_code":["my-status-code"]}`
	filtersStr, err := jsonifyFilters(filters)
	require.NoError(t, err)
	require.Equal(t, expected, filtersStr)
}

func TestJsonifyFiltersEmpty(t *testing.T) {
	filters := &LogFilters{
		FilterAccount:        []string{},
		FilterIPAddress:      []string{},
		FilterHTTPMethod:     []string{},
		FilterRequestPath:    []string{},
		FilterRequestStatus:  []string{},
		FilterSource:         []string{},
		FilterStatusCode:     []string{},
		FilterStatusCodeType: []string{},
	}
	filtersStr, err := jsonifyFilters(filters)
	require.NoError(t, err)
	require.Equal(t, "{}", filtersStr)
}
