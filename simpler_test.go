package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSequentialPlayback(t *testing.T) {
	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(s1, r1)

	s2 := Event{Name: "Event 2", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(s2, r2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	replayer, err := NewReplayer(filepath, func(req1 Event, req2 Event) (accept bool, shortCircuitNow bool) {
		return true, true
	})

	fmt.Println("Replaying...")
	replayedResp1, err1 := replayer.Write(s1)
	replayedResp2, err2 := replayer.Write(s2)

	if err1 != nil {
		fmt.Println(err1)
	} else {
		fmt.Println(replayedResp1)
	}
	assert.Equal(t, replayedResp1.Name, r1.Name)
	assert.Equal(t, replayedResp1.Id, r1.Id)

	if err2 != nil {
		fmt.Println(err2)
	} else {
		fmt.Println(replayedResp2)
	}

	assert.Equal(t, replayedResp2.Name, r2.Name)
	assert.Equal(t, replayedResp2.Id, r2.Id)

	// TODO: fix this
	// replayedResp1 = replayedResp1.(Event)
	// replayedResp2 = replayedResp2.(Event)

	// assert.Equal(t, replayedResp1, r1)
	// assert.Equal(t, replayedResp2, r2)
}

func TestFirstMatchingEvent(t *testing.T) {
	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(s1, r1)

	s2 := Event{Name: "Event 2", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(s2, r2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	replayer, err := NewReplayer(filepath, func(req1 Event, req2 Event) (accept bool, shortCircuitNow bool) {
		return (req1.Name == req2.Name), true
	})

	fmt.Println("Replaying...")
	// replayedResp2, err2 := replayer.Write(s2)
	replayedResp2, err2 := replayer.Write(Event{})
	assert.EqualError(t, err2, "No matching events.")
	assert.Equal(t, replayedResp2, Event{})

	replayedResp1, err1 := replayer.Write(s1)
	if err1 != nil {
		fmt.Println("Error! ", err1)
	} else {
		fmt.Println(replayedResp1)
	}
	// TODO: fix this
	// replayedResp1 = replayedResp1.(Event)
	// replayedResp2 = replayedResp2.(Event)

	// assert.Equal(t, replayedResp1, r1)
	// assert.Equal(t, replayedResp2, r2)
}

func TestLastMatchingEvent(t *testing.T) {
	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	r1 := Event{Name: "Response 1", Id: 23}
	fmt.Printf("%+v | %+v\n", s1, r1)
	recorder.Write(s1, r1)

	s2 := Event{Name: "Event 1", Id: 46}
	r2 := Event{Name: "Response 2", Id: 46}
	fmt.Printf("%+v | %+v\n", s2, r2)
	recorder.Write(s2, r2)

	s3 := Event{Name: "Event 3", Id: 52}
	r3 := Event{Name: "Response 3", Id: 52}
	fmt.Printf("%+v | %+v\n", s3, r3)
	recorder.Write(s3, r3)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	replayer, err := NewReplayer(filepath, func(req1 Event, req2 Event) (accept bool, shortCircuitNow bool) {
		return (req1.Name == req2.Name), false // false to return last match
	})

	fmt.Println("Replaying...")
	fmt.Println("Should return last matching event to \"Event 1\"")
	respA, errA := replayer.Write(s1)
	check(errA)
	assert.Equal(t, respA.Name, r2.Name)
	assert.Equal(t, respA.Id, r2.Id)

	fmt.Println("Should match the single \"Event 3\"")
	respB, errB := replayer.Write(s3)
	check(errB)
	assert.Equal(t, respB.Name, r3.Name)
	assert.Equal(t, respB.Id, r3.Id)

	fmt.Println("Should return first matching event to \"Event 1\" - since the last one was removed")
	respC, errC := replayer.Write(s1)
	check(errC)
	assert.Equal(t, respC.Name, r1.Name)
	assert.Equal(t, respC.Id, r1.Id)
	// TODO: fix this
	// replayedResp1 = replayedResp1.(Event)
	// replayedResp2 = replayedResp2.(Event)

	// assert.Equal(t, replayedResp1, r1)
	// assert.Equal(t, replayedResp2, r2)
}
