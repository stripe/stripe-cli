package status

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// Response contains the structure of system statuses from Stripe
type Response struct {
	Statuses    statuses `json:"statuses"`
	LargeStatus string   `json:"largestatus"`
	Message     string   `json:"message"`
	Time        string   `json:"time"`
}

type statuses struct {
	API        string `json:"api"`
	Dashboard  string `json:"dashboard"`
	Stripejs   string `json:"stripejs"`
	Checkoutjs string `json:"checkoutjs"`
	// These two are not used and may not be reliable
	Webhooks string `json:"webhooks"`
	Emails   string `json:"emails"`
}

// GetStatus makes a request to the Stripe status site and returns all the
// current system statuses
func GetStatus() (Response, error) {
	var status Response

	resp, err := http.Get("https://status.stripe.com/current")
	if err != nil {
		return status, err
	}

	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return status, err
	}

	json.Unmarshal(respBytes, &status)

	return status, nil
}

func (r *Response) getMap(verbose bool) map[string]interface{} {
	responseObject := map[string]interface{}{
		"status":  r.LargeStatus,
		"message": r.Message,
		"time":    r.Time,
	}

	if verbose {
		responseObject["statuses"] = map[string]string{
			"api":        r.Statuses.API,
			"dashboard":  r.Statuses.Dashboard,
			"stripejs":   r.Statuses.Stripejs,
			"checkoutjs": r.Statuses.Checkoutjs,
		}
	}

	return responseObject
}

// FormattedMessage returns a properly structured API status response
// in either a json structure or a templated plain text output, conditionally
// populated with extra data depending on verbosity
func (r *Response) FormattedMessage(format string, verbose bool) (string, error) {
	statusData := r.getMap(verbose)

	if format == "json" {
		data, err := json.MarshalIndent(statusData, "", "  ")
		if err != nil {
			return "", err
		}

		return string(data), nil
	}

	statusString := fmt.Sprintf(`%s %s{{if .statuses}}
%s API
%s Dashboard
%s Stripe.js
%s Checkout.js{{end}}
As of: %s`,
		emojifiedStatus(r.LargeStatus),
		ansi.Bold(r.Message),
		emojifiedStatus(r.Statuses.API),
		emojifiedStatus(r.Statuses.Dashboard),
		emojifiedStatus(r.Statuses.Stripejs),
		emojifiedStatus(r.Statuses.Checkoutjs),
		ansi.Italic(r.Time),
	)

	tmpl, err := template.New("status").Parse(statusString)
	if err != nil {
		return "", err
	}

	var output bytes.Buffer

	err = tmpl.Execute(&output, statusData)
	if err != nil {
		return "", nil
	}

	return output.String(), nil
}

func emojifiedStatus(status string) string {
	color := ansi.Color(os.Stdout)

	switch status {
	case "up":
		return color.Green("✔").String()
	case "degraded":
		return color.Yellow("!").String()
	case "down":
		return color.Red("✘").String()
	}

	// To avoid potentially confusing users, if the status does not fit one of
	// the three above, let's return nothing
	return ""
}
