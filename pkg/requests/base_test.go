package requests

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDataForRequest(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry=human"}}
	expected := "bender=robot&fry=human"

	output, _ := rb.BuildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestParamOrdering(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"fry=human", "bender=robot"}}
	expected := "fry=human&bender=robot"

	output, _ := rb.BuildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestExpand(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{expand: []string{"futurama.employees", "futurama.ships"}}
	expected := "expand[]=futurama.employees&expand[]=futurama.ships"

	output, _ := rb.BuildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestPagination(t *testing.T) {
	rb := Base{}
	rb.Method = http.MethodGet

	params := &RequestParameters{
		limit:         "10",
		startingAfter: "bender",
		endingBefore:  "leela",
	}

	expected := "limit=10&starting_after=bender&ending_before=leela"

	output, _ := rb.BuildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestGetOnly(t *testing.T) {
	rb := Base{}
	rb.Method = http.MethodPost

	params := &RequestParameters{
		limit:         "10",
		startingAfter: "bender",
		endingBefore:  "leela",
	}

	expected := ""

	output, _ := rb.BuildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestInvalidArgument(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry"}}
	expected := "invalid data argument: fry"

	data, err := rb.BuildDataForRequest(params)
	require.Equal(t, "", data)
	require.Equal(t, expected, err.Error())
}

func TestMakeRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/foo/bar", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Equal(t, "bender=robot&fry=human&expand[]=futurama.employees&expand[]=futurama.ships", r.URL.RawQuery)
		require.Equal(t, "", string(reqBody))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodGet

	params := &RequestParameters{
		data:   []string{"bender=robot", "fry=human"},
		expand: []string{"futurama.employees", "futurama.ships"},
	}

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/foo/bar", params, make(map[string]interface{}), true, nil)
	require.NoError(t, err)
}

func TestMakeRequest_GetV2(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/core/events", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Equal(t, "limit=10&types=v2.core.event_destination.ping&types=v1.billing.meter.no_meter_found", r.URL.RawQuery)
		require.Equal(t, "", string(reqBody))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodGet

	params := &RequestParameters{
		data: []string{`{
			"limit": 10,
			"types": [
				"v2.core.event_destination.ping",
				"v1.billing.meter.no_meter_found"
			]
		}`},
	}

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v2/core/events", params, make(map[string]interface{}), true, nil)
	require.NoError(t, err)
}

func TestMakeRequest_PostV2(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/core/event_destinations", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Equal(t, "", r.URL.RawQuery)
		require.Equal(t, `{"name":"foo"}`, string(reqBody))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodPost

	params := &RequestParameters{
		data: []string{`{"name": "foo"}`},
	}

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v2/core/event_destinations", params, make(map[string]interface{}), true, nil)
	require.NoError(t, err)
}

func TestMakeRequest_ErrOnStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(":("))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodGet

	params := &RequestParameters{}

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/foo/bar", params, make(map[string]interface{}), true, nil)
	require.Error(t, err)
	require.Equal(t, "Request failed, status=500, body=:(", err.Error())
}

func TestMakeRequest_ErrOnAPIKeyExpired(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`
{
  "error": {
    "code": "api_key_expired",
    "doc_url": "https://stripe.com/docs/error-codes/api-key-expired",
    "message": "Expired API Key provided: rk_test_***123",
    "type": "invalid_request_error"
  }
}
		`))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodGet

	params := &RequestParameters{}

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/foo/bar", params, make(map[string]interface{}), false, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Request failed, status=401, body=")
}

func TestMakeMultiPartRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("FILES!"))

		reqBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/foo/bar", r.URL.Path)
		require.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		require.NotEmpty(t, r.UserAgent())
		require.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		require.Contains(t, string(reqBody), "purpose")
		require.Contains(t, string(reqBody), "app_upload")
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodPost

	tempFile, err := os.CreateTemp("", "upload.zip")
	if err != nil {
		t.Error("Error creating temp file")
	}
	defer os.Remove(tempFile.Name())

	params := &RequestParameters{
		data: []string{"purpose=app_upload", fmt.Sprintf("file=@%v", tempFile.Name())},
	}

	_, err = rb.MakeMultiPartRequest(context.Background(), "sk_test_1234", "/foo/bar", params, true)
	require.NoError(t, err)
}

