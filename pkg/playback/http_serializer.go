// Serialize, deserialize and store HTTP responses and requests.

package playback

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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

type httpResponse struct {
	Headers    http.Header
	Body       []byte
	StatusCode int
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
