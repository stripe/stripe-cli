package requests

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDataForRequest(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry=human"}}
	expected := "bender=robot&fry=human"

	output, _ := rb.buildDataForRequest(params)
	assert.Equal(t, expected, output)
}

func TestBuildDataForRequestParamOrdering(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"fry=human", "bender=robot"}}
	expected := "fry=human&bender=robot"

	output, _ := rb.buildDataForRequest(params)
	assert.Equal(t, expected, output)
}

func TestBuildDataForRequestExpand(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{expand: []string{"futurama.employees", "futurama.ships"}}
	expected := "expand[]=futurama.employees&expand[]=futurama.ships"

	output, _ := rb.buildDataForRequest(params)
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

	expected := "limit=10&starting_after=bender&ending_before=leela"

	output, _ := rb.buildDataForRequest(params)
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

	output, _ := rb.buildDataForRequest(params)
	assert.Equal(t, expected, output)
}

func TestBuildDataForRequestInvalidArgument(t *testing.T) {
	rb := Base{}
	params := &RequestParameters{data: []string{"bender=robot", "fry"}}
	expected := "Invalid data argument: fry"

	data, err := rb.buildDataForRequest(params)
	assert.Equal(t, "", data)
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
		assert.Equal(t, "bender=robot&fry=human&expand[]=futurama.employees&expand[]=futurama.ships", r.URL.RawQuery)
		assert.Equal(t, "", string(reqBody))
	}))
	defer ts.Close()

	rb := Base{APIBaseURL: ts.URL}
	rb.Method = http.MethodGet

	params := &RequestParameters{
		data:   []string{"bender=robot", "fry=human"},
		expand: []string{"futurama.employees", "futurama.ships"},
	}

	_, err := rb.MakeRequest("sk_test_1234", "/foo/bar", params)
	assert.Nil(t, err)
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