func TestGetUserConfirmationRequired(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("yes\n"))

	rb := Base{}
	rb.Method = http.MethodDelete
	rb.autoConfirm = false

	confirmed, err := rb.getUserConfirmation(reader)
	require.True(t, confirmed)
	require.NoError(t, err)
}

func TestGetUserConfirmationNotRequired(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))

	rb := Base{}
	rb.Method = http.MethodGet
	rb.autoConfirm = false

	confirmed, err := rb.getUserConfirmation(reader)
	require.True(t, confirmed)
	require.NoError(t, err)
}

func TestGetUserConfirmationAutoConfirm(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))

	rb := Base{}
	rb.Method = http.MethodDelete
	rb.autoConfirm = true

	confirmed, err := rb.getUserConfirmation(reader)
	require.True(t, confirmed)
	require.NoError(t, err)
}

func TestGetUserConfirmationNoConfirm(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("blah\n"))

	rb := Base{}
	rb.Method = http.MethodDelete
	rb.autoConfirm = false

	confirmed, err := rb.getUserConfirmation(reader)
	require.False(t, confirmed)
	require.NoError(t, err)
}

func TestNormalizePath(t *testing.T) {
	require.Equal(t, "/v1/charges", normalizePath("/v1/charges"))
	require.Equal(t, "/v1/charges", normalizePath("v1/charges"))
	require.Equal(t, "/v1/charges", normalizePath("/charges"))
	require.Equal(t, "/v1/charges", normalizePath("charges"))

	require.Equal(t, "/v2/core/events", normalizePath("/v2/core/events"))
	require.Equal(t, "/v2/core/events", normalizePath("v2/core/events"))
}

func TestCreateOrNormalizePath(t *testing.T) {
	result, _ := createOrNormalizePath("ch_12345")
	require.Equal(t, "/v1/charges/ch_12345", result)

	result, _ = createOrNormalizePath("cs_test_12345")
	require.Equal(t, "/v1/checkout/sessions/cs_test_12345", result)

	result, _ = createOrNormalizePath("cs_live_12345")
	require.Equal(t, "/v1/checkout/sessions/cs_live_12345", result)

	result, _ = createOrNormalizePath("sub_sched_12345")
	require.Equal(t, "/v1/subscription_schedules/sub_sched_12345", result)

	result, _ = createOrNormalizePath("/v1/charges")
	require.Equal(t, "/v1/charges", result)

	result, _ = createOrNormalizePath("v1/charges")
	require.Equal(t, "/v1/charges", result)

	result, _ = createOrNormalizePath("/charges")
	require.Equal(t, "/v1/charges", result)

	result, _ = createOrNormalizePath("charges")
	require.Equal(t, "/v1/charges", result)
}

func TestIsAPIKeyExpiredError(t *testing.T) {
	for _, tt := range []struct {
		statusCode int
		errorCode  string
		want       bool
	}{
		{200, "", false},
		{401, "authentication_required", false},
		{500, "api_key_expired", false},
		{401, "api_key_expired", true},
	} {
		t.Run(fmt.Sprintf("status=%v,code=%q", tt.statusCode, tt.errorCode), func(t *testing.T) {
			err := RequestError{
				StatusCode: tt.statusCode,
				ErrorCode:  tt.errorCode,
			}
			require.Equal(t, tt.want, IsAPIKeyExpiredError(err))
		})
	}

	t.Run("non-RequestError", func(t *testing.T) {
		require.False(t, IsAPIKeyExpiredError(fmt.Errorf("other")))
	})
}

