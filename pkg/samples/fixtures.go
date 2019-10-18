package samples

import (
	"encoding/json"
	"fmt"
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

// Fixture contains a mapping of an individual fixtures responses for querying
type Fixture struct {
	Fs        afero.Fs
	APIKey    string
	BaseURL   string
	responses map[string]*gojsonq.JSONQ
	fixture   fixtureFile
}

// NewFixture creates a to later run steps for populating test data
func NewFixture(fs afero.Fs, apiKey, baseURL, file string) (*Fixture, error) {
	fxt := Fixture{
		Fs:        fs,
		APIKey:    apiKey,
		BaseURL:   baseURL,
		responses: make(map[string]*gojsonq.JSONQ),
	}

	filedata, err := afero.ReadFile(fxt.Fs, file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(filedata, &fxt.fixture)
	if err != nil {
		return nil, err
	}

	if fxt.fixture.Meta.Version > SupportedVersions {
		return nil, fmt.Errorf("Fixture version not supported: %s", string(fxt.fixture.Meta.Version))
	}

	return &fxt, nil
}

// Execute takes the parsed fixture file and runs through all the requests
// defined to populate the user's account
func (fxt *Fixture) Execute() error {
	for _, data := range fxt.fixture.Fixtures {
		fmt.Println(fmt.Sprintf("Setting up fixture for: %s", data.Name))

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
	r := regexp.MustCompile(`(\${[\w-]+:[\w-\.]+})`)
	if r.Match([]byte(http.Path)) {
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

	return &requestParams
}

func (fxt *Fixture) parseInterface(params interface{}) []string {
	var data []string
	var cleanData []string

	switch v := reflect.ValueOf(params); v.Kind() {
	case reflect.Map:
		m := params.(map[string]interface{})
		data = append(data, fxt.parseMap(m, "")...)
	case reflect.Array:
		a := params.([]interface{})
		data = append(data, fxt.parseArray(a, "")...)
	default:
	}

	for _, d := range data {
		if strings.TrimSpace(d) != "" {
			cleanData = append(cleanData, strings.TrimSpace(d))
		}
	}

	return cleanData
}

func (fxt *Fixture) parseMap(params map[string]interface{}, parent string) []string {
	data := make([]string, len(params))

	var keyname string

	for key, value := range params {
		if parent != "" {
			keyname = fmt.Sprintf("%s[%s]", parent, key)
		} else {
			keyname = key
		}

		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.String:
			data = append(data, fmt.Sprintf("%s=%s", keyname, fxt.parseQuery(v.String())))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data = append(data, fmt.Sprintf("%s=%v", keyname, v.Int()))
		case reflect.Float32, reflect.Float64:
			data = append(data, fmt.Sprintf("%s=%v", keyname, v.Float()))
		case reflect.Map:
			m := value.(map[string]interface{})

			result := fxt.parseMap(m, keyname)
			if len(result) > 0 {
				data = append(data, result...)
			}
		case reflect.Array:
			a := value.([]interface{})

			result := fxt.parseArray(a, keyname)
			if len(result) > 0 {
				data = append(data, result...)
			}
		default:
			continue
		}
	}

	return data
}

func (fxt *Fixture) parseArray(params []interface{}, parent string) []string {
	data := make([]string, len(params))

	for _, value := range params {
		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.String:
			data = append(data, fxt.parseQuery(v.String()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data = append(data, string(v.Int()))
		case reflect.Map:
			m := value.(map[string]interface{})
			data = append(data, fxt.parseMap(m, parent)...)
		case reflect.Array:
			a := value.([]interface{})
			data = append(data, fxt.parseArray(a, parent)...)
		default:
			continue
		}
	}

	return data
}

func (fxt *Fixture) parseQuery(value string) string {
	// Queries to fill data will start with #$ and contain a : -- search for both
	// to make sure that we're trying to parse a query
	r := regexp.MustCompile(`\${(.+):(.+)}`)
	if r.Match([]byte(value)) {
		nameAndQuery := r.FindStringSubmatch(value)
		name := nameAndQuery[1]

		// Reset just in case someone else called a query here
		fxt.responses[name].Reset()

		query := nameAndQuery[2]

		return fxt.responses[name].Find(query).(string)
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
