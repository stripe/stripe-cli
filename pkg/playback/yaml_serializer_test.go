package playback

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
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
	yamlReq, err := serializer.serializeReq(req)
	if err != nil {
		t.Fatal(err)
	}
	bytes, _ := yaml.Marshal(yamlReq)
	assert.Equal(t, expected, string(bytes))
}

func TestSerializeResp(t *testing.T) {
	serializer := YAMLSerializer{}

	expected := `headers: {}
body: response body
status_code: 200
`

	resp := response()
	yamlResp, err := serializer.serializeResp(resp)
	if err != nil {
		t.Fatal(err)
	}
	bytes, _ := yaml.Marshal(yamlResp)
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

func TestEncodeCassette(t *testing.T) {
	serializer := YAMLSerializer{}

	interaction1 := interaction{Type: 1, Request: request(), Response: response()}
	interaction2 := interaction{Type: 0, Request: request(), Response: response()}
	cassette := Cassette{interaction1, interaction2}

	encoded, err := serializer.EncodeCassette(cassette)
	if err != nil {
		t.Fatal(err)
	}

	expected := "- type: 1\n  request:\n    method: POST\n    body: hello world\n    headers: {}\n    url:\n      scheme: \"\"\n      opaque: \"\"\n      user: null\n      host: \"\"\n      path: \"\"\n      rawpath: \"\"\n      forcequery: false\n      rawquery: \"\"\n      fragment: \"\"\n      rawfragment: \"\"\n  response:\n    headers: {}\n    body: response body\n    status_code: 200\n- type: 0\n  request:\n    method: POST\n    body: hello world\n    headers: {}\n    url:\n      scheme: \"\"\n      opaque: \"\"\n      user: null\n      host: \"\"\n      path: \"\"\n      rawpath: \"\"\n      forcequery: false\n      rawquery: \"\"\n      fragment: \"\"\n      rawfragment: \"\"\n  response:\n    headers: {}\n    body: response body\n    status_code: 200\n"

	assert.Equal(t, expected, string(encoded))
}

func TestDecodeCassette(t *testing.T) {
	serializer := YAMLSerializer{}

	cassetteData := []byte("- type: 1\n  request:\n    method: POST\n    body: hello world\n    headers: {}\n    url:\n      scheme: \"\"\n      opaque: \"\"\n      user: null\n      host: \"\"\n      path: \"\"\n      rawpath: \"\"\n      forcequery: false\n      rawquery: \"\"\n      fragment: \"\"\n      rawfragment: \"\"\n  response:\n    headers: {}\n    body: response body\n    status_code: 200\n- type: 0\n  request:\n    method: POST\n    body: hello world\n    headers: {}\n    url:\n      scheme: \"\"\n      opaque: \"\"\n      user: null\n      host: \"\"\n      path: \"\"\n      rawpath: \"\"\n      forcequery: false\n      rawquery: \"\"\n      fragment: \"\"\n      rawfragment: \"\"\n  response:\n    headers: {}\n    body: response body\n    status_code: 200\n")

	cassette, err := serializer.DecodeCassette(cassetteData)
	if err != nil {
		t.Fatal(err)
	}

	firstInter := cassette[0]
	assert.Equal(t, incomingInteraction, firstInter.Type)
	assert.Equal(t, "POST", firstInter.Request.(httpRequest).Method)
	assert.Equal(t, []byte("hello world"), firstInter.Request.(httpRequest).Body)

	secondInter := cassette[1]
	assert.Equal(t, outgoingInteraction, secondInter.Type)
	assert.Equal(t, "POST", secondInter.Request.(httpRequest).Method)
	assert.Equal(t, []byte("hello world"), secondInter.Request.(httpRequest).Body)
}
