package fixtures

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imdario/mergo"
	"github.com/joho/godotenv"
	"github.com/spf13/afero"
	"github.com/tidwall/gjson"

	"github.com/stripe/stripe-cli/pkg/requests"
)

// SupportedVersions is the version number of the fixture template the CLI supports
const SupportedVersions = 0

type metaFixture struct {
	Version         int  `json:"template_version"`
	ExcludeMetadata bool `json:"exclude_metadata"`
}

type fixtureFile struct {
	Meta     metaFixture       `json:"_meta"`
	Fixtures []fixture         `json:"fixtures"`
	Env      map[string]string `json:"env"`
}

type fixture struct {
	Name              string                 `json:"name"`
	ExpectedErrorType string                 `json:"expected_error_type"`
	Path              string                 `json:"path"`
	Method            string                 `json:"method"`
	Params            map[string]interface{} `json:"params"`
}

type fixtureQuery struct {
	Name         string
	Query        string
	DefaultValue string
}

// Fixture contains a mapping of an individual fixtures responses for querying
type Fixture struct {
	Fs            afero.Fs
	APIKey        string
	StripeAccount string
	Skip          []string
	BaseURL       string
	responses     map[string]gjson.Result
	fixture       fixtureFile
}

// NewFixture creates a to later run steps for populating test data
func NewFixture(fs afero.Fs, apiKey string, stripeAccount string, skip []string, baseURL string, file string) (*Fixture, error) {
	fxt := Fixture{
		Fs:            fs,
		APIKey:        apiKey,
		StripeAccount: stripeAccount,
		Skip:          skip,
		BaseURL:       baseURL,
		responses:     make(map[string]gjson.Result),
	}

	var filedata []byte

	var err error

	if _, ok := reverseMap()[file]; ok {
		f, err := triggers.Open(file)
		if err != nil {
			return nil, err
		}

		filedata, err = ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
	} else {
		filedata, err = afero.ReadFile(fxt.Fs, file)
		if err != nil {
			return nil, err
		}
	}

	err = json.Unmarshal(filedata, &fxt.fixture)
	if err != nil {
		return nil, err
	}

	// If the method is `post` and params are nil, initialize them
	// in case the user wants to add something dynamically using `Add`
	for i, data := range fxt.fixture.Fixtures {
		if data.Method == "post" && data.Params == nil {
			fxt.fixture.Fixtures[i].Params = make(map[string]interface{})
		}
	}

	if fxt.fixture.Meta.Version > SupportedVersions {
		return nil, fmt.Errorf("Fixture version not supported: %s", fmt.Sprint(fxt.fixture.Meta.Version))
	}

	return &fxt, nil
}

// Override forcefully overrides fields with existing data on a fixture
func (fxt *Fixture) Override(overrides []string) {
	data := buildRewrites(overrides, false)
	for _, f := range fxt.fixture.Fixtures {
		if _, ok := data[f.Name]; ok {
			if err := mergo.Merge(&f.Params, data[f.Name], mergo.WithOverride); err != nil {
				fmt.Println(err)
			}
		}
	}
}

// Add safely only adds any missing fields that do not already exist.
// If the field is already on the fixture, it does not get copied
// over. For that, `Override` should be used
func (fxt *Fixture) Add(additions []string) {
	data := buildRewrites(additions, false)
	for _, f := range fxt.fixture.Fixtures {
		if _, ok := data[f.Name]; ok {
			if err := mergo.Merge(&f.Params, data[f.Name]); err != nil {
				fmt.Println(err)
			}
		}
	}
}

// Remove todo implement - removes fields from the fixture
func (fxt *Fixture) Remove(removals []string) {
	data := buildRewrites(removals, true)
	for _, f := range fxt.fixture.Fixtures {
		if _, ok := data[f.Name]; ok {
			for remove, _ := range data[f.Name].(map[string]interface{}) {
				delete(f.Params, remove)
			}
		}
	}
}

// Execute takes the parsed fixture file and runs through all the requests
// defined to populate the user's account
func (fxt *Fixture) Execute() error {
	for _, data := range fxt.fixture.Fixtures {
		if isNameIn(data.Name, fxt.Skip) {
			fmt.Printf("Skipping fixture for: %s\n", data.Name)
			continue
		}

		fmt.Printf("Running fixture for: %s\n", data.Name)
		resp, err := fxt.makeRequest(data)
		if err != nil && !errWasExpected(err, data.ExpectedErrorType) {
			return err
		}

		fxt.responses[data.Name] = gjson.ParseBytes(resp)
	}

	return nil
}

