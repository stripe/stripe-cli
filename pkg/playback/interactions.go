package playback

import (
	"errors"
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

// interactions stores info on a single request + response interaction
type interaction struct {
	// may have other fields like -- sequence number
	Type     interactionType
	Request  httpRequest
	Response httpResponse
}

// cassette is used to store a sequence of interactions that happened part of a single action
type cassette []interaction

// An interactionRecorder can read in a cassette, and write them out
// in a serialized format.
type interactionRecorder struct {
	writer         io.Writer
	cassette       cassette
	reqSerializer  serializer
	respSerializer serializer
}

func newInteractionRecorder(writer io.Writer, reqSerializer serializer, respSerializer serializer) (recorder *interactionRecorder, err error) {
	recorder = &interactionRecorder{
		writer:         writer,
		reqSerializer:  reqSerializer,
		respSerializer: respSerializer,
	}

	return recorder, nil
}

// write adds a new interaction to the current cassette.
func (recorder *interactionRecorder) write(typeOfInteraction interactionType, req httpRequest, resp httpResponse) error {
	recorder.cassette = append(recorder.cassette, interaction{Type: typeOfInteraction, Request: req, Response: resp})
	return nil
}

// saveAndClose persists the cassette to the filesystem.
func (recorder *interactionRecorder) saveAndClose() error {
	// open cassette file
	// loop over interactions
	// format interactions with recorder.interactionSerializer
	// write to file
	// end of loop => close file

	// yaml, err := yaml.Marshal(recorder.cassette)
	// if err != nil {
	// 	return err
	// }

	var serializedInteractions []interface{}

	_, err = recorder.writer.Write(yaml)
	return err
}

// An interactionReplayer contains a set of recorded interactions and exposes
// functionality to play them back.
type interactionReplayer struct {
	cursor           int
	comparator       requestComparator
	cassette         cassette
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
	if len(replayer.cassette) == 0 {
		return nil, errors.New("nothing left in cassette to replay")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	for idx, interaction := range replayer.cassette {
		// var reader io.Reader = bytes.NewReader(val.Request)
		// requestStruct, err := replayer.reqDeserializer(&reader)
		// if err != nil {
		// 	return nil, fmt.Errorf("error when deserializing cassette: %w", err)
		// }

		accept, shortCircuit := replayer.comparator(interaction.Request, req)

		if accept {
			// var reader io.Reader = bytes.NewReader(val.Response)
			// responseStruct, err := replayer.respDeserializer(&reader)

			// if err != nil {
			// 	return nil, fmt.Errorf("error when deserializing cassette: %w", err)
			// }

			lastAccepted = interaction.Response
			acceptedIdx = idx

			if shortCircuit {
				break
			}
		}
	}
	if acceptedIdx != -1 {
		// remove interactions that were accepted from tape
		replayer.cassette = append(replayer.cassette[:acceptedIdx], replayer.cassette[acceptedIdx+1:]...)
		return &lastAccepted, nil
	}

	return nil, errors.New("no matching events")
}

func (replayer *interactionReplayer) interactionsRemaining() int {
	return len(replayer.cassette)
}

func (replayer *interactionReplayer) peekFront() (interaction, error) {
	if len(replayer.cassette) == 0 {
		return interaction{}, errors.New("nothing left in cassette to replay")
	}

	return replayer.cassette[0], nil
}

func (replayer *interactionReplayer) popFront() (interaction, error) {
	if len(replayer.cassette) == 0 {
		return interaction{}, errors.New("nothing left in cassette to replay")
	}

	first := replayer.cassette[0]
	replayer.cassette = replayer.cassette[1:]
	return first, nil
}
