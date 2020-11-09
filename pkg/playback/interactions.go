package playback

import (
	"errors"
	"io"
	"io/ioutil"
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

// interaction stores info on a single request + response interaction
// interactions are on the tape ready to be persisted, so Request/Response
// are interface{} - already encoded by the serializer, ready to be persisted.
type interaction struct {
	// may have other fields like -- sequence number
	Type     interactionType
	Request  httpRequest
	Response httpResponse
}

// Cassette is used to store a sequence of interactions that happened part of a single action
type Cassette []interaction

// An interactionRecorder can read and write a cassette in a serialized format.
type interactionRecorder struct {
	writer     io.Writer
	cassette   Cassette
	serializer YAMLSerializer
}

func newInteractionRecorder(writer io.Writer, serializer serializer) (recorder *interactionRecorder, err error) {
	recorder = &interactionRecorder{
		writer:     writer,
		serializer: YAMLSerializer{},
	}

	return recorder, nil
}

// write adds a new interaction to the current cassette.
func (recorder *interactionRecorder) write(typeOfInteraction interactionType, req httpRequest, resp httpResponse) {
	interaction := interaction{
		Type:     typeOfInteraction,
		Request:  req,
		Response: resp,
	}
	recorder.cassette = append(recorder.cassette, interaction)
}

// saveAndClose persists the cassette to the filesystem.
func (recorder *interactionRecorder) saveAndClose() error {
	output, err := recorder.serializer.EncodeCassette(recorder.cassette)
	if err != nil {
		return err
	}

	_, err = recorder.writer.Write(output)
	return err
}

// An interactionReplayer contains a set of recorded interactions and exposes
// functionality to play them back.
type interactionReplayer struct {
	cursor     int
	comparator requestComparator
	cassette   Cassette
	serializer serializer
}

// newInteractionReplayer uses the serializer to decode the cassette file and ready the cassette for reading.
func newInteractionReplayer(reader io.Reader, serializer serializer, comparator requestComparator) (replayer *interactionReplayer, err error) {
	replayer = &interactionReplayer{}
	replayer.cursor = 0
	replayer.comparator = comparator
	replayer.serializer = serializer

	buffer, err := ioutil.ReadAll(reader)
	if err != nil {
		return replayer, err
	}

	cassette, err := serializer.DecodeCassette(buffer)
	if err != nil {
		return replayer, err
	}
	replayer.cassette = cassette

	return replayer, nil
}

// write parses the cassette for matching responses and returns them
// core "replay" logic
func (replayer *interactionReplayer) write(req *httpRequest) (resp *interface{}, err error) {
	if len(replayer.cassette) == 0 {
		return nil, errors.New("nothing left in cassette to replay")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	for idx, interaction := range replayer.cassette {
		accept, shortCircuit := replayer.comparator(interaction.Request, *req)

		if accept {
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
