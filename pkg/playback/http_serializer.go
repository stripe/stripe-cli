package playback

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

type serializer func(input interface{}) (bytes []byte, err error)
type deserializer func(input *io.Reader) (value interface{}, err error)

func httpRequestToBytes(input interface{}) (data []byte, err error) {
	var buffer = &bytes.Buffer{}
	var request *http.Request
	request, castOk := input.(*http.Request)

	if !castOk {
		return buffer.Bytes(), errors.New("input struct is not of type *http.Request")
	}

	err = request.Write(buffer)
	return buffer.Bytes(), err
}

func httpRequestfromBytes(input *io.Reader) (val interface{}, err error) {
	r := bufio.NewReader(*input)
	req, err := http.ReadRequest(r)
	return req, err
}

func httpResponseToBytes(input interface{}) (data []byte, err error) {
	var buffer = &bytes.Buffer{}
	var response *http.Response
	response, castOk := input.(*http.Response)

	if !castOk {
		return buffer.Bytes(), errors.New("input struct is not of type *http.Response")
	}

	err = response.Write(buffer)
	return buffer.Bytes(), err
}

func httpResponsefromBytes(input *io.Reader) (val interface{}, err error) {
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
