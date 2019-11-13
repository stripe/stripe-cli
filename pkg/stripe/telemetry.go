package stripe

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

//
// Public types
//

// CLITelemetry is the structure that holds telemetry data sent to Stripe in
// API requests.
type CLITelemetry struct {
	CommandPath       string `json:"command_path"`
	DeviceName        string `json:"device_name"`
	GeneratedResource bool   `json:"generated_resource"`
}

// SetCommandContext sets the telemetry values for the command being executed.
func (t *CLITelemetry) SetCommandContext(cmd *cobra.Command) {
	t.CommandPath = cmd.CommandPath()
	t.GeneratedResource = false

	for _, value := range cmd.Annotations {
		// Generated commands have an annotation called "operation", we can
		// search for that to let us know it's generated
		if value == "operation" {
			t.GeneratedResource = true
		}
	}
}

// SetDeviceName puts the device name into telemetry
func (t *CLITelemetry) SetDeviceName(deviceName string) {
	t.DeviceName = deviceName
}

//
// Public functions
//

// GetTelemetryInstance returns the CLITelemetry instance (initializing it
// first if necessary).
func GetTelemetryInstance() *CLITelemetry {
	once.Do(func() {
		instance = &CLITelemetry{}
	})

	return instance
}

//
// Private variables
//

var instance *CLITelemetry
var once sync.Once

//
// Private functions
//

func getTelemetryHeader() (string, error) {
	telemetry := GetTelemetryInstance()
	b, err := json.Marshal(telemetry)

	if err != nil {
		return "", err
	}

	return string(b), nil
}

// telemetryOptedOut returns true if the user has opted out of telemetry,
// false otherwise.
func telemetryOptedOut(optoutVar string) bool {
	optoutVar = strings.ToLower(optoutVar)

	return optoutVar == "1" || optoutVar == "true"
}
