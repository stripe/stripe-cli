package fixtures

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/joho/godotenv"
	"github.com/spf13/afero"
	"github.com/tidwall/gjson"

	"github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/parsers"
	"github.com/stripe/stripe-cli/pkg/requests"
)

// SupportedVersions is the version number of the fixture template the CLI supports
const SupportedVersions = 0

// MetaFixture contains fixture metadata
type MetaFixture struct {
	Version         int  `json:"template_version"`
	ExcludeMetadata bool `json:"exclude_metadata"`
}

// FixtureData contains the whole fixture file
type FixtureData struct {
	Meta     MetaFixture       `json:"_meta"`
	Requests []FixtureRequest  `json:"fixtures"`
	Env      map[string]string `json:"env"`
}

// FixtureRequest is the individual request payload
type FixtureRequest struct {
	Name              string                 `json:"name"`
	ExpectedErrorType string                 `json:"expected_error_type"`
	Path              string                 `json:"path"`
	Method            string                 `json:"method"`
	Params            map[string]interface{} `json:"params"`
	IdempotencyKey    string                 `json:"idempotency_key,omitempty"`
	Context           string                 `json:"context,omitempty"`
}

// Fixture contains a mapping of an individual fixtures responses for querying
type Fixture struct {
	Fs            afero.Fs
	APIKey        string
	StripeAccount string
	Skip          []string
	Overrides     map[string]interface{}
	Additions     map[string]interface{}
	Removals      map[string]interface{}
	BaseURL       string
	Responses     map[string]gjson.Result
	FixtureData   FixtureData
}

// NewFixtureFromFile creates a to later run steps for populating test data
func NewFixtureFromFile(fs afero.Fs, apiKey, stripeAccount, baseURL, file string, skip, override, add, remove []string, edit bool) (*Fixture, error) {
	fxt := Fixture{
		Fs:            fs,
		APIKey:        apiKey,
		StripeAccount: stripeAccount,
		BaseURL:       baseURL,
		Responses:     make(map[string]gjson.Result),
	}

	var filedata []byte
	var err error

	if _, ok := reverseMap()[file]; ok {
		f, err := triggers.Open(file)
		if err != nil {
			return nil, err
		}

		filedata, err = io.ReadAll(f)
		if err != nil {
			return nil, err
		}
	} else {
		filedata, err = afero.ReadFile(fxt.Fs, file)
		if err != nil {
			return nil, err
		}
	}

	// Customize fixture data

	if edit {
		filedata, err = fxt.Edit(file, filedata)
		if err != nil {
			return nil, err
		}
	}

	err = json.Unmarshal(filedata, &fxt.FixtureData)
	if err != nil {
		return nil, err
	}

	if len(override) > 0 || len(add) > 0 || len(remove) > 0 || len(skip) > 0 {
		if edit {
			fmt.Println("Warning: --edit cannot be used with --add, --remove, --override, or --skip. Skipping those flags...")
		} else {
			if err := fxt.Override(override); err != nil {
				return nil, err
			}
			if err := fxt.Add(add); err != nil {
				return nil, err
			}
			if err := fxt.Remove(remove); err != nil {
				return nil, err
			}
			fxt.Skip = skip
		}
	}

	if fxt.FixtureData.Meta.Version > SupportedVersions {
		return nil, fmt.Errorf("Fixture version not supported: %s", fmt.Sprint(fxt.FixtureData.Meta.Version))
	}

	return &fxt, nil
}

// NewFixtureFromRawString creates fixtures from user inputted string
func NewFixtureFromRawString(fs afero.Fs, apiKey, stripeAccount, baseURL, raw string) (*Fixture, error) {
	fxt := Fixture{
		Fs:            fs,
		APIKey:        apiKey,
		StripeAccount: stripeAccount,
		Skip:          []string{},
		BaseURL:       baseURL,
		Responses:     make(map[string]gjson.Result),
	}

	err := json.Unmarshal([]byte(raw), &fxt.FixtureData)
	if err != nil {
		return nil, err
	}

	if fxt.FixtureData.Meta.Version > SupportedVersions {
		return nil, fmt.Errorf("Fixture version not supported: %s", fmt.Sprint(fxt.FixtureData.Meta.Version))
	}

	return &fxt, nil
}

