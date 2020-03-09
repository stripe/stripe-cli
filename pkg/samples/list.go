package samples

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// SampleData stores the information needed for Stripe Samples to operate in
// the CLI
type SampleData struct {
	Name        string `json:"name"`
	URL         string `json:"URL"`
	Description string `json:"description"`
}

// SampleList is used to unmarshal the samples array from the JSON response
type SampleList struct {
	Samples []SampleData `json:"samples"`
}

func getJSON(url string, target interface{}) error {
	spinner := ansi.StartSpinner("Loading...", os.Stdout)
	r, err := http.Get(url)
	if err != nil {
		ansi.StopSpinner(spinner, "Error: Check your network connection and retry.", os.Stdout)
		return err
	}
	defer r.Body.Close()
	ansi.StopSpinner(spinner, "", os.Stdout)
	return json.NewDecoder(r.Body).Decode(target)
}

func getSamples() []SampleData {
	// Fetch samples from gh-pages
	sampleList := SampleList{}
	getJSON("https://thorsten-stripe.github.io/stripe-cli-gh-pages/samples.json", &sampleList)
	return sampleList.Samples
}

// List contains a mapping of Stripe Samples that we want to be available in the
// CLI to some of their metadata.
// TODO: what do we want to name these for it to be easier for users to select?
// TODO: should we group them by products for easier exploring?
var List = map[string]*SampleData{}

// InitSampleList fetches the samples from a remote JSON file and sets up the
// List mapping.
func InitSampleList() {
	sampleList := getSamples()
	for i, sample := range sampleList {
		List[sample.Name] = &sampleList[i]
	}
}

// BoldName returns an ansi bold string for the name
func (sd *SampleData) BoldName() string {
	return ansi.Bold(sd.Name)
}

// GitRepo returns a string of the repo with the .git prefix
func (sd *SampleData) GitRepo() string {
	return fmt.Sprintf("%s.git", sd.URL)
}

// Names returns a list of all the sample's names
func Names() []string {
	keys := make([]string, 0, len(List))
	for k := range List {
		keys = append(keys, k)
	}

	return keys
}
