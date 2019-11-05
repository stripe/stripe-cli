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
	CommandPath string `json:"command_path"`
}

// SetCommandContext sets the telemetry values for the command being executed.
func (t *CLITelemetry) SetCommandContext(cmd *cobra.Command) {
	t.CommandPath = cmd.CommandPath()
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
