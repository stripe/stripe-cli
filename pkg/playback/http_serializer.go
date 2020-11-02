// Serialize, deserialize and store HTTP responses and requests.

package playback

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type serializer func(input interface{}) (bytes []byte, err error)
type deserializer func(input *io.Reader) (value interface{}, err error)

func httpRequestToBytes(input interface{}) (data []byte, err error) {
	return json.Marshal(input)
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

func httpResponsefromBytes(input *io.Reader) (val interface{}, err error) {
	output := httpResponse{}

	inputBytes, err := ioutil.ReadAll(*input)
	if err != nil {
		return output, err
	}

	err = json.Unmarshal(inputBytes, &output)
	return output, err
}

type httpResponse struct {
	Headers    http.Header
	Body       map[string]interface{}
	StatusCode int
}

func newHTTPResponse(resp *http.Response) (wrappedResponse httpResponse, err error) {
	wrappedResponse = httpResponse{}

	wrappedResponse.Headers = resp.Header
	wrappedResponse.StatusCode = resp.StatusCode

	// var bodyBytes []byte
	// bodyBytes, err = ioutil.ReadAll(resp.Body)
	// defer resp.Body.Close()
	// if err != nil {
	// 	return wrappedResponse, err
	// }
	// json.Unmarshal(bodyBytes, &wrappedResponse.Body)
	json.NewDecoder(resp.Body).Decode(&wrappedResponse.Body)

	return wrappedResponse, nil
}

type httpRequest struct {
	Method  string
	Body    map[string]interface{}
	Headers http.Header
	URL     url.URL
}

func newHTTPRequest(req *http.Request) (wrappedRequest httpRequest, err error) {
	wrappedRequest = httpRequest{}

	wrappedRequest.Method = req.Method
	wrappedRequest.Headers = req.Header
	wrappedRequest.URL = *req.URL

	// var bodyBytes []byte
	// bodyBytes, err = ioutil.ReadAll(req.Body)
	// defer req.Body.Close()
	// if err != nil {
	// 	return wrappedRequest, err
	// }
	// wrappedRequest.Body = bodyBytes
	json.NewDecoder(req.Body).Decode(&wrappedRequest.Body)
	fmt.Println(wrappedRequest.Body["type"])

	return wrappedRequest, nil
}