func TestComputeVersionHeader(t *testing.T) {
	t.Run("explicit version", func(t *testing.T) {
		rb := Base{}
		params := &RequestParameters{version: "2025-01-01"}
		require.Equal(t, "2025-01-01", rb.computeVersionHeader(params, "/v1/customers"))
	})
	t.Run("preview command", func(t *testing.T) {
		rb := Base{IsPreviewCommand: true}
		params := &RequestParameters{}
		require.Equal(t, StripePreviewVersionHeaderValue, rb.computeVersionHeader(params, "/v1/customers"))
	})
	t.Run("v2 path", func(t *testing.T) {
		rb := Base{}
		params := &RequestParameters{}
		require.Equal(t, StripeVersionHeaderValue, rb.computeVersionHeader(params, "/v2/billing/meters"))
	})
	t.Run("v1 path no version", func(t *testing.T) {
		rb := Base{}
		params := &RequestParameters{}
		require.Equal(t, "", rb.computeVersionHeader(params, "/v1/customers"))
	})
	t.Run("explicit version takes precedence over preview", func(t *testing.T) {
		rb := Base{IsPreviewCommand: true}
		params := &RequestParameters{version: "2025-01-01"}
		require.Equal(t, "2025-01-01", rb.computeVersionHeader(params, "/v1/customers"))
	})
}

func TestParseV1DataForDryRun(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result, err := parseV1DataForDryRun([]string{})
		require.NoError(t, err)
		require.Empty(t, result)
	})
	t.Run("simple key-value", func(t *testing.T) {
		result, err := parseV1DataForDryRun([]string{"email=test@example.com"})
		require.NoError(t, err)
		require.Equal(t, map[string]interface{}{"email": "test@example.com"}, result)
	})
	t.Run("value with equals sign", func(t *testing.T) {
		result, err := parseV1DataForDryRun([]string{"redirect=https://example.com?a=1&b=2"})
		require.NoError(t, err)
		require.Equal(t, "https://example.com?a=1&b=2", result["redirect"])
	})
	t.Run("nested bracket notation", func(t *testing.T) {
		result, err := parseV1DataForDryRun([]string{"metadata[env]=staging", "metadata[version]=2"})
		require.NoError(t, err)
		meta, ok := result["metadata"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "staging", meta["env"])
		require.Equal(t, "2", meta["version"])
	})
	t.Run("deep nesting", func(t *testing.T) {
		result, err := parseV1DataForDryRun([]string{"shipping[address][line1]=123 Main St"})
		require.NoError(t, err)
		shipping, ok := result["shipping"].(map[string]interface{})
		require.True(t, ok)
		address, ok := shipping["address"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "123 Main St", address["line1"])
	})
	t.Run("array notation", func(t *testing.T) {
		result, err := parseV1DataForDryRun([]string{"items[]=a", "items[]=b"})
		require.NoError(t, err)
		items, ok := result["items"].([]interface{})
		require.True(t, ok)
		require.Equal(t, []interface{}{"a", "b"}, items)
	})
	t.Run("invalid argument no equals", func(t *testing.T) {
		_, err := parseV1DataForDryRun([]string{"no-equals-sign"})
		require.Error(t, err)
	})
}

