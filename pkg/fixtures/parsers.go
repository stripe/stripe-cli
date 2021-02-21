package fixtures

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

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
			/*
				When converting fixture values to JSON, numeric values
				reflect as float types. Thus in order to output the correct
				value we should parse as such:

				10 => 10
				3.145 => 3.145
				25.00 => 25
				20.10 => 20.1

				In order to preserve decimal places but strip them when
				unnecessary (i.e 1.0), we must use strconv with the special
				precision value of -1.

				We cannot use %v here because it reverts to %g which uses
				%e (scientific notation) for larger values otherwise %f
				(float), which will not strip the decimal places from 4.00
			*/
			s64 := strconv.FormatFloat(v.Float(), 'f', -1, 64)
			data = append(data, fmt.Sprintf("%s=%s", keyname, s64))
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

		result := fxt.responses[name].Get(query.Query)
		if len(result.String()) != 0 {
			return result.String()
		} else {
			return value
		}
	}

	return value
}
