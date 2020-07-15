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

type basicEvent struct {
	Name string
	ID   int
}

func (e basicEvent) toBytes() (bytes []byte, err error) {
	return json.Marshal(e)
}

func (e basicEvent) fromBytes(input *io.Reader) (val interface{}, err error) {
	out := basicEvent{}
	bytes, err := ioutil.ReadAll(*input)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(bytes, &out)
	return out, err
}

func toEvent(input interface{}) basicEvent {
	jsonString, _ := json.Marshal(input)
	// convert json to struct
	cast1 := basicEvent{}
	json.Unmarshal(jsonString, &cast1)

	return cast1
}

func TestSerializableEventInterface(t *testing.T) {
	event := basicEvent{"John", 1}

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
	recorder, err := newInteractionRecorder(&writeBuffer)

	if err != nil {
		panic(err)
	}

	// --- Record
	fmt.Println("Recording...")

	s1 := basicEvent{Name: "Request 1", ID: 23}
	r1 := basicEvent{Name: "Response 1", ID: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.write(outgoingInteraction, s1, r1)

	s2 := basicEvent{Name: "Request 2", ID: 46}
	r2 := basicEvent{Name: "Response 2", ID: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.write(outgoingInteraction, s2, r2)

	err = recorder.close()

	if err != nil {
		panic(err)
	}

	// --- Load cassette and replay: matching on sequence
	fmt.Println(writeBuffer.Len())
	replayer, err := newInteractionReplayer(&writeBuffer, basicEvent{}, basicEvent{}, sequentialComparator)
	assert.NoError(t, err)

	fmt.Println("Replaying...")
	// feed the requests in *backwards* order, but responses come back in original order
	replayedResp1, err1 := replayer.write(s2)
	replayedResp2, err2 := replayer.write(s1)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	castResp1 := (*replayedResp1).(basicEvent)
	castResp2 := (*replayedResp2).(basicEvent)

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
		cast2 := req2.(basicEvent)
		return (cast1.Name == cast2.Name), true
	}

	// -- Set up recorder
	var writeBuffer bytes.Buffer
	recorder, err := newInteractionRecorder(&writeBuffer)

	if err != nil {
		panic(err)
	}

	// --- Record
	fmt.Println("Recording...")

	s1 := basicEvent{Name: "Event 1", ID: 23}
	r1 := basicEvent{Name: "Response 1", ID: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.write(outgoingInteraction, s1, r1)

	s2 := basicEvent{Name: "Event 2", ID: 46}
	r2 := basicEvent{Name: "Response 2", ID: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.write(outgoingInteraction, s2, r2)

	err = recorder.close()

	if err != nil {
		panic(err)
	}

	// --- Replay - returning the first match
	replayer, err := newInteractionReplayer(&writeBuffer, basicEvent{}, basicEvent{}, firstMatchingComparator)
	assert.NoError(t, err)

	fmt.Println("Replaying...")

	// Send an unrecognized request
	_, err2 := replayer.write(basicEvent{})
	assert.EqualError(t, err2, "no matching events")

	// Send the requests in opposite order, responses should come in opposite order
	replayedResp2, err2 := replayer.write(s2)
	replayedResp1, err1 := replayer.write(s1)

	castResp1 := (*replayedResp1).(basicEvent)
	castResp2 := (*replayedResp2).(basicEvent)

	assert.NoError(t, err2)
	assert.NoError(t, err1)
	assert.Equal(t, castResp1, r1)
	assert.Equal(t, castResp2, r2)
}

func TestLastMatchingEvent(t *testing.T) {
	lastMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		jsonString, _ := json.Marshal(req1)
		// convert json to struct
		cast1 := basicEvent{}
		json.Unmarshal(jsonString, &cast1)

		cast2 := req2.(basicEvent)
		return (cast1.Name == cast2.Name), false // false to return last match
	}

	var writeBuffer bytes.Buffer
	recorder, err := newInteractionRecorder(&writeBuffer)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := basicEvent{Name: "Event 1", ID: 23}
	r1 := basicEvent{Name: "Response 1", ID: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.write(outgoingInteraction, s1, r1)

	s2 := basicEvent{Name: "Event 1", ID: 46}
	r2 := basicEvent{Name: "Response 2", ID: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.write(outgoingInteraction, s2, r2)

	s3 := basicEvent{Name: "Event 3", ID: 52}
	r3 := basicEvent{Name: "Response 3", ID: 52}
	fmt.Printf("%+v | %+v\n", s3, r3)
	recorder.write(outgoingInteraction, s3, r3)

	err = recorder.close()

	if err != nil {
		panic(err)
	}

	replayer, err := newInteractionReplayer(&writeBuffer, basicEvent{}, basicEvent{}, lastMatchingComparator)
	assert.NoError(t, err)

	fmt.Println("Replaying...")
	fmt.Println("Should return last matching event to \"Event 1\"")
	respA, errA := replayer.write(s1)
	castA := (*respA).(basicEvent)

	check(errA)
	assert.Equal(t, castA.Name, r2.Name)
	assert.Equal(t, castA.ID, r2.ID)

	fmt.Println("Should match the single \"Event 3\"")
	respB, errB := replayer.write(s3)
	castB := (*respB).(basicEvent)
	check(errB)
	assert.Equal(t, r3, castB)

	fmt.Println("Should return first matching event to \"Event 1\" - since the last one was removed")
	respC, errC := replayer.write(s1)
	castC := toEvent(respC)
	check(errC)
	assert.Equal(t, r1, castC)
}
