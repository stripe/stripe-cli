package playback

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func request() httpRequest {
	return httpRequest{
		Method:  "POST",
		Body:    []byte("hello world"),
		Headers: http.Header{},
		URL:     url.URL{},
	}
}

func response() httpResponse {
	return httpResponse{
		Headers:    http.Header{},
		StatusCode: 200,
		Body:       []byte("response body"),
	}
}

func TestSerializeReq(t *testing.T) {
	serializer := YAMLSerializer{}

	expected := `method: POST
body: hello world
headers: {}
url:
  scheme: ""
  opaque: ""
  user: null
  host: ""
  path: ""
  rawpath: ""
  forcequery: false
  rawquery: ""
  fragment: ""
  rawfragment: ""
`

	req := request()
	bytes, err := serializer.serializeReq(req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(bytes))
}

func TestSerializeResp(t *testing.T) {
	serializer := YAMLSerializer{}

	expected := `headers: {}
body: response body
status_code: 200
`

	resp := response()
	bytes, err := serializer.serializeResp(resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(bytes))
}

func TestDeserializeReq(t *testing.T) {
	serializer := YAMLSerializer{}

	data := []byte(`method: POST
body: hello world
headers: {}
url:
  scheme: ""
  opaque: ""
  user: null
  host: ""
  path: ""
  rawpath: ""
  forcequery: false
  rawquery: ""
  fragment: ""
  rawfragment: ""
`)
	req, err := serializer.deserializeReq(data)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, request(), req)
}

func TestDeserializeResp(t *testing.T) {
	serializer := YAMLSerializer{}

	data := []byte(`headers: {}
body: response body
status_code: 200
`)

	resp, err := serializer.deserializeResp(data)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, response(), resp)
}

func TestNewInteraction(t *testing.T) {
	serializer := YAMLSerializer{}

	interaction := serializer.newInteraction(1, request(), response())
	req := interaction.Request.(YAMLRequest)
	resp := interaction.Response.(YAMLResponse)
	assert.Equal(t, incomingInteraction, interaction.Type)
	assert.Equal(t, req.Body, "hello world")
	assert.Equal(t, resp.StatusCode, 200)
}

func TestEncodeCassette(t *testing.T) {
	serializer := YAMLSerializer{}

	interaction1 := serializer.newInteraction(1, request(), response())
	interaction2 := serializer.newInteraction(0, request(), response())
	cassette := cassette{interaction1, interaction2}

	encoded, err := serializer.encodeCassette(cassette)
	if err != nil {
		t.Fatal(err)
	}

	expected := "- type: 1\n  request:\n    method: POST\n    body: hello world\n    headers: {}\n    url:\n      scheme: \"\"\n      opaque: \"\"\n      user: null\n      host: \"\"\n      path: \"\"\n      rawpath: \"\"\n      forcequery: false\n      rawquery: \"\"\n      fragment: \"\"\n      rawfragment: \"\"\n  response:\n    headers: {}\n    body: response body\n    status_code: 200\n- type: 0\n  request:\n    method: POST\n    body: hello world\n    headers: {}\n    url:\n      scheme: \"\"\n      opaque: \"\"\n      user: null\n      host: \"\"\n      path: \"\"\n      rawpath: \"\"\n      forcequery: false\n      rawquery: \"\"\n      fragment: \"\"\n      rawfragment: \"\"\n  response:\n    headers: {}\n    body: response body\n    status_code: 200\n"

	assert.Equal(t, expected, string(encoded))
}
