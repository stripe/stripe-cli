package main

// type Cassette struct {
// 	recordMode bool

// }


// func NewEmptyCassette(filepath) *Cassette {

// } 

// func ReadExistingCassette(filepath) *Cassette {

// }

// // TODO: this part of the interface is where the specifics of how we're implementing recording/replaying start to matter

// // --- Replay mode interface

// // If the request matches the history, return the associated response. Otherwise return an descriptive error.
// func AdvanceReplay(request *http.Request) (resp *http.Response, err error) {
// 	if recordMode {
// 		return nil, errors.New("Can't AdvanceReplay while in Record mode!")
// 	}

// 	// potentially meaty logic thats related to decisions about how we implement matching up replay requests to recorded history
// }

// // --- Record mode interface

// func RecordEvent(request *http.Request, response *http.Response) error{
// 	if !recordMode {
// 		return errors.New("Can't record while in replay mode!")
// 	}
// }