// GetFixtureFileContent returns the file content of the given fixture file name
func (fxt *Fixture) GetFixtureFileContent() string {
	data, err := json.MarshalIndent(fxt.FixtureData, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

type fixtureRewriteError struct {
	operation string // override/add/remove
	err       error
	fixture   *Fixture
}

func (e fixtureRewriteError) Unwrap() error {
	return e.err
}

func (e fixtureRewriteError) example() string {
	example := fmt.Sprintf("--%s fixtureName:path.to.param", e.operation)
	if e.operation != "remove" {
		example += "=value"
	}
	return example
}

func (e fixtureRewriteError) Error() string {
	var nameError missingFixtureNameError
	if errors.As(e.err, &nameError) {
		fixtureNames := []string{}
		for _, fixture := range e.fixture.FixtureData.Requests {
			fixtureNames = append(fixtureNames, fixture.Name)
		}

		return fmt.Sprintf(
			"Invalid value for %s flag (%s). The %s flag requires the name of the fixture to apply (%s).\n\nValid fixture names are: %v",
			e.operation, nameError.value, e.operation, e.example(), fixtureNames,
		)
	}

	var valueError missingRewriteValueError
	if errors.As(e.err, &valueError) {
		return fmt.Sprintf(
			"Invalid value for %s flag (%s). The %s flag requires a value to set (%s).",
			e.operation, valueError.value, e.operation, e.example(),
		)
	}

	return fmt.Sprintf("Invalid value for %s flag: %s", e.operation, e.err)
}

// Override forcefully overrides fields with existing data on a fixture
func (fxt *Fixture) Override(overrides []string) error {
	data, err := buildRewrites(overrides, false)
	if err != nil {
		return fixtureRewriteError{operation: "override", err: err, fixture: fxt}
	}
	for _, f := range fxt.FixtureData.Requests {
		if _, ok := data[f.Name]; ok {
			if err := mergo.Merge(&f.Params, data[f.Name], mergo.WithOverride); err != nil {
				fmt.Println(err)
			}
		}
	}

	return nil
}

// Add safely only adds any missing fields that do not already exist.
// If the field is already on the fixture, it does not get copied
// over. For that, `Override` should be used
func (fxt *Fixture) Add(additions []string) error {
	// If the params is empty, initialize it before merging with added data
	for i, data := range fxt.FixtureData.Requests {
		if data.Method == "post" && data.Params == nil {
			fxt.FixtureData.Requests[i].Params = make(map[string]interface{})
		}
	}

	data, err := buildRewrites(additions, false)
	if err != nil {
		return fixtureRewriteError{operation: "add", err: err, fixture: fxt}
	}
	for _, f := range fxt.FixtureData.Requests {
		if _, ok := data[f.Name]; ok {
			if err := mergo.Merge(&f.Params, data[f.Name]); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

// Remove removes fields from the fixture
func (fxt *Fixture) Remove(removals []string) error {
	data, err := buildRewrites(removals, true)
	if err != nil {
		return fixtureRewriteError{operation: "remove", err: err, fixture: fxt}
	}
	for _, f := range fxt.FixtureData.Requests {
		if _, ok := data[f.Name]; ok {
			for remove := range data[f.Name].(map[string]interface{}) {
				delete(f.Params, remove)
			}
		}
	}
	return nil
}

// Edit opens the fixture in the git's default IDE to edit directly
func (fxt *Fixture) Edit(path string, filedata []byte) ([]byte, error) {
	return Edit(path, filedata)
}

// Edit is separated into a var so we can mock this in fixtures_test
var Edit = func(path string, filedata []byte) ([]byte, error) {
	filename := getFixtureFilenameWithWildcard(path)
	editor, err := git.NewTemporaryFileEditor(filename, filedata)
	if err != nil {
		return nil, err
	}

	return editor.EditContent()
}

func getFixtureFilenameWithWildcard(path string) string {
	pathComponents := strings.Split(path, "/")
	fixtureName := strings.Split(pathComponents[len(pathComponents)-1], ".")
	// Add a wildcard that is replaced by a random string when passing this filename to os.CreateTemp
	return strings.Join(fixtureName[0:len(fixtureName)-1], ".") + ".*." + fixtureName[len(fixtureName)-1]
}

// Execute takes the parsed fixture file and runs through all the requests
// defined to populate the user's account
func (fxt *Fixture) Execute(ctx context.Context, apiVersion string) ([]string, error) {
	requestNames := make([]string, len(fxt.FixtureData.Requests))
	for i, data := range fxt.FixtureData.Requests {
		if isNameIn(data.Name, fxt.Skip) {
			fmt.Printf("Skipping fixture for: %s\n", data.Name)
			continue
		}

		fmt.Printf("Setting up fixture for: %s\n", data.Name)
		requestNames[i] = data.Name

		fmt.Printf("Running fixture for: %s\n", data.Name)
		resp, err := fxt.makeRequest(ctx, data, apiVersion)
		if err != nil && !errWasExpected(err, data.ExpectedErrorType) {
			return nil, err
		}

		fxt.Responses[data.Name] = gjson.ParseBytes(resp)
	}
	return requestNames, nil
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
	if len(fxt.FixtureData.Env) > 0 {
		return fxt.updateEnv(fxt.FixtureData.Env)
	}

	return nil
}

func (fxt *Fixture) makeRequest(ctx context.Context, data FixtureRequest, apiVersion string) ([]byte, error) {
	req := requests.Base{
		Method:         strings.ToUpper(data.Method),
		SuppressOutput: true,
		APIBaseURL:     fxt.BaseURL,
	}

	path, err := parsers.ParsePath(data.Path, fxt.Responses)

	if err != nil {
		return make([]byte, 0), err
	}

	params, err := fxt.createParams(data.Params, apiVersion)
	if err != nil {
		return make([]byte, 0), err
	}

	if data.IdempotencyKey != "" {
		idempotencyKey, err := parsers.ParseQuery(data.IdempotencyKey, fxt.Responses)
		if err != nil {
			return nil, fmt.Errorf("error parsing idempotency_key field: %w", err)
		}
		params.SetIdempotency(idempotencyKey)
	}

	return req.MakeRequest(ctx, fxt.APIKey, path, params, true)
}

func (fxt *Fixture) createParams(params interface{}, apiVersion string) (*requests.RequestParameters, error) {
	requestParams := requests.RequestParameters{}
	parsed, err := parsers.ParseInterface(params, fxt.Responses)
	if err != nil {
		return &requestParams, err
	}
	requestParams.AppendData(parsed)

	requestParams.SetStripeAccount(fxt.StripeAccount)

	if apiVersion != "" {
		requestParams.SetVersion(apiVersion)
	}

	return &requestParams, nil
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
		parsed, err := parsers.ParseQuery(value, fxt.Responses)
		if err != nil {
			return err
		}

		dotenv[key] = parsed
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

type missingFixtureNameError struct {
	value string
}

func (e missingFixtureNameError) Error() string {
	return fmt.Sprintf("rewrite rule %s missing fixture name", e.value)
}

type missingRewriteValueError struct {
	value string
}

func (e missingRewriteValueError) Error() string {
	return fmt.Sprintf("rewrite rule %s missing value", e.value)
}

// buildRewrites takes a slice of json queries and values then builds
// them into a map to later be merged. We work through the entire
// list at the same time because the user might pass in multiple
// changes for the same fixture.
//
// The query supported is <fixture_name>:path.to.field=value
func buildRewrites(changes []string, toRemove bool) (map[string]interface{}, error) {
	builtChanges := make(map[string]interface{})
	for _, change := range changes {
		if change == "" {
			continue
		}
		changeSplit := strings.SplitN(change, "=", 2)
		path := changeSplit[0]

		// When removing a field there will be no value so we set a default
		// empty string or trying to get the split value from above
		var value string
		if !toRemove {
			if len(changeSplit) == 1 {
				return nil, missingRewriteValueError{value: change}
			}
			value = changeSplit[1]
		}

		pathSplit := strings.SplitN(path, ":", 2)
		if len(pathSplit) == 1 {
			return nil, missingFixtureNameError{value: change}
		}
		name := pathSplit[0]
		keys := pathSplit[1]

		keysSplit := strings.Split(keys, ".")

		field, paths := pop(keysSplit)
		keyMap := make(map[string]interface{})
		keyMap[field] = value

		keysReversed := reverse(paths)
		for _, key := range keysReversed {
			keyMap = map[string]interface{}{
				key: keyMap,
			}
		}
		_, ok := builtChanges[name]
		if ok {
			if err := mergo.Merge(&keyMap, builtChanges[name]); err != nil {
				fmt.Println(err)
			}
		}

		builtChanges[name] = keyMap
	}

	return builtChanges, nil
}

// pop returns the last item and the rest of the list minus the last item
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
