// Serialize, deserialize and store HTTP responses and requests.

package playback

import (
	"io/ioutil"
	"net/http"
	"net/url"
)

type serializer interface {
	serializeReq(httpRequest) (interface{}, error)
	serializeResp(httpResponse) (interface{}, error)
	EncodeCassette(Cassette) ([]byte, error)
	DecodeCassette([]byte) (Cassette, error)
}

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

func newHTTPResponse(resp *http.Response) (wrappedResponse httpResponse, err error) {
	wrappedResponse = httpResponse{}

	wrappedResponse.Headers = resp.Header
	wrappedResponse.StatusCode = resp.StatusCode

	var bodyBytes []byte
	bodyBytes, err = ioutil.ReadAll(resp.Body)
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

	var bodyBytes []byte
	bodyBytes, err = ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		return wrappedRequest, err
	}
	wrappedRequest.Body = bodyBytes

	return wrappedRequest, nil
}
