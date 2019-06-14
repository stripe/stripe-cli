package useragent

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"runtime"

	"github.com/stripe/stripe-cli/version"
)

//
// Public functions
//

// GetEncodedStripeUserAgent returns the string to be used as the value for
// the `X-Stripe-Client-User-Agent` HTTP header.
func GetEncodedStripeUserAgent() string {
	return encodedStripeUserAgent
}

// GetEncodedUserAgent returns the string to be used as the value for
// the `User-Agent` HTTP header.
func GetEncodedUserAgent() string {
	return encodedUserAgent
}

//
// Private constants
//

const (
	// UnknownPlatform is the string returned as the system name if we couldn't
	// get one from `uname`.
	unknownPlatform string = "unknown platform"
)

//
// Private types
//

// stripeClientUserAgent contains information about the current runtime which
// is serialized and sent in the `X-Stripe-Client-User-Agent` as additional
// debugging information.
type stripeClientUserAgent struct {
	Name      string `json:"name"`
	OS        string `json:"os"`
	Publisher string `json:"publisher"`
	Uname     string `json:"uname"`
	Version   string `json:"version"`
}

//
// Private variables
//

var encodedStripeUserAgent string
var encodedUserAgent string

//
// Private functions
//

// getUname tries to get a uname from the system, but not that hard. It tries
// to execute `uname -a`, but swallows any errors in case that didn't work
// (i.e. non-Unix non-Mac system or some other reason).
func getUname() string {
	path, err := exec.LookPath("uname")
	if err != nil {
		return unknownPlatform
	}

	cmd := exec.Command(path, "-a") // #nosec G204
	var out bytes.Buffer
	cmd.Stderr = nil // goes to os.DevNull
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return unknownPlatform
	}

	return out.String()
}

func init() {
	initUserAgent()
}

func initUserAgent() {
	encodedUserAgent = "Stripe/v1 stripe-cli/" + version.Version

	stripeUserAgent := &stripeClientUserAgent{
		Name:      "stripe-cli",
		Version:   version.Version,
		Publisher: "stripe",
		OS:        runtime.GOOS,
		Uname:     getUname(),
	}
	marshaled, err := json.Marshal(stripeUserAgent)
	// Encoding this struct should never be a problem, so we're okay to panic
	// in case it is for some reason.
	if err != nil {
		panic(err)
	}
	encodedStripeUserAgent = string(marshaled)
}
