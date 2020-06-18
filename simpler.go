package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type Event struct {
	Name string
	Id   int
}

type RequestComparator func(req1 interface{}, req2 interface{}) (accept bool, shortCircuitNow bool)

type VcrRecorder struct {
	fileHandle os.File
}

func NewRecorder(filepath string) (recorder *VcrRecorder, err error) {
	recorder = &VcrRecorder{}

	// open a cassette file for writing
	fileHandle, err := os.Create(filepath)

	if err != nil {
		return nil, err
	}

	recorder.fileHandle = *fileHandle

	return recorder, nil
}

// takes a generic struct
func (recorder *VcrRecorder) Write(req interface{}, resp interface{}) error {
	bytes, err := json.Marshal(Pair{req, resp})

	if err != nil {
		return err
	}

	_, err = recorder.fileHandle.Write(bytes)
	recorder.fileHandle.Write([]byte("\n"))

	return err
}

func (recorder *VcrRecorder) Close() error {
	return recorder.fileHandle.Close()
}

type VcrReplayer struct {
	fileHandle   *os.File
	events       []Pair // it'd be *real* nice to use a generic type here. eg: Pair<T, K>
	historyIndex int
	comparator   RequestComparator
}

func NewReplayer(filepath string, comparator RequestComparator) (replayer VcrReplayer, err error) {

	replayer = VcrReplayer{}
	replayer.historyIndex = 0
	replayer.comparator = comparator

	file, err := os.Open(filepath)
	if err != nil {
		return VcrReplayer{}, err
	}

	replayer.fileHandle = file

	scanner := bufio.NewScanner(file)
	fmt.Println("In NewReplayer")

	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	replayer.events = make([]Pair, lineCount)
	idx := 0

	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		json.Unmarshal(bytes, &replayer.events[idx])
		idx++
	}

	return replayer, nil

}

func (replayer *VcrReplayer) Write(req Event) (resp interface{}, err error) {
	if len(replayer.events) == 0 {
		return Event{}, errors.New("Nothing left in cassette to replay.")
	}

	var lastAccepted interface{}
	acceptedIdx := -1

	for idx, val := range replayer.events {
		accept, shortCircuit := replayer.comparator(val.First, req)

		if accept {
			lastAccepted = val.Second
			acceptedIdx = idx
		}

		if shortCircuit {
			break
		}
	}

	if acceptedIdx != -1 {
		// remove the matched event
		replayer.events = append(replayer.events[:acceptedIdx], replayer.events[acceptedIdx+1:]...)
		return lastAccepted, nil
	}

	return Event{}, errors.New("No matching events.")
}

func (replayer *VcrReplayer) Close() {

}

type Pair struct {
	First  interface{}
	Second interface{}
}

type BetterEvent struct {
	Name string
	Type string
	Id   int
}

// func testBaseRecorder2() {
// 	filepath := "test.txt"
// 	recorder, err := NewRecorder(filepath)

// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("Recording...")

// 	s1 := Event{Name: "Event 1", Id: 23}
// 	r1 := Event{Name: "Response 1", Id: 23}
// 	fmt.Printf("%s | %s\n", s1, r1)
// 	recorder.Write(s1, r1)

// 	s2 := Event{Name: "Event 2", Id: 46}
// 	r2 := Event{Name: "Event 2", Id: 46}
// 	fmt.Printf("%s | %s\n", s2, r2)
// 	recorder.Write(s2, r2)

// 	err = recorder.Close()

// 	if err != nil {
// 		panic(err)
// 	}

// 	replayer, err := NewReplayer(filepath)

// 	fmt.Println("Replaying...")
// 	fmt.Println(replayer.Write(s2))
// 	fmt.Println(replayer.Write(s1))
// }

func main() {
}

/*
	Notes 5pm
	- could we do configurable matching?
		func compare(req1, req2) bool {...}
	- what should happen if you do something that is different than is recorded?
	- can we not impose the sequential constraint? -- aka otherways of advancing/matching
		- where we can re-order and still work (eg: match names) <<<<<< write a test for this
	- ^^^
	- have the structs always have a comprable / idetnity function

	if we can support all of the below behaviors by taking in a matchign/comrpable function and applying it in some way... then its a good sign for our abstraction
	- always return the responses in sequence
	- return responses in sequence so long as they "match", if they don't match, throw an error
	- take requests in, and return the first response that "matches", if it exists, throw an error
	- take requests in, and return the last response that "matches", if it exists, throw an error

	also think about templating with an eye towards dealing with the fake id issues we heard about

	make use of automated test cases that run automatically



*/

// Test: VCRRecoder VCRReplayer
/*

try to write a test case

// recorder.Write('a' -> 'A')
// recorder.Write('b')
// recorder.Write('c')
// buf := recorder.Close()
//
// replayer.Open(buf)
// resp := replayer.Write('a')
//



*/
