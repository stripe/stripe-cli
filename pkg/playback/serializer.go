package playback

import (
	"bufio"
	"bytes"
	"net/http"
)

// type Serializer interface {

// 	func write()

// }

type HttpRequestSerializable struct {
	baseRequest *http.Request
}

func NewSerializableHttpRequest(r *http.Request) HttpRequestSerializable {
	return HttpRequestSerializable{r}
}

func (reqWrapper HttpRequestSerializable) toBytes() (data []byte, err error) {
	var buffer = &bytes.Buffer{}
	err = reqWrapper.baseRequest.Write(buffer)
	return buffer.Bytes(), err
}

func (reqWrapper HttpRequestSerializable) fromBytes(buffer *bytes.Buffer) (val interface{}, err error) {
	r := bufio.NewReader(buffer)
	req, err := http.ReadRequest(r)
	return req, err
}

type HttpResponseSerializable struct {
	baseResponse *http.Response
}

func NewSerializableHttpResponse(r *http.Response) HttpResponseSerializable {
	return HttpResponseSerializable{r}
}

func (respWrapper HttpResponseSerializable) toBytes() (data []byte, err error) {
	var buffer = &bytes.Buffer{}

	err = respWrapper.baseResponse.Write(buffer)
	return buffer.Bytes(), err
}

func (respWrapper HttpResponseSerializable) fromBytes(buffer *bytes.Buffer) (val interface{}, err error) {
	r := bufio.NewReader(buffer)
	req, err := http.ReadResponse(r, nil)
	return req, err
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