func TestBuildDryRunOutput_V1Post(t *testing.T) {
	rb := Base{Method: http.MethodPost}
	additional := map[string]interface{}{
		"email":       "test@example.com",
		"description": "Test Customer",
	}

	// "sk_test_1234567890abcdef" (24 chars) redacts to "sk_test_************cdef"
	output, err := rb.BuildDryRunOutput("sk_test_1234567890abcdef", "https://api.stripe.com", "/v1/customers", &RequestParameters{}, additional)
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "POST",
		URL:    "https://api.stripe.com/v1/customers",
		Params: map[string]interface{}{
			"email":       "test@example.com",
			"description": "Test Customer",
		},
		Headers: map[string]string{
			"Authorization": "Bearer sk_test_************cdef",
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		AuthAvailable:        true,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_V1PostDataParams(t *testing.T) {
	rb := Base{Method: http.MethodPost}
	params := &RequestParameters{
		data: []string{"metadata[env]=staging", "metadata[version]=2"},
	}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers", params, map[string]interface{}{"email": "test@example.com"})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "POST",
		URL:    "https://api.stripe.com/v1/customers",
		Params: map[string]interface{}{
			"email": "test@example.com",
			"metadata": map[string]interface{}{
				"env":     "staging",
				"version": "2",
			},
		},
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_V1Get(t *testing.T) {
	rb := Base{Method: http.MethodGet}
	params := &RequestParameters{
		limit:         "5",
		startingAfter: "cus_abc",
		endingBefore:  "cus_xyz",
		expand:        []string{"default_source"},
	}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers", params, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "GET",
		URL:    "https://api.stripe.com/v1/customers",
		Params: map[string]interface{}{
			"limit":          "5",
			"starting_after": "cus_abc",
			"ending_before":  "cus_xyz",
			"expand":         []interface{}{"default_source"},
		},
		Headers:              map[string]string{},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_V1PostExpand(t *testing.T) {
	rb := Base{Method: http.MethodPost}
	params := &RequestParameters{
		expand: []string{"default_source", "invoice_settings"},
	}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers", params, map[string]interface{}{"email": "test@example.com"})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "POST",
		URL:    "https://api.stripe.com/v1/customers",
		Params: map[string]interface{}{
			"email":  "test@example.com",
			"expand": []interface{}{"default_source", "invoice_settings"},
		},
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_V2Post(t *testing.T) {
	rb := Base{Method: http.MethodPost}
	params := &RequestParameters{
		data: []string{`{"event_name": "foo", "value": 100}`},
	}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v2/billing/meter_events", params, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "POST",
		URL:    "https://api.stripe.com/v2/billing/meter_events",
		Params: map[string]interface{}{
			"event_name": "foo",
			"value":      float64(100),
		},
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"Stripe-Version": StripeVersionHeaderValue,
		},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_Delete(t *testing.T) {
	rb := Base{Method: http.MethodDelete}

	output, err := rb.BuildDryRunOutput("sk_test_abcd", "https://api.stripe.com", "/v1/customers/cus_abc123", &RequestParameters{}, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "DELETE",
		URL:    "https://api.stripe.com/v1/customers/cus_abc123",
		Params: map[string]interface{}{},
		Headers: map[string]string{
			"Authorization": "Bearer sk_test_abcd",
			"Content-Type":  "application/x-www-form-urlencoded",
		},
		AuthAvailable:        true,
		RequiresConfirmation: true,
	}}, *output)
}

func TestBuildDryRunOutput_NoAPIKey(t *testing.T) {
	rb := Base{Method: http.MethodPost}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers", &RequestParameters{}, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method:               "POST",
		URL:                  "https://api.stripe.com/v1/customers",
		Params:               map[string]interface{}{},
		Headers:              map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_ExplicitStripeVersion(t *testing.T) {
	rb := Base{Method: http.MethodPost}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers", &RequestParameters{version: "2025-01-01"}, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method:               "POST",
		URL:                  "https://api.stripe.com/v1/customers",
		Params:               map[string]interface{}{},
		Headers:              map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Stripe-Version": "2025-01-01"},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_OptionalHeaders(t *testing.T) {
	rb := Base{Method: http.MethodPost}
	params := &RequestParameters{
		idempotency:   "idem-key-123",
		stripeAccount: "acct_123",
		stripeContext: "ctx_456",
	}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers", params, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method: "POST",
		URL:    "https://api.stripe.com/v1/customers",
		Params: map[string]interface{}{},
		Headers: map[string]string{
			"Content-Type":    "application/x-www-form-urlencoded",
			"Idempotency-Key": "idem-key-123",
			"Stripe-Account":  "acct_123",
			"Stripe-Context":  "ctx_456",
		},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func TestBuildDryRunOutput_PathParamSubstitutedURL(t *testing.T) {
	rb := Base{Method: http.MethodGet}

	output, err := rb.BuildDryRunOutput("", "https://api.stripe.com", "/v1/customers/cus_abc123", &RequestParameters{}, map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, DryRunOutput{DryRun: DryRunDetails{
		Method:               "GET",
		URL:                  "https://api.stripe.com/v1/customers/cus_abc123",
		Params:               map[string]interface{}{},
		Headers:              map[string]string{},
		AuthAvailable:        false,
		RequiresConfirmation: false,
	}}, *output)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	origStderr := os.Stderr
	defer func() { os.Stderr = origStderr }()

	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = stderrWriter

	fn()

	stderrWriter.Close()
	out, err := io.ReadAll(stderrReader)
	require.NoError(t, err)
	return string(out)
}

func TestMakeRequest_VersionUpgradeNotice(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Stripe-Api-Version-Upgrade-Notice", "Please upgrade to the latest API version.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	out := captureStderr(t, func() {
		rb := Base{APIBaseURL: ts.URL}
		rb.Method = http.MethodGet
		params := &RequestParameters{}
		_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v1/charges", params, make(map[string]interface{}), false, nil)
		require.NoError(t, err)
	})

	require.Contains(t, out, "API version upgrade notice: Please upgrade to the latest API version.")
}

func TestMakeRequest_IntegrationPathUpgradeNotice(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Stripe-Api-Integration-Path-Upgrade-Notice", "Please migrate to the new integration path.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	out := captureStderr(t, func() {
		rb := Base{APIBaseURL: ts.URL}
		rb.Method = http.MethodGet
		params := &RequestParameters{}
		_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v1/charges", params, make(map[string]interface{}), false, nil)
		require.NoError(t, err)
	})

	require.Contains(t, out, "API integration path upgrade notice: Please migrate to the new integration path.")
}

func TestMakeRequest_BothUpgradeNotices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Stripe-Api-Version-Upgrade-Notice", "Upgrade your API version.")
		w.Header().Set("Stripe-Api-Integration-Path-Upgrade-Notice", "Migrate your integration path.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	out := captureStderr(t, func() {
		rb := Base{APIBaseURL: ts.URL}
		rb.Method = http.MethodGet
		params := &RequestParameters{}
		_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v1/charges", params, make(map[string]interface{}), false, nil)
		require.NoError(t, err)
	})

	require.Contains(t, out, "API version upgrade notice: Upgrade your API version.")
	require.Contains(t, out, "API integration path upgrade notice: Migrate your integration path.")
}

func TestMakeRequest_NoUpgradeNotice(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	out := captureStderr(t, func() {
		rb := Base{APIBaseURL: ts.URL}
		rb.Method = http.MethodGet
		params := &RequestParameters{}
		_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v1/charges", params, make(map[string]interface{}), false, nil)
		require.NoError(t, err)
	})

	require.NotContains(t, out, "upgrade notice:")
}

func TestMakeRequest_UpgradeNoticeSuppressed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Stripe-Api-Version-Upgrade-Notice", "Please upgrade.")
		w.Header().Set("Stripe-Api-Integration-Path-Upgrade-Notice", "Please migrate.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	out := captureStderr(t, func() {
		rb := Base{APIBaseURL: ts.URL, SuppressOutput: true}
		rb.Method = http.MethodGet
		params := &RequestParameters{}
		_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/v1/charges", params, make(map[string]interface{}), false, nil)
		require.NoError(t, err)
	})

	require.NotContains(t, out, "upgrade notice:")
}

func TestParseJSONDataFlag(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		data, err := parseJSONDataFlag([]string{})
		require.Nil(t, err)
		require.Empty(t, data)
	})
	t.Run("empty data", func(t *testing.T) {
		_, err := parseJSONDataFlag([]string{""})
		require.ErrorIs(t, errJSONDataFlagInvalid, err)

		_, err = parseJSONDataFlag([]string{"  "})
		require.ErrorIs(t, errJSONDataFlagInvalid, err)
	})
	t.Run("multiple data arguments", func(t *testing.T) {
		_, err := parseJSONDataFlag([]string{`{}`, `{}`})
		require.ErrorIs(t, errJSONDataFlagInvalid, err)
	})
	t.Run("key-value data", func(t *testing.T) {
		_, err := parseJSONDataFlag([]string{"x=y"})
		require.ErrorIs(t, errJSONDataFlagInvalid, err)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		_, err := parseJSONDataFlag([]string{`{"key": }`})
		require.Error(t, err)
	})
	t.Run("valid JSON", func(t *testing.T) {
		data, err := parseJSONDataFlag([]string{`{"key": "x=y"}`})
		require.Nil(t, err)
		require.Equal(t, map[string]interface{}{"key": "x=y"}, data)
	})
}
