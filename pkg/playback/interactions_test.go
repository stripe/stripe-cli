package playback

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

type basicEvent struct {
	Name string
	ID   int
}

func BasicEventToBytes(input interface{}) (bytes []byte, err error) {
	event, castOk := input.(basicEvent)
	if !castOk {
		return []byte{}, errors.New("input is not of type basicEvent")
	}
	return json.Marshal(event)
}

func BasicEventFromBytes(input *io.Reader) (val interface{}, err error) {
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

	rawBytes, err := BasicEventToBytes(event)
	check(t, err)

	var r io.Reader = bytes.NewReader(rawBytes)

	newEvent, err := BasicEventFromBytes(&r)
	check(t, err)
	assert.Equal(t, event, newEvent)
}

func TestSequentialPlayback(t *testing.T) {
	sequentialComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		return true, true
	}

	var writeBuffer bytes.Buffer
	recorder, err := newInteractionRecorder(&writeBuffer, BasicEventToBytes, BasicEventToBytes)

	if err != nil {
		t.Fatal(err)
	}

	s1 := basicEvent{Name: "Request 1", ID: 23}
	r1 := basicEvent{Name: "Response 1", ID: 23}
	recorder.write(outgoingInteraction, s1, r1)

	s2 := basicEvent{Name: "Request 2", ID: 46}
	r2 := basicEvent{Name: "Response 2", ID: 46}
	recorder.write(outgoingInteraction, s2, r2)

	err = recorder.saveAndClose()

	if err != nil {
		t.Fatal(err)
	}

	replayer, err := newInteractionReplayer(&writeBuffer, BasicEventFromBytes, BasicEventFromBytes, sequentialComparator)
	assert.NoError(t, err)

	replayedResp1, err1 := replayer.write(s2)
	replayedResp2, err2 := replayer.write(s1)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	castResp1 := (*replayedResp1).(basicEvent)
	castResp2 := (*replayedResp2).(basicEvent)

	assert.Equal(t, r1, castResp1)

	assert.Equal(t, r2, castResp2)
}

func TestFirstMatchingEvent(t *testing.T) {
	firstMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		// convert map to json
		cast1 := toEvent(req1)
		cast2 := req2.(basicEvent)
		return (cast1.Name == cast2.Name), true
	}

	var writeBuffer bytes.Buffer
	recorder, err := newInteractionRecorder(&writeBuffer, BasicEventToBytes, BasicEventToBytes)

	if err != nil {
		t.Fatal(err)
	}

	s1 := basicEvent{Name: "Event 1", ID: 23}
	r1 := basicEvent{Name: "Response 1", ID: 23}
	recorder.write(outgoingInteraction, s1, r1)

	s2 := basicEvent{Name: "Event 2", ID: 46}
	r2 := basicEvent{Name: "Response 2", ID: 46}
	recorder.write(outgoingInteraction, s2, r2)

	err = recorder.saveAndClose()

	if err != nil {
		t.Fatal(err)
	}

	replayer, err := newInteractionReplayer(&writeBuffer, BasicEventFromBytes, BasicEventFromBytes, firstMatchingComparator)
	assert.NoError(t, err)

	_, err2 := replayer.write(basicEvent{})
	assert.EqualError(t, err2, "no matching events")

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
	recorder, err := newInteractionRecorder(&writeBuffer, BasicEventToBytes, BasicEventToBytes)

	if err != nil {
		t.Fatal(err)
	}

	s1 := basicEvent{Name: "Event 1", ID: 23}
	r1 := basicEvent{Name: "Response 1", ID: 23}
	recorder.write(outgoingInteraction, s1, r1)

	s2 := basicEvent{Name: "Event 1", ID: 46}
	r2 := basicEvent{Name: "Response 2", ID: 46}
	recorder.write(outgoingInteraction, s2, r2)

	s3 := basicEvent{Name: "Event 3", ID: 52}
	r3 := basicEvent{Name: "Response 3", ID: 52}
	recorder.write(outgoingInteraction, s3, r3)

	err = recorder.saveAndClose()

	if err != nil {
		t.Fatal(err)
	}

	replayer, err := newInteractionReplayer(&writeBuffer, BasicEventFromBytes, BasicEventFromBytes, lastMatchingComparator)
	assert.NoError(t, err)

	respA, errA := replayer.write(s1)
	castA := (*respA).(basicEvent)

	check(t, errA)
	assert.Equal(t, castA.Name, r2.Name)
	assert.Equal(t, castA.ID, r2.ID)

	respB, errB := replayer.write(s3)
	castB := (*respB).(basicEvent)
	check(t, errB)
	assert.Equal(t, r3, castB)

	respC, errC := replayer.write(s1)
	castC := toEvent(respC)
	check(t, errC)
	assert.Equal(t, r1, castC)
}
