package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

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
func (recorder *VcrRecorder) Write(event interface{}) error {
	bytes, err := json.Marshal(event)

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
	events       []interface{}
	historyIndex int
}

func NewReplayer(filepath string) (replayer VcrReplayer, err error) {

	replayer = VcrReplayer{}
	replayer.historyIndex = 0

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

	replayer.events = make([]interface{}, lineCount)
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

func (replayer *VcrReplayer) Write(event interface{}) interface{} {
	if replayer.historyIndex == len(replayer.events) {
		return nil
	}
	response := replayer.events[replayer.historyIndex]
	replayer.historyIndex++
	return response
}

func (replayer *VcrReplayer) Close() {

}

type Event struct {
	Name string
	Id   int
}

func testBaseRecorder() {
	filepath := "test.txt"
	recorder, err := NewRecorder(filepath)

	if err != nil {
		panic(err)
	}

	fmt.Println("Recording...")

	s1 := Event{Name: "Event 1", Id: 23}
	fmt.Println(s1)
	recorder.Write(s1)

	s2 := Event{Name: "Event 2", Id: 46}
	fmt.Println(s2)
	recorder.Write(s2)

	err = recorder.Close()

	if err != nil {
		panic(err)
	}

	replayer, err := NewReplayer(filepath)

	fmt.Println("Replaying...")
	fmt.Println(replayer.Write(Event{"placeholder", 2}))
	fmt.Println(replayer.Write(Event{"placeholder", 2}))

}

func main() {
	testBaseRecorder()
}

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
