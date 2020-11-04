// Serialize, deserialize and store HTTP responses and requests.

package playback

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"gopkg.in/yaml.v2"
)

type serializer interface {
	serializeReq(interface{}) ([]byte, error)
	serializeResp(interface{}) ([]byte, error)
	newInteraction(interactionType, httpRequest, httpResponse) interaction
	encodeCassetteToBytes(cassette) ([]byte, error)
}

// type serializer func(input interface{}) (bytes []byte, err error)
type deserializer func(input *io.Reader) (value interface{}, err error)

type httpRequest struct {
	Method  string
	Body    []byte
	Headers http.Header
	URL     url.URL
}

type YAMLRequest struct {
	Method  string      `yaml:"method"`
	Body    string      `yaml:"body"`
	Headers http.Header `yaml:"headers"`
	URL     url.URL     `yaml:"url"`
}

type httpResponse struct {
	Headers    http.Header
	Body       []byte
	StatusCode int
}

type YAMLResponse struct {
	Headers    http.Header `yaml:"headers"`
	Body       string      `yaml:"body"`
	StatusCode int         `yaml: "status"`
}

func httpRequestToBytes(input interface{}) (data []byte, err error) {
	return json.Marshal(input)
}

func httpRequestToYAML(input interface{}) ([]byte, error) {
	req := input.(httpRequest)
	yml := YAMLRequest{
		req.Method,
		string(req.Body),
		req.Headers,
		req.URL,
	}
	return yaml.Marshal(yml)
}

func httpRequestfromBytes(input *io.Reader) (val interface{}, err error) {
	output := httpRequest{}

	inputBytes, err := ioutil.ReadAll(*input)
	if err != nil {
		return output, err
	}

	err = json.Unmarshal(inputBytes, &output)
	return output, err
}

func httpResponseToBytes(input interface{}) (data []byte, err error) {
	return json.Marshal(input)
}

func httpResponseToYAML(input interface{}) ([]byte, error) {
	res := input.(httpResponse)
	yml := YAMLResponse{
		res.Headers,
		string(res.Body),
		res.StatusCode,
	}
	return yaml.Marshal(yml)
}

func httpResponsefromBytes(input *io.Reader) (val interface{}, err error) {
	output := httpResponse{}

	inputBytes, err := ioutil.ReadAll(*input)
	if err != nil {
		return output, err
	}

	err = json.Unmarshal(inputBytes, &output)
	return output, err
}

func newHTTPResponse(resp *http.Response) (wrappedResponse httpResponse, err error) {
	wrappedResponse = httpResponse{}

	wrappedResponse.Headers = resp.Header
	wrappedResponse.StatusCode = resp.StatusCode

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return wrappedResponse, err
	}
	wrappedResponse.Body = bodyBytes

	return wrappedResponse, nil
}

func newHTTPRequest(req *http.Request) (wrappedRequest httpRequest, err error) {
	wrappedRequest = httpRequest{}

	wrappedRequest.Method = req.Method
	wrappedRequest.Headers = req.Header
	wrappedRequest.URL = *req.URL

	bodyBytes, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		return wrappedRequest, err
	}
	wrappedRequest.Body = bodyBytes

	return wrappedRequest, nil
}
