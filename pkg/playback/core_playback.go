package playback

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

// A function that compares a provided request against a recorded request from a cassette
// and determines 1) whether that are equivalent 2) whether we should short-circuit
// our search (return this one immediately, or keep scanning the cassette)
type RequestComparator func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool)

// A struct that can be serialized/deserialized to bytes.
type Serializable interface {
	toBytes() (bytes []byte, err error)
	fromBytes(input *io.Reader) (val interface{}, err error)
}

type Recorder struct {
	writer       io.Writer
	interactions []CassettePair
}

func NewRecorder(writer io.Writer) (recorder *Recorder, err error) {
	recorder = &Recorder{}

	recorder.writer = writer

	recorder.interactions = make([]CassettePair, 0)

	return recorder, nil
}

// takes a generic struct
func (recorder *Recorder) Write(interactionType InteractionType, req Serializable, resp Serializable) error {
	reqBytes, err := req.toBytes()

	if err != nil {
		return err
	}

	respBytes, err := resp.toBytes()

	if err != nil {
		return err
	}

	recorder.interactions = append(recorder.interactions, CassettePair{Type: interactionType, Request: reqBytes, Response: respBytes})

	// _, err = recorder.fileHandle.Write(reqBytes)
	// recorder.fileHandle.Write([]byte("\n"))

	// if err != nil {
	// 	return err
	// }

	// _, err = recorder.fileHandle.Write(respBytes)
	// recorder.fileHandle.Write([]byte("\n"))

	return err
}

// Contains binary data representing a generic request and response saved in a cassette.
type InteractionType int

const (
	OutgoingInteraction InteractionType = iota // eg: Stripe API requests
	IncomingInteraction                        // eg: webhooks
)

type CassettePair struct {
	// may have other fields like -- sequence number
	Type     InteractionType
	Request  []byte
	Response []byte
}

// Top-level struct used to serialize cassette data to a YAML file
type CassetteYaml struct {
	Interactions []CassettePair
}

func (recorder *Recorder) Close() error {
	// Write everything to a YAML File

	// Put everything in a wrapping CassetteYaml struct that can be marshaled
	cassette := CassetteYaml{}
	cassette.Interactions = recorder.interactions

	yamlBytes, err := yaml.Marshal(cassette)
	if err != nil {
		return err
	}

	_, err = recorder.writer.Write(yamlBytes)
	return err
}

type Replayer struct {
	reader       io.Reader
	historyIndex int
	comparator   RequestComparator
	cassette     CassetteYaml
	reqType      Serializable
	respType     Serializable
}

func NewReplayer(reader io.Reader, reqType Serializable, respType Serializable, comparator RequestComparator) (replayer *Replayer, err error) {
	replayer = &Replayer{}
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

func (replayer *Replayer) Write(req Serializable) (resp *interface{}, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return nil, errors.New("Nothing left in cassette to replay.")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	// TODO: this can be optimized to do the deserialization from bytes
	// once per interaction, instead of every time
	for idx, val := range replayer.cassette.Interactions {
		// Deserialize the recorded request in this interaction
		var reader io.Reader = bytes.NewReader(val.Request)
		requestStruct, err := replayer.reqType.fromBytes(&reader)
		if err != nil {
			return nil, fmt.Errorf("Error when deserializing cassette: %w", err)
		}

		// Compare it with the provided request
		accept, shortCircuit := replayer.comparator(requestStruct, req)

		// If it matches, then deserialize the matching recorded response
		if accept {
			var reader io.Reader = bytes.NewReader(val.Response)
			responseStruct, err := replayer.respType.fromBytes(&reader)

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

func (replayer *Replayer) InteractionsRemaining() int {
	return len(replayer.cassette.Interactions)
}

func (replayer *Replayer) PeekFront() (interaction CassettePair, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return CassettePair{}, errors.New("Nothing left in cassette to replay.")
	}

	return replayer.cassette.Interactions[0], nil
}

func (replayer *Replayer) PopFront() (interaction CassettePair, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return CassettePair{}, errors.New("Nothing left in cassette to replay.")
	}

	first := replayer.cassette.Interactions[0]
	replayer.cassette.Interactions = replayer.cassette.Interactions[1:]
	return first, nil
}

func (replayer *Replayer) Close() {

}
