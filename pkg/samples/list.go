package samples

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

const sampleListGithubURL = "https://github.com/stripe-samples/samples-list.git"

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

// BoldName returns an ansi bold string for the name
func (sd *SampleData) BoldName() string {
	return ansi.Bold(sd.Name)
}

// GitRepo returns a string of the repo with the .git prefix
func (sd *SampleData) GitRepo() string {
	return fmt.Sprintf("%s.git", sd.URL)
}

// Names returns a list of all the sample's names
func Names(list map[string]*SampleData) []string {
	keys := make([]string, 0, len(list))
	for k := range list {
		keys = append(keys, k)
	}

	return keys
}

// Returns a list that contains a mapping of Stripe Samples
var list = map[string]*SampleData{}

func (s *Samples) getFromCacheOrGithub(noNetwork bool) error {
	listPath, err := s.appCacheFolder("samples-list")
	if err != nil {
		return err
	}

	if _, err := s.Fs.Stat(listPath); os.IsNotExist(err) {
		err = s.Git.Clone(listPath, sampleListGithubURL)
		if err != nil {
			return err
		}
	} else if !noNetwork {
		err := s.Git.Pull(listPath)
		if err != nil {
			if err != nil {
				switch e := err.Error(); e {
				case git.NoErrAlreadyUpToDate.Error():
					// Repo is already up to date. This isn't a program
					// error to continue as normal
					break
				default:
					return err
				}
			}
		}
	}

	file, err := afero.ReadFile(s.Fs, filepath.Join(listPath, "samples.json"))
	if err != nil {
		return err
	}

	var allSamples SampleList

	err = json.Unmarshal(file, &allSamples)
	if err != nil {
		return err
	}

	for i, sample := range allSamples.Samples {
		list[sample.Name] = &allSamples.Samples[i]
	}

	return nil
}

// GetSamples returns a list that contains a mapping of Stripe Samples that
// we want to be available in the CLI to some of their metadata.
// TODO: what do we want to name these for it to be easier for users to select?
// TODO: should we group them by products for easier exploring?
func (s *Samples) GetSamples(mode string) map[string]*SampleData {
	spinner := ansi.StartNewSpinner("Loading...", os.Stdout)

	if len(list) != 0 {
		ansi.StopSpinner(spinner, "", os.Stdout)
		return list
	}

	// Reduce the number of requests to GitHub
	var noNetwork bool
	switch mode {
	case "list":
		noNetwork = false
	case "create":
		noNetwork = true
	default:
		noNetwork = false
	}

	// Get the samples from the cache or GitHub
	err := s.getFromCacheOrGithub(noNetwork)
	if err != nil {
		ansi.StopSpinner(spinner, "Error: please check your internet connection and try again!", os.Stdout)
	}

	ansi.StopSpinner(spinner, "", os.Stdout)
	return list
}
