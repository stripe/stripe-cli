package playback

import (
	"bufio"
	"bytes"
	"io"
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
	r := bufio.NewReader(*input)
	resp, err := http.ReadResponse(r, nil)
	// Don't close the body here, because will be read and closed by the caller
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
