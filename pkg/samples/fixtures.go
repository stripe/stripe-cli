package samples

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/afero"
	"github.com/thedevsaddam/gojsonq"

	"github.com/stripe/stripe-cli/pkg/requests"
)

// SupportedVersions is the version number of the fixture template the CLI supports
const SupportedVersions = 0

type metaFixture struct {
	Version int `json:"_version"`
}

type fixtureFile struct {
	Meta     metaFixture `json:"_meta"`
	Fixtures []fixture   `json:"fixtures"`
}

type fixture struct {
	Name string            `json:"name"`
	HTTP map[string]string `json:"http"`
	Data interface{}       `json:"data"`
}

// Fixture foo
type Fixture struct {
	Fs        afero.Fs
	APIKey    string
	BaseURL   string
	responses map[string]*gojsonq.JSONQ
}

// NewFixture foo
func (fxt *Fixture) NewFixture(file string) error {
	var fixture fixtureFile
	fxt.responses = make(map[string]*gojsonq.JSONQ)

	filedata, err := afero.ReadFile(fxt.Fs, file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(filedata, &fixture)
	if err != nil {
		return err
	}

	if fixture.Meta.Version > SupportedVersions {
		return fmt.Errorf("Fixture version not supported: %s", string(fixture.Meta.Version))
	}

	for _, data := range fixture.Fixtures {
		fmt.Println(fmt.Sprintf("Setting up fixture for: %s", data.Name))

		resp, err := fxt.makeRequest(data)
		if err != nil {
			return err
		}

		fxt.responses[data.Name] = gojsonq.New().FromString(string(resp))
	}

	return nil
}

func (fxt *Fixture) makeRequest(data fixture) ([]byte, error) {
	req := requests.Base{
		Method:         strings.ToUpper(data.HTTP["method"]),
		SuppressOutput: true,
		APIBaseURL:     fxt.BaseURL,
	}
	return req.MakeRequest(fxt.APIKey, data.HTTP["path"], fxt.createParams(data.Data))
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
	if strings.HasPrefix(value, "#$") && strings.Contains(value, ":") {
		nameAndQuery := strings.SplitN(value, ":", 2)
		name := strings.TrimLeft(nameAndQuery[0], "#$")
		query := nameAndQuery[1]
		return fxt.responses[name].Find(query).(string)
	}

	return value
}
