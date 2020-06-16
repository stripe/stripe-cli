//go:generate go run -tags=dev vfsgen.go

package fixtures

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/afero"
	"github.com/thedevsaddam/gojsonq"

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
	Name   string      `json:"name"`
	Path   string      `json:"path"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
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
	BaseURL       string
	responses     map[string]*gojsonq.JSONQ
	fixture       fixtureFile
}

// NewFixture creates a to later run steps for populating test data
func NewFixture(fs afero.Fs, apiKey, stripeAccount, baseURL, file string) (*Fixture, error) {
	fxt := Fixture{
		Fs:            fs,
		APIKey:        apiKey,
		StripeAccount: stripeAccount,
		BaseURL:       baseURL,
		responses:     make(map[string]*gojsonq.JSONQ),
	}

	var filedata []byte

	var err error

	if _, ok := reverseMap()[file]; ok {
		f, err := FS.Open(file)
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

	if fxt.fixture.Meta.Version > SupportedVersions {
		return nil, fmt.Errorf("Fixture version not supported: %d", fxt.fixture.Meta.Version)
	}

	return &fxt, nil
}

// Execute takes the parsed fixture file and runs through all the requests
// defined to populate the user's account
func (fxt *Fixture) Execute() error {
	for _, data := range fxt.fixture.Fixtures {
		fmt.Printf("Setting up fixture for: %s\n", data.Name)

		resp, err := fxt.makeRequest(data)
		if err != nil {
			return err
		}

		fxt.responses[data.Name] = gojsonq.New().FromString(string(resp))
	}

	return nil
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

func (fxt *Fixture) parsePath(http fixture) string {
	if r, containsQuery := matchFixtureQuery(http.Path); containsQuery {
		var newPath []string

		matches := r.FindAllStringSubmatch(http.Path, -1)
		pathParts := r.Split(http.Path, -1)

		for i, match := range matches {
			value := fxt.parseQuery(match[0])

			newPath = append(newPath, pathParts[i])
			newPath = append(newPath, value)
		}

		if len(pathParts)%2 == 0 {
			newPath = append(newPath, pathParts[len(pathParts)-1])
		}

		return path.Join(newPath...)
	}

	return http.Path
}

func (fxt *Fixture) createParams(params interface{}) *requests.RequestParameters {
	requestParams := requests.RequestParameters{}
	requestParams.AppendData(fxt.parseInterface(params))

	requestParams.SetStripeAccount(fxt.StripeAccount)

	return &requestParams
}

func (fxt *Fixture) parseInterface(params interface{}) []string {
	var data []string

	var cleanData []string

	switch v := reflect.ValueOf(params); v.Kind() {
	case reflect.Map:
		m := params.(map[string]interface{})
		data = append(data, fxt.parseMap(m, "", -1)...)
	case reflect.Array:
		a := params.([]interface{})
		data = append(data, fxt.parseArray(a, "", -1)...)
	default:
	}

	for _, d := range data {
		if strings.TrimSpace(d) != "" {
			cleanData = append(cleanData, strings.TrimSpace(d))
		}
	}

	return cleanData
}

func (fxt *Fixture) parseMap(params map[string]interface{}, parent string, index int) []string {
	data := make([]string, len(params))

	var keyname string

	for key, value := range params {
		switch {
		case parent != "" && index >= 0:
			keyname = fmt.Sprintf("%s[%d][%s]", parent, index, key)
		case parent != "":
			keyname = fmt.Sprintf("%s[%s]", parent, key)
		default:
			keyname = key
		}

		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.String:
			data = append(data, fmt.Sprintf("%s=%s", keyname, fxt.parseQuery(v.String())))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data = append(data, fmt.Sprintf("%s=%v", keyname, v.Int()))
		case reflect.Float32, reflect.Float64:
			data = append(data, fmt.Sprintf("%s=%v", keyname, v.Float()))
		case reflect.Bool:
			data = append(data, fmt.Sprintf("%s=%t", keyname, v.Bool()))
		case reflect.Map:
			m := value.(map[string]interface{})

			result := fxt.parseMap(m, keyname, index)
			data = append(data, result...)
		case reflect.Array, reflect.Slice:
			a := value.([]interface{})

			result := fxt.parseArray(a, keyname, index)
			data = append(data, result...)
		default:
			continue
		}
	}

	return data
}

func (fxt *Fixture) parseArray(params []interface{}, parent string, index int) []string {
	data := make([]string, len(params))

	for _, value := range params {
		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.String:
			data = append(data, fmt.Sprintf("%s[]=%s", parent, fxt.parseQuery(v.String())))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data = append(data, fmt.Sprintf("%s[]=%v", parent, v.Int()))
		case reflect.Map:
			m := value.(map[string]interface{})
			// When we parse arrays of maps, we want to track an index for the request
			data = append(data, fxt.parseMap(m, parent, index+1)...)
		case reflect.Array, reflect.Slice:
			a := value.([]interface{})
			data = append(data, fxt.parseArray(a, parent, index)...)
		default:
			continue
		}
	}

	return data
}

func (fxt *Fixture) parseQuery(value string) string {
	if query, isQuery := toFixtureQuery(value); isQuery {
		name := query.Name

		// Check if there is a default value specified
		if query.DefaultValue != "" {
			value = query.DefaultValue
		}

		// Catch and insert .env values
		if name == ".env" {
			key := query.Query
			// Check if env variable is present
			envValue := os.Getenv(key)
			if envValue == "" {
				// Try to load from .env file
				dir, err := os.Getwd()
				if err != nil {
					dir = ""
				}
				err = godotenv.Load(path.Join(dir, ".env"))
				if err != nil {
					return value
				}
				envValue = os.Getenv(key)
			}
			if envValue == "" {
				fmt.Printf("No value for env var: %s\n", key)
				return value
			}
			return envValue
		}

		// Reset just in case someone else called a query here
		fxt.responses[name].Reset()

		query := query.Query
		findResult, err := fxt.responses[name].FindR(query)
		if err != nil {
			return value
		}
		findResultString, _ := findResult.String()
		return findResultString
	}

	return value
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

// toFixtureQuery will parse a string into a fixtureQuery struct, additionally
// returning a bool indicating the value did contain a fixtureQuery.
func toFixtureQuery(value string) (fixtureQuery, bool) {
	var query fixtureQuery
	isQuery := false

	if r, didMatch := matchFixtureQuery(value); didMatch {
		isQuery = true
		match := r.FindStringSubmatch(value)
		query = fixtureQuery{Name: match[1], Query: match[2], DefaultValue: match[3]}
	}

	return query, isQuery
}

// matchQuery will attempt to find matches for a fixture query pattern
// returning a *Regexp which can be used to further parse and a boolean
// indicating a match was found.
func matchFixtureQuery(value string) (*regexp.Regexp, bool) {
	// Queries will start with `${` and end with `}`. The `:` is a
	// separator for `name:json_path`. Additionally, default value will
	// be specified after the `|`.
	// example: ${name:json_path|default_value}
	r := regexp.MustCompile(`\${([^\|}]+):([^\|}]+)\|?([^/\n]+)?}`)
	if r.Match([]byte(value)) {
		return r, true
	}

	return nil, false
}
