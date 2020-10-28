package playback

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// requestComparator compares 2 structs in the context of comparing a given request struct against
// a request recorded in the cassette.
// It then determines
// 1) whether they are equivalent
// 2) whether we should short-circuit our search (return this one immediately, or keep scanning the cassette)
type requestComparator func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool)

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

// cassetteYaml is used to store interaction data to be serialized to a YAML file
type cassetteYaml struct {
	Interactions []cassettePair
}

// A interactionRecorder can read in a sequence of interactions, and write them out
// in a serialized format.
type interactionRecorder struct {
	writer         io.Writer
	interactions   []cassettePair
	reqSerializer  serializer
	respSerializer serializer
}

func newInteractionRecorder(writer io.Writer, reqSerializer serializer, respSerializer serializer) (recorder *interactionRecorder, err error) {
	recorder = &interactionRecorder{
		writer:         writer,
		reqSerializer:  reqSerializer,
		respSerializer: respSerializer,
		interactions:   make([]cassettePair, 0),
	}

	return recorder, nil
}

// takes a generic struct
func (recorder *interactionRecorder) write(typeOfInteraction interactionType, req interface{}, resp interface{}) error {
	reqBytes, err := recorder.reqSerializer(req)

	if err != nil {
		return err
	}

	respBytes, err := recorder.respSerializer(resp)

	if err != nil {
		return err
	}

	recorder.interactions = append(recorder.interactions, cassettePair{Type: typeOfInteraction, Request: reqBytes, Response: respBytes})

	return err
}

func (recorder *interactionRecorder) saveAndClose() error {
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
	cursor           int
	comparator       requestComparator
	cassette         cassetteYaml
	respDeserializer deserializer
	reqDeserializer  deserializer
}

func newInteractionReplayer(reader io.Reader, reqDeserializer deserializer, respDeserializer deserializer, comparator requestComparator) (replayer *interactionReplayer, err error) {
	replayer = &interactionReplayer{}
	replayer.cursor = 0
	replayer.comparator = comparator
	replayer.reqDeserializer = reqDeserializer
	replayer.respDeserializer = respDeserializer

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

func (replayer *interactionReplayer) write(req interface{}) (resp *interface{}, err error) {
	if len(replayer.cassette.Interactions) == 0 {
		return nil, errors.New("nothing left in cassette to replay")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	for idx, val := range replayer.cassette.Interactions {
		var reader io.Reader = bytes.NewReader(val.Request)
		requestStruct, err := replayer.reqDeserializer(&reader)
		if err != nil {
			return nil, fmt.Errorf("error when deserializing cassette: %w", err)
		}

		accept, shortCircuit := replayer.comparator(requestStruct, req)

		if accept {
			var reader io.Reader = bytes.NewReader(val.Response)
			responseStruct, err := replayer.respDeserializer(&reader)

			if err != nil {
				return nil, fmt.Errorf("error when deserializing cassette: %w", err)
			}

			lastAccepted = responseStruct
			acceptedIdx = idx

			if shortCircuit {
				break
			}
		}
	}
	if acceptedIdx != -1 {
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
