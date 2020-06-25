package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type RequestComparator func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool)

type Serializable interface {
	toBytes() (bytes []byte, err error)
	fromBytes(bytes *bytes.Buffer) (val interface{}, err error)
}

type VcrRecorder struct {
	writer    io.Writer
	requests  [][]byte
	responses [][]byte
}

func NewVcrRecorder(writer io.Writer) (recorder *VcrRecorder, err error) {
	recorder = &VcrRecorder{}

	recorder.writer = writer

	recorder.requests = make([][]byte, 0)
	recorder.responses = make([][]byte, 0)

	return recorder, nil
}

// takes a generic struct
func (recorder *VcrRecorder) Write(req Serializable, resp Serializable) error {
	reqBytes, err := req.toBytes()

	if err != nil {
		return err
	}

	respBytes, err := resp.toBytes()

	if err != nil {
		return err
	}

	recorder.requests = append(recorder.requests, reqBytes)
	recorder.responses = append(recorder.responses, respBytes)

	// _, err = recorder.fileHandle.Write(reqBytes)
	// recorder.fileHandle.Write([]byte("\n"))

	// if err != nil {
	// 	return err
	// }

	// _, err = recorder.fileHandle.Write(respBytes)
	// recorder.fileHandle.Write([]byte("\n"))

	return err
}

type CassettePair struct {
	// may have other fields like -- sequence number
	Request  []byte
	Response []byte
}

type CassetteYaml struct {
	Interactions []CassettePair
}

func (recorder *VcrRecorder) Close() error {
	fmt.Println("Calling recorder.Close()")

	// Write everything to a YAML File

	// Put everything in a wrapping CassetteYaml struct that can be marshalled
	cassette := CassetteYaml{}
	interactions := make([]CassettePair, 0)

	for i := 0; i < len(recorder.requests); i++ {
		pair := CassettePair{}
		pair.Request = recorder.requests[i]
		pair.Response = recorder.responses[i]

		interactions = append(interactions, pair)
	}
	cassette.Interactions = interactions

	yamlBytes, err := yaml.Marshal(cassette)
	if err != nil {
		return err
	}

	_, err = recorder.writer.Write(yamlBytes)
	return err

	// return recorder.fileHandle.Close()
}

type VcrReplayer struct {
	reader       io.Reader
	historyIndex int
	comparator   RequestComparator
	cassette     CassetteYaml
	reqType      Serializable
	respType     Serializable
}

func NewVcrReplayer(reader io.Reader, reqType Serializable, respType Serializable, comparator RequestComparator) (replayer *VcrReplayer, err error) {
	replayer = &VcrReplayer{}
	replayer.historyIndex = 0
	replayer.comparator = comparator
	replayer.reqType = reqType
	replayer.respType = respType

	yamlBuf, err := ioutil.ReadAll(reader)
	if err != nil {
		return replayer, err
	}

	err = yaml.Unmarshal(yamlBuf, &replayer.cassette)
	if err != nil {
		return replayer, err
	}

	return replayer, nil

}

func (replayer *VcrReplayer) Write(req Serializable) (resp *interface{}, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return nil, errors.New("Nothing left in cassette to replay.")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	// TODO: this should be optimized to do the deserialization from bytes
	// once per interaction, instead of every time
	for idx, val := range replayer.cassette.Interactions {
		// TODO: This deserialization boilerplate is messy - refactor it
		var b bytes.Buffer
		b.Write(val.Request)
		requestStruct, err := replayer.reqType.fromBytes(&b)

		if err != nil {
			return nil, fmt.Errorf("Error when deserializing cassette: %w", err)
		}

		accept, shortCircuit := replayer.comparator(requestStruct, req)

		if accept {
			var respBuffer bytes.Buffer
			respBuffer.Write(val.Response)
			responseStruct, err := replayer.respType.fromBytes(&respBuffer)

			if err != nil {
				return nil, errors.New("Error when deserializing cassette.")
			}

			lastAccepted = responseStruct
			acceptedIdx = idx

			if shortCircuit {
				break
			}
		}

	}

	if acceptedIdx != -1 {
		// remove the matched event
		replayer.cassette.Interactions = append(replayer.cassette.Interactions[:acceptedIdx], replayer.cassette.Interactions[acceptedIdx+1:]...)
		return &lastAccepted, nil
	}

	return nil, errors.New("No matching events.")
}

func (replayer *VcrReplayer) Close() {

}

type Pair struct {
	First  interface{}
	Second interface{}
}

type BetterEvent struct {
	Name string
	Type string
	Id   int
}
