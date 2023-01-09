package samples

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4"

	"github.com/stripe/stripe-cli/pkg/ansi"
	gitpkg "github.com/stripe/stripe-cli/pkg/git"
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

// SampleLister gets the list of valid stripe samples. It is used both in
// `stripe samples list` to show the users what they can do, and in
// `stripe samples create <name>` in order to look up the repo url corresponding
// to <name>
type SampleLister interface {
	ListSamples(mode string) (map[string]*SampleData, error)
}

type cachedGithubSampleLister struct {
	// Used for .Fs and .Git
	s *SampleManager

	// URL like https://github.com/stripe-samples/samples-list.git
	// expected to contain a repository with a "samples.json" file at the root
	// with the expected format/contents
	sampleListGithubURL string

	// Place on the user's filesystem where a cached copy of the repo designated by
	// https://github.com/stripe-samples/samples-list.git will be stored
	cacheFolder string

	// In-memory cache of the sample data contained in samples.json
	result map[string]*SampleData
}

func newCachedGithubSampleLister(s *SampleManager, sampleListGithubURL string, cacheFolder string) SampleLister {
	return &cachedGithubSampleLister{
		s:                   s,
		sampleListGithubURL: sampleListGithubURL,
		cacheFolder:         cacheFolder,
	}
}

func (l *cachedGithubSampleLister) getFromCacheOrGithub(noNetwork bool) (map[string]*SampleData, error) {
	if _, err := l.s.Fs.Stat(l.cacheFolder); os.IsNotExist(err) {
		err = l.s.Git.Clone(l.cacheFolder, l.sampleListGithubURL)
		if err != nil {
			return nil, err
		}
	} else if !noNetwork {
		err := l.s.Git.Pull(l.cacheFolder)
		if err != nil {
			if err != nil {
				switch e := err.Error(); e {
				case git.NoErrAlreadyUpToDate.Error():
					// Repo is already up to date. This isn't a program
					// error to continue as normal
					break
				default:
					return nil, err
				}
			}
		}
	}

	file, err := afero.ReadFile(l.s.Fs, filepath.Join(l.cacheFolder, "samples.json"))
	if err != nil {
		return nil, err
	}

	var allSamples SampleList

	err = json.Unmarshal(file, &allSamples)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*SampleData)
	for i, sample := range allSamples.Samples {
		ret[sample.Name] = &allSamples.Samples[i]
	}

	return ret, nil
}

func (l *cachedGithubSampleLister) ListSamples(mode string) (map[string]*SampleData, error) {
	if len(l.result) != 0 {
		return l.result, nil
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

	ret, err := l.getFromCacheOrGithub(noNetwork)
	if err != nil {
		return nil, err
	}
	l.result = ret
	return ret, nil
}

// GetSamples returns a list that contains a mapping of Stripe Samples that
// we want to be available in the CLI to some of their metadata.
// TODO: what do we want to name these for it to be easier for users to select?
// TODO: should we group them by products for easier exploring?
func GetSamples(mode string) (map[string]*SampleData, error) {
	sampleManager := SampleManager{
		Fs:  afero.NewOsFs(),
		Git: gitpkg.Operations{},
	}

	cacheFolder, err := sampleManager.appCacheFolder("samples-list")
	if err != nil {
		return nil, err
	}
	sampleManager.SampleLister = newCachedGithubSampleLister(&sampleManager, sampleListGithubURL, cacheFolder)

	return sampleManager.SampleLister.ListSamples(mode)
}
