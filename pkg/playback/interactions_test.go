package playback

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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

func toHTTPRequest(input interface{}) httpRequest {
	jsonString, _ := json.Marshal(input)
	// convert json to struct
	cast1 := httpRequest{}
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
	recorder := newRecorder("example.com", "example.com/wh", YAMLSerializer{})
	recorder.insertCassette(&writeBuffer)

	s1 := httpRequest{Method: "POST"}
	r1 := httpResponse{Headers: http.Header{}, Body: []byte("body 1"), StatusCode: 200}
	recorder.write(outgoingInteraction, s1, r1)

	s2 := httpRequest{Method: "GET"}
	r2 := httpResponse{Headers: http.Header{}, Body: []byte("body 2"), StatusCode: 300}
	recorder.write(outgoingInteraction, s2, r2)

	recorder.saveAndClose()

	replayer := newReplayer("example.com/wh", YAMLSerializer{}, sequentialComparator)
	replayer.readCassette(&writeBuffer)

	replayedResp1, err1 := replayer.write(&s2)
	replayedResp2, err2 := replayer.write(&s1)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	castResp1 := (*replayedResp1).(httpResponse)
	castResp2 := (*replayedResp2).(httpResponse)

	assert.Equal(t, r1, castResp1)

	assert.Equal(t, r2, castResp2)
}

func TestFirstMatchingEvent(t *testing.T) {
	firstMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		// convert map to json
		cast1 := toHTTPRequest(req1)
		cast2 := req2.(httpRequest)
		return (cast1.Method == cast2.Method), true
	}

	var writeBuffer bytes.Buffer
	recorder := newRecorder("example.com", "example.com/wh", YAMLSerializer{})
	recorder.insertCassette(&writeBuffer)

	s1 := httpRequest{Method: "POST"}
	r1 := httpResponse{Headers: http.Header{}, Body: []byte("body 1"), StatusCode: 200}
	recorder.write(outgoingInteraction, s1, r1)

	s2 := httpRequest{Method: "GET"}
	r2 := httpResponse{Headers: http.Header{}, Body: []byte("body 2"), StatusCode: 300}
	recorder.write(outgoingInteraction, s2, r2)

	recorder.saveAndClose()

	replayer := newReplayer("example.com/wh", YAMLSerializer{}, firstMatchingComparator)
	replayer.readCassette(&writeBuffer)

	_, err2 := replayer.write(&httpRequest{})
	assert.EqualError(t, err2, "no matching events")

	replayedResp2, err2 := replayer.write(&s2)
	replayedResp1, err1 := replayer.write(&s1)

	castResp1 := (*replayedResp1).(httpResponse)
	castResp2 := (*replayedResp2).(httpResponse)

	assert.NoError(t, err2)
	assert.NoError(t, err1)
	assert.Equal(t, castResp1, r1)
	assert.Equal(t, castResp2, r2)
}

func TestLastMatchingEvent(t *testing.T) {
	lastMatchingComparator := func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool) {
		cast1 := req1.(httpRequest)
		cast2 := req2.(httpRequest)
		return (cast1.Method == cast2.Method), false // false to return last match
	}

	var writeBuffer bytes.Buffer
	recorder := newRecorder("example.com", "example.com/wh", YAMLSerializer{})
	recorder.insertCassette(&writeBuffer)

	s1 := httpRequest{Method: "POST"}
	r1 := httpResponse{StatusCode: 200}
	recorder.write(outgoingInteraction, s1, r1)

	s2 := httpRequest{Method: "GET"}
	r2 := httpResponse{StatusCode: 300}
	recorder.write(outgoingInteraction, s2, r2)

	s3 := httpRequest{Method: "POST"}
	r3 := httpResponse{StatusCode: 400}
	recorder.write(outgoingInteraction, s3, r3)

	recorder.saveAndClose()

	replayer := newReplayer("example.com/wh", YAMLSerializer{}, lastMatchingComparator)
	replayer.readCassette(&writeBuffer)

	respA, errA := replayer.write(&s1)
	castA := (*respA).(httpResponse)
	check(t, errA)
	assert.Equal(t, castA.StatusCode, r3.StatusCode)

	respB, errB := replayer.write(&s2)
	castB := (*respB).(httpResponse)
	check(t, errB)
	assert.Equal(t, castB.StatusCode, r2.StatusCode)
}

func TestSaveAndCloseToYaml(t *testing.T) {
	// create cassette file
	cassetteFile, err := ioutil.TempFile(os.TempDir(), "test_cassette.yaml")
	defer os.Remove(cassetteFile.Name())
	check(t, err)

	// create new recorder
	recorder := newRecorder("example.com", "example.com/wh", YAMLSerializer{})
	recorder.insertCassette(cassetteFile)

	// write req/resp interaction to cassette
	header := http.Header{}
	header.Set("test", "header 1")
	s1 := httpRequest{Method: "POST", Body: []byte("hello world"), Headers: header}
	r1 := httpResponse{StatusCode: 200}
	recorder.write(outgoingInteraction, s1, r1)

	// persist cassette to file
	recorder.saveAndClose()

	expectedYAML :=
		`- type: 0
  request:
    method: POST
    body: hello world
    headers:
      Test:
      - header 1
    url:
      scheme: ""
      opaque: ""
      user: null
      host: ""
      path: ""
      rawpath: ""
      forcequery: false
      rawquery: ""
      fragment: ""
      rawfragment: ""
  response:
    headers: {}
    body: ""
    status_code: 200
`

	dat, _ := ioutil.ReadFile(cassetteFile.Name())
	assert.Equal(t, expectedYAML, string(dat))
}
