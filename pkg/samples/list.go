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

type SampleList struct {
	Samples []SampleData `json:"samples"`
}

func getJson(url string, target interface{}) error {
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

func GetSamples() []SampleData {
	// Fetch samples from gh-pages
	sampleList := SampleList{}
	getJson("https://thorsten-stripe.github.io/stripe-cli-gh-pages/samples.json", &sampleList)
	return sampleList.Samples
}

var List = map[string]*SampleData{}

func InitSampleList() {
	sampleList := GetSamples()
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
