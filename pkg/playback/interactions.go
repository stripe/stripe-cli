package playback

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
	Request  interface{}
	Response interface{}
}

// Cassette is used to store a sequence of interactions that happened part of a single action
type Cassette []interaction
