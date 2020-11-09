package playback

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTTPRequestReturnsWrappedRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", bytes.NewBuffer([]byte{}))
	wrappedRequest, err := newHTTPRequest(req)
	check(t, err)
	assert.Equal(t, wrappedRequest.Method, req.Method)

	bodyBytes, _ := json.Marshal("some json body")
	post, _ := http.NewRequest("POST", "example.com", bytes.NewBuffer(bodyBytes))
	wrappedPost, err := newHTTPRequest(post)
	check(t, err)
	assert.Equal(t, &wrappedPost.URL, post.URL)
}

func TestNewHTTPResponseReturnsWrappedResponse(t *testing.T) {
	body := ioutil.NopCloser(bytes.NewBuffer([]byte("Hello World")))
	res := http.Response{
		Header:     http.Header{},
		Body:       body,
		StatusCode: 200,
	}
	wrappedResponse, err := newHTTPResponse(&res)
	check(t, err)

	assert.Equal(t, wrappedResponse.StatusCode, res.StatusCode)
	assert.Equal(t, string(wrappedResponse.Body), "Hello World")
}