func errWasExpected(err error, expectedErrorType string) bool {
	if rerr, ok := err.(requests.RequestError); ok {
		return rerr.ErrorType == expectedErrorType
	}
	return false
}

// UpdateEnv uses the results of the fixtures command just executed and
// updates a local .env with the resulting data
func (fxt *Fixture) UpdateEnv() error {
	if len(fxt.fixture.Env) > 0 {
		return fxt.updateEnv(fxt.fixture.Env)
	}

	return nil
}

func (fxt *Fixture) makeRequest(data fixture) ([]byte, error) {
	var rp requests.RequestParameters

	if data.Method == "post" && !fxt.fixture.Meta.ExcludeMetadata {
		now := time.Now().String()
		metadata := fmt.Sprintf("metadata[_created_by_fixture]=%s", now)
		rp.AppendData([]string{metadata})
	}

	req := requests.Base{
		Method:         strings.ToUpper(data.Method),
		SuppressOutput: true,
		APIBaseURL:     fxt.BaseURL,
		Parameters:     rp,
	}

	path := fxt.parsePath(data)

	return req.MakeRequest(fxt.APIKey, path, fxt.createParams(data.Params), true)
}

func (fxt *Fixture) createParams(params interface{}) *requests.RequestParameters {
	requestParams := requests.RequestParameters{}
	requestParams.AppendData(fxt.parseInterface(params))

	requestParams.SetStripeAccount(fxt.StripeAccount)

	return &requestParams
}

func (fxt *Fixture) updateEnv(env map[string]string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	envFile := filepath.Join(dir, ".env")

	exists, _ := afero.Exists(fxt.Fs, envFile)
	if !exists {
		// If there is no .env in the current directory, return and do nothing
		return nil
	}

	file, err := fxt.Fs.Open(envFile)
	if err != nil {
		return err
	}

	dotenv, err := godotenv.Parse(file)
	if err != nil {
		return err
	}

	for key, value := range env {
		dotenv[key] = fxt.parseQuery(value)
	}

	content, err := godotenv.Marshal(dotenv)
	if err != nil {
		return err
	}

	afero.WriteFile(fxt.Fs, envFile, []byte(content), os.ModePerm)

	return nil
}

// isNameIn will search if the current fixture is in the skip list
func isNameIn(name string, skip []string) bool {
	for _, skipName := range skip {
		if name == skipName {
			return true
		}
	}

	return false
}

// buildRewrites takes a slice of json queries and values then builds
// them into a map to later by merged. We work through the entire
// list at the same time because the user might pass in multiple
// changes for the same fixture.
//
// The query supported is <fixture_name>:path.to.field=value
func buildRewrites(changes []string, toRemove bool) map[string]interface{} {
	builtChanges := make(map[string]interface{})
	for _, change := range changes {
		changeSplit := strings.SplitN(change, "=", 2)
		path := changeSplit[0]

		// When removing a field there will be no value so we set a default
		// empty string or trying to get the split value from above
		var value string
		if !toRemove {
			value = changeSplit[1]
		}

		pathSplit := strings.SplitN(path, ":", 2)
		name := pathSplit[0]
		keys := pathSplit[1]

		keysSplit := strings.Split(keys, ".")

		keysReversed := reverse(keysSplit)
		key, keysReversed := pop(keysReversed)
		keyMap := make(map[string]interface{})
		keyMap[key] = value

		for _, key := range keysReversed {
			keyMap = map[string]interface{}{
				key: keyMap,
			}
		}
		_, ok := builtChanges[name]
		if ok {
			if err := mergo.Merge(builtChanges[name], keyMap); err != nil {
				fmt.Println(err)
			}
		} else {
			builtChanges[name] = keyMap
		}
	}

	return builtChanges
}

// pop returns the first item and the rest of the list minus the first item
// From: https://github.com/golang/go/wiki/SliceTricks#pop
func pop(list []string) (string, []string) {
	return list[len(list)-1], list[:len(list)-1]
}

// reverse reverses the list
// From: https://github.com/golang/go/wiki/SliceTricks#reversing
func reverse(list []string) []string {
	reversed := make([]string, len(list))
	copy(reversed, list)

	for i := len(reversed)/2 - 1; i >= 0; i-- {
		opp := len(reversed) - 1 - i
		reversed[i], reversed[opp] = reversed[opp], reversed[i]
	}

	return reversed
}
