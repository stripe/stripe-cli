package requests

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
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

	output, _ := rb.buildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestParamOrdering(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"fry=human", "bender=robot"}}
	expected := "fry=human&bender=robot"

	output, _ := rb.buildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestExpand(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{expand: []string{"futurama.employees", "futurama.ships"}}
	expected := "expand[]=futurama.employees&expand[]=futurama.ships"

	output, _ := rb.buildDataForRequest(params)
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

	output, _ := rb.buildDataForRequest(params)
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

	output, _ := rb.buildDataForRequest(params)
	require.Equal(t, expected, output)
}

func TestBuildDataForRequestInvalidArgument(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry"}}
	expected := "Invalid data argument: fry"

	data, err := rb.buildDataForRequest(params)
	require.Equal(t, "", data)
	require.Equal(t, expected, err.Error())
}

func TestMakeRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := ioutil.ReadAll(r.Body)
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

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/foo/bar", params, true)
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

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/foo/bar", params, true)
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

	_, err := rb.MakeRequest(context.Background(), "sk_test_1234", "/foo/bar", params, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Request failed, status=401, body=")
}

func TestMakeMultiPartRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("FILES!"))

		reqBody, err := ioutil.ReadAll(r.Body)
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
