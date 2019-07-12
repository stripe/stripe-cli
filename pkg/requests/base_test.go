package requests

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func convertToString(data url.Values) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(strings.NewReader(data.Encode()))
	return buf.String()
}

func TestBuildDataForRequest(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry=human"}}
	expected := "bender=robot&fry=human"

	data, _ := rb.buildDataForRequest(params)
	output := convertToString(data)
	assert.Equal(t, expected, output)
}

func TestBuildDataForRequestExpand(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"expand=futurama.employees", "expand=futurama.ships"}}
	expected := "expand=futurama.employees&expand=futurama.ships"

	data, _ := rb.buildDataForRequest(params)
	output := convertToString(data)
	assert.Equal(t, expected, output)
}

func TestBuildDataForRequestPagination(t *testing.T) {
	rb := Base{}
	rb.Method = http.MethodGet

	params := &RequestParameters{
		limit:         "10",
		startingAfter: "bender",
		endingBefore:  "leela",
	}

	expected := "ending_before=leela&limit=10&starting_after=bender"

	data, _ := rb.buildDataForRequest(params)
	output := convertToString(data)
	assert.Equal(t, expected, output)
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

	data, _ := rb.buildDataForRequest(params)
	output := convertToString(data)
	assert.Equal(t, expected, output)
}

func TestBuildDataForRequestInvalidArgument(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry"}}
	expected := "Invalid data argument: fry"

	data, err := rb.buildDataForRequest(params)
	assert.Nil(t, data)
	assert.Equal(t, expected, err.Error())
}

func TestMakeRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))

		reqBody, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)

		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/foo/bar", r.URL.Path)
		assert.Equal(t, "Bearer sk_test_1234", r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.UserAgent())
		assert.NotEmpty(t, r.Header.Get("X-Stripe-Client-User-Agent"))
		assert.Equal(t, "bender=robot&expand=expand%3Dfuturama.employees&expand=expand%3Dfuturama.ships&fry=human", r.URL.RawQuery)
		assert.Equal(t, "", string(reqBody))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodGet

	params := &RequestParameters{
		data:   []string{"bender=robot", "fry=human"},
		expand: []string{"expand=futurama.employees", "expand=futurama.ships"},
	}

	_, err := rb.MakeRequest("sk_test_1234", "/foo/bar", params)
	assert.Nil(t, err)
}

func TestFormatHeaders(t *testing.T) {
	rb := Base{}

	resp := &http.Response{
		Header: make(http.Header, 0),
	}

	emptyHeaders := rb.formatHeaders(resp)
	emptyExpected := "\n"

	assert.Equal(t, emptyExpected, emptyHeaders)

	resp.Header.Add("header-one", "header-one-value")
	resp.Header.Add("header-two", "header-two-value")
	resp.Header.Add("header-three", "header-three-value")

	headers := rb.formatHeaders(resp)

	// Since Headers are stored in a map, the order is not deterministic
	// and we must check for contains each header vs. direct string comparison
	assert.Contains(t, headers, "< Header-One: header-one-value\n")
	assert.Contains(t, headers, "< Header-Two: header-two-value\n")
	assert.Contains(t, headers, "< Header-Three: header-three-value\n")
}

func TestGetUserConfirmationRequired(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("yes\n"))

	rb := Base{}
	rb.Method = http.MethodDelete
	rb.autoConfirm = false

	confirmed, err := rb.getUserConfirmation(reader)
	assert.True(t, confirmed)
	assert.Nil(t, err)
}

func TestGetUserConfirmationNotRequired(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))

	rb := Base{}
	rb.Method = http.MethodGet
	rb.autoConfirm = false

	confirmed, err := rb.getUserConfirmation(reader)
	assert.True(t, confirmed)
	assert.Nil(t, err)
}

func TestGetUserConfirmationAutoConfirm(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))

	rb := Base{}
	rb.Method = http.MethodDelete
	rb.autoConfirm = true

	confirmed, err := rb.getUserConfirmation(reader)
	assert.True(t, confirmed)
	assert.Nil(t, err)
}

func TestGetUserConfirmationNoConfirm(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("blah\n"))

	rb := Base{}
	rb.Method = http.MethodDelete
	rb.autoConfirm = false

	confirmed, err := rb.getUserConfirmation(reader)
	assert.False(t, confirmed)
	assert.Nil(t, err)
}

func TestNormalizePath(t *testing.T) {
	assert.Equal(t, "/v1/charges", normalizePath("/v1/charges"))
	assert.Equal(t, "/v1/charges", normalizePath("v1/charges"))
	assert.Equal(t, "/v1/charges", normalizePath("/charges"))
	assert.Equal(t, "/v1/charges", normalizePath("charges"))
}
