package logtailing

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonifyFiltersAll(t *testing.T) {
	filters := &LogFilters{
		FilterIPAddress:      "my-ip-address",
		FilterHTTPMethod:     "my-http-method",
		FilterRequestPath:    "my-request-path",
		FilterSource:         "my-source",
		FilterStatusCode:     "my-status-code",
		FilterStatusCodeType: "my-status-code-type",
	}
	expected := fmt.Sprintf(`{"filter_ip_address":"my-ip-address","filter_http_method":"my-http-method","filter_request_path":"my-request-path","filter_source":"my-source","filter_status_code":"my-status-code","filter_status_code_type":"my-status-code-type"}`)
	filtersStr, err := jsonifyFilters(filters)
	assert.Nil(t, err)
	assert.Equal(t, expected, filtersStr)
}

func TestJsonifyFiltersSome(t *testing.T) {
	filters := &LogFilters{
		FilterHTTPMethod: "my-http-method",
		FilterStatusCode: "my-status-code",
	}
	expected := fmt.Sprintf(`{"filter_http_method":"my-http-method","filter_status_code":"my-status-code"}`)
	filtersStr, err := jsonifyFilters(filters)
	assert.Nil(t, err)
	assert.Equal(t, expected, filtersStr)
}

func TestJsonifyFiltersEmpty(t *testing.T) {
	filters := &LogFilters{}
	filtersStr, err := jsonifyFilters(filters)
	assert.Nil(t, err)
	assert.Equal(t, "{}", filtersStr)
}
