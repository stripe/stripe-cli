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

// requestComparator compares 2 structs in the context of comparing a given request struct against
// a request recorded in the request/
// It then determines 1) whether that are equivalent 2) whether we should short-circuit
// our search (return this one immediately, or keep scanning the cassette)
type requestComparator func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool)

// A struct that can be serialized/deserialized to bytes.
type serializable interface {
	toBytes() (bytes []byte, err error)
	fromBytes(input *io.Reader) (val interface{}, err error)
}

// A interactionRecorder can read in a sequence of interactions, and write them out
// in a serialized format.
type interactionRecorder struct {
	writer       io.Writer
	interactions []cassettePair
}

func newInteractionRecorder(writer io.Writer) (recorder *interactionRecorder, err error) {
	recorder = &interactionRecorder{}

	recorder.writer = writer

	recorder.interactions = make([]cassettePair, 0)

	return recorder, nil
}

// takes a generic struct
func (recorder *interactionRecorder) write(typeOfInteraction interactionType, req serializable, resp serializable) error {
	reqBytes, err := req.toBytes()

	if err != nil {
		return err
	}

	respBytes, err := resp.toBytes()

	if err != nil {
		return err
	}

	recorder.interactions = append(recorder.interactions, cassettePair{Type: typeOfInteraction, Request: reqBytes, Response: respBytes})

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
type interactionType int

const (
	outgoingInteraction interactionType = iota // eg: Stripe API requests
	incomingInteraction                        // eg: webhooks
)

// cassettePairs stores info on a single request + response interaction
type cassettePair struct {
	// may have other fields like -- sequence number
	Type     interactionType
	Request  []byte
	Response []byte
}

// cassetteYaml is used store interaction data to be serialized a YAML file
type cassetteYaml struct {
	Interactions []cassettePair
}

func (recorder *interactionRecorder) close() error {
	// Write everything to a YAML File

	// Put everything in a wrapping CassetteYaml struct that can be marshaled
	cassette := cassetteYaml{}
	cassette.Interactions = recorder.interactions

	yamlBytes, err := yaml.Marshal(cassette)
	if err != nil {
		return err
	}

	_, err = recorder.writer.Write(yamlBytes)
	return err
}

// A interactionReplayer contains a set of recorded interactions and exposes
// functionality to play them back.
type interactionReplayer struct {
	historyIndex int
	comparator   requestComparator
	cassette     cassetteYaml
	reqType      serializable
	respType     serializable
}

func newInteractionReplayer(reader io.Reader, reqType serializable, respType serializable, comparator requestComparator) (replayer *interactionReplayer, err error) {
	replayer = &interactionReplayer{}
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

func (replayer *interactionReplayer) write(req serializable) (resp *interface{}, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return nil, errors.New("nothing left in cassette to replay")
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
				return nil, errors.New("error when deserializing cassette")
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

	return nil, errors.New("no matching events")
}

func (replayer *interactionReplayer) interactionsRemaining() int {
	return len(replayer.cassette.Interactions)
}

func (replayer *interactionReplayer) peekFront() (interaction cassettePair, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return cassettePair{}, errors.New("nothing left in cassette to replay")
	}

	return replayer.cassette.Interactions[0], nil
}

func (replayer *interactionReplayer) popFront() (interaction cassettePair, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return cassettePair{}, errors.New("nothing left in cassette to replay")
	}

	first := replayer.cassette.Interactions[0]
	replayer.cassette.Interactions = replayer.cassette.Interactions[1:]
	return first, nil
}
