package playback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Event struct {
	Name string
	Id   int
}

func (e Event) toBytes() (bytes []byte, err error) {
	return json.Marshal(e)
}

func (e Event) fromBytes(input *io.Reader) (val interface{}, err error) {
	out := Event{}
	bytes, err := ioutil.ReadAll(*input)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(bytes, &out)
	return out, err
}

func toEvent(input interface{}) Event {
	jsonString, _ := json.Marshal(input)
	// convert json to struct
	cast1 := Event{}
	json.Unmarshal(jsonString, &cast1)

	return cast1
}

func TestSerializableEventInterface(t *testing.T) {
	event := Event{"John", 1}

	rawBytes, err := event.toBytes()
	check(err)

	var r io.Reader = bytes.NewReader(rawBytes)

	newEvent, err := event.fromBytes(&r)
	check(err)
	assert.Equal(t, event, newEvent)
}

func TestSequentialPlayback(t *testing.T) {
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	// --- Set up recording
	var writeBuffer bytes.Buffer
	recorder, err := NewRecorder(&writeBuffer)

	if err != nil {
		panic(err)
	}

	// --- Record
	fmt.Println("Recording...")

	s1 := Event{Name: "Request 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(OutgoingInteraction, s1, r1)

	s2 := Event{Name: "Request 2", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(OutgoingInteraction, s2, r2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	// --- Load cassette and replay: matching on sequence
	fmt.Println(writeBuffer.Len())
	replayer, err := NewReplayer(&writeBuffer, Event{}, Event{}, sequentialComparator)
	assert.NoError(t, err)

	fmt.Println("Replaying...")
	// feed the requests in *backwards* order, but responses come back in original order
	replayedResp1, err1 := replayer.Write(s2)
	replayedResp2, err2 := replayer.Write(s1)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	castResp1 := (*replayedResp1).(Event)
	castResp2 := (*replayedResp2).(Event)

	if err1 != nil {
		fmt.Println(err1)
	} else {
		fmt.Println(replayedResp1)
	}
	assert.Equal(t, r1, castResp1)

	if err2 != nil {
		fmt.Println(err2)
	} else {
		fmt.Println(castResp2)
	}

	assert.Equal(t, r2, castResp2)
}

func TestFirstMatchingEvent(t *testing.T) {
	firstMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		// convert map to json
		cast1 := toEvent(req1)
		cast2 := req2.(Event)
		return (cast1.Name == cast2.Name), true
	}

	// -- Set up recorder
	var writeBuffer bytes.Buffer
	recorder, err := NewRecorder(&writeBuffer)

	if err != nil {
		panic(err)
	}

	// --- Record
	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(OutgoingInteraction, s1, r1)

	s2 := Event{Name: "Event 2", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(OutgoingInteraction, s2, r2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	// --- Replay - returning the first match
	replayer, err := NewReplayer(&writeBuffer, Event{}, Event{}, firstMatchingComparator)
	assert.NoError(t, err)

	fmt.Println("Replaying...")

	// Send an unrecognized request
	_, err2 := replayer.Write(Event{})
	assert.EqualError(t, err2, "No matching events.")

	// Send the requests in opposite order, responses should come in opposite order
	replayedResp2, err2 := replayer.Write(s2)
	replayedResp1, err1 := replayer.Write(s1)

	castResp1 := (*replayedResp1).(Event)
	castResp2 := (*replayedResp2).(Event)

	assert.NoError(t, err2)
	assert.NoError(t, err1)
	assert.Equal(t, castResp1, r1)
	assert.Equal(t, castResp2, r2)
}

func TestLastMatchingEvent(t *testing.T) {
	lastMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		jsonString, _ := json.Marshal(req1)
		// convert json to struct
		cast1 := Event{}
		json.Unmarshal(jsonString, &cast1)

		cast2 := req2.(Event)
		return (cast1.Name == cast2.Name), false // false to return last match
	}

	var writeBuffer bytes.Buffer
	recorder, err := NewRecorder(&writeBuffer)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(OutgoingInteraction, s1, r1)

	s2 := Event{Name: "Event 1", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(OutgoingInteraction, s2, r2)

	s3 := Event{Name: "Event 3", Id: 52}
	r3 := Event{Name: "Response 3", Id: 52}
	fmt.Printf("%+v | %+v\n", s3, r3)
	recorder.Write(OutgoingInteraction, s3, r3)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	replayer, err := NewReplayer(&writeBuffer, Event{}, Event{}, lastMatchingComparator)
	assert.NoError(t, err)

	fmt.Println("Replaying...")
	fmt.Println("Should return last matching event to \"Event 1\"")
	respA, errA := replayer.Write(s1)
	castA := (*respA).(Event)

	check(errA)
	assert.Equal(t, castA.Name, r2.Name)
	assert.Equal(t, castA.Id, r2.Id)

	fmt.Println("Should match the single \"Event 3\"")
	respB, errB := replayer.Write(s3)
	castB := (*respB).(Event)
	check(errB)
	assert.Equal(t, r3, castB)

	fmt.Println("Should return first matching event to \"Event 1\" - since the last one was removed")
	respC, errC := replayer.Write(s1)
	castC := toEvent(respC)
	check(errC)
	assert.Equal(t, r1, castC)
}
