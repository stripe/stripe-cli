package playback

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

// type Serializer interface {

// 	func write()

// }

type httpRequestSerializable struct {
	baseRequest *http.Request
}

func newSerializableHTTPRequest(r *http.Request) httpRequestSerializable {
	return httpRequestSerializable{r}
}

func (reqWrapper httpRequestSerializable) toBytes() (data []byte, err error) {
	var buffer = &bytes.Buffer{}
	err = reqWrapper.baseRequest.Write(buffer)
	return buffer.Bytes(), err
}

func (reqWrapper httpRequestSerializable) fromBytes(input *io.Reader) (val interface{}, err error) {
	r := bufio.NewReader(*input)
	req, err := http.ReadRequest(r)
	return req, err
}

type httpResponseSerializable struct {
	baseResponse *http.Response
}

func newSerializableHTTPResponse(r *http.Response) httpResponseSerializable {
	return httpResponseSerializable{r}
}

func (respWrapper httpResponseSerializable) toBytes() (data []byte, err error) {
	var buffer = &bytes.Buffer{}

	err = respWrapper.baseResponse.Write(buffer)
	return buffer.Bytes(), err
}

func (respWrapper httpResponseSerializable) fromBytes(input *io.Reader) (val interface{}, err error) {
	// Read data from input and parse into a http.Response
	r := bufio.NewReader(*input)
	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		return resp, err
	}

	// We need to close the body in this scope, so first, read the bytes and
	// reset the resp.Body so that it can be read again by the caller.
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	err = resp.Body.Close()

	// This block below is necessary so that calling resp.Body() on the returned resp doesn't
	// error
	if err != nil {
		return resp, err
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return resp, err
}

// func WriteHttpResponse(resp http.Response, buffer *bytes.Buffer) error {
// 	err := resp.Write(buffer)
// 	return err
// }

// func ReadHttpResponse(buffer *bytes.Buffer) (resp *http.Response, err error) {
// 	r := bufio.NewReader(buffer)
// 	resp, err = http.ReadResponse(r, nil)
// 	return resp, err
// }
