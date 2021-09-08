package fixtures

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// The functions in this file are responsible for taking the JSON
// data inside of a fixture file and building the corresponding form
// data representation. Stripe's API does not support JSON data so
// anything that gets sent over the network must be converted.
//
// It *might* be possible to clean this file up  a bit by using
// golangs "mime/multipart" package with `Writer`:
//	- https://golang.org/pkg/mime/multipart/#Writer
//
// Generally, if there is an easier way to correctly (and
// recursively) handle taking a JSON key value representation and
// turning into form data, we should do that.
//
// Additionally, this will handle our query parsing to dynamically
// replace fields inside of fixtures. Queries will take values from
// previous request responses and insert them as part of the
// executing query.
//
// As fixtures are run, each fixture response is stored in a map:
//		{
//   		<name of the fixture>: { json response },
//   		<name of the fixture>: { json response },
// 	 		...
//		}
//
// The supported query shapes are simple:
// 		$<name of fixture>:dot.path.to.field

// parsePath will inspect the path to see if it has a query in the
// path for requests that operate on specific objects (for example,
// GET /v1/customers/:id or POST /v1/subscriptions/:id)
//
// If a query is found, this returns the path with the value already
// in place. If there is no query, it returns the old path as-is.
func (fxt *Fixture) parsePath(http fixture) (string, error) {
	if r, containsQuery := matchFixtureQuery(http.Path); containsQuery {
		var newPath []string

		matches := r.FindAllStringSubmatch(http.Path, -1)
		pathParts := r.Split(http.Path, -1)

		for i, match := range matches {
			value, err := fxt.parseQuery(match[0])

			if err != nil {
				return "", err
			}

			newPath = append(newPath, pathParts[i])
			newPath = append(newPath, value)
		}

		if len(pathParts)%2 == 0 {
			newPath = append(newPath, pathParts[len(pathParts)-1])
		}

		return path.Join(newPath...), nil
	}

	return http.Path, nil
}

// parseInterface is the primary entrypoint into building the request
// data for fixtures. The data will always be provided as an
// interface{} and this will need to use reflection to determine how
// to proceed. There are two primary paths here, `parseMap` and
// `parseArray`, which will recursively traverse and convert the data
//
// This returns an array of clean form data to make the request.
func (fxt *Fixture) parseInterface(params interface{}) ([]string, error) {
	var data []string

	var cleanData []string

	switch v := reflect.ValueOf(params); v.Kind() {
	case reflect.Map:
		m := params.(map[string]interface{})
		parsed, err := fxt.parseMap(m, "", -1)
		if err != nil {
			return make([]string, 0), err
		}
		data = append(data, parsed...)
	case reflect.Array:
		a := params.([]interface{})
		parsed, err := fxt.parseArray(a, "")
		if err != nil {
			return make([]string, 0), err
		}
		data = append(data, parsed...)
	default:
	}

	for _, d := range data {
		if strings.TrimSpace(d) != "" {
			cleanData = append(cleanData, strings.TrimSpace(d))
		}
	}

	return cleanData, nil
}

// parseMap recursively parses a map of string => interface{} until
// each leaf node has a terminal type (String, Int, etc) that can no
// longer be recursively traversed.
func (fxt *Fixture) parseMap(params map[string]interface{}, parent string, index int) ([]string, error) {
	data := make([]string, len(params))

	var keyname string

	for key, value := range params {
		// Create the key name. As we start nesting deeper into the
		// request data, we need to nest this with brackets,
		// otherwise the data will be created at the wrong level.
		switch {
		case parent != "" && index >= 0:
			// ex: lines[0][id] = "id_0000", lines[1][id] = "id_1234", etc.
			keyname = fmt.Sprintf("%s[%d][%s]", parent, index, key)
		case parent != "":
			// ex: metadata[name] = "blah", metadata[timestamp] = 1231341525, etc.
			keyname = fmt.Sprintf("%s[%s]", parent, key)
		default:
			keyname = key
		}

		// Check the type of the value for this pair. If this is a
		// terminal type, append the data with the key. For maps and
		// arrays, keep parsing.
		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.String:
			// Strings can contain queries to load data from other
			// responses, check and load those.
			parsed, err := fxt.parseQuery(v.String())
			if err != nil {
				return make([]string, 0), err
			}
			data = append(data, fmt.Sprintf("%s=%s", keyname, parsed))
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

			result, err := fxt.parseMap(m, keyname, index)

			if err != nil {
				return make([]string, 0), err
			}

			data = append(data, result...)
		case reflect.Array, reflect.Slice:
			a := value.([]interface{})

			result, err := fxt.parseArray(a, keyname)

			if err != nil {
				return make([]string, 0), err
			}

			data = append(data, result...)
		// If for some reason we cannot parse the data, skip it
		default:
			continue
		}
	}

	return data, nil
}

// parseArray is similar to parseMap but doesn't have to build the
// multi-depth keys. Form data arrays contain brackets with nothing
// inside the bracket to designate an array instead of a key value
// pair.
func (fxt *Fixture) parseArray(params []interface{}, parent string) ([]string, error) {
	data := make([]string, len(params))

	// The index is only used for arrays of maps
	index := -1
	for _, value := range params {
		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.String:
			// A string can be a regular value or one we need to look up first, ex: ${product.id}
			parsed, err := fxt.parseQuery(v.String())
			if err != nil {
				return make([]string, 0), err
			}
			data = append(data, fmt.Sprintf("%s[]=%s", parent, parsed))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			data = append(data, fmt.Sprintf("%s[]=%v", parent, v.Int()))
		case reflect.Map:
			m := value.(map[string]interface{})
			// When we parse arrays of maps, we want to track the index of the element for the request
			// ex: lines[0][id] = "id_0000", lines[1][id] = "id_1234", etc.
			index++
			parsed, err := fxt.parseMap(m, parent, index)
			if err != nil {
				return make([]string, 0), err
			}
			data = append(data, parsed...)
		case reflect.Array, reflect.Slice:
			a := value.([]interface{})
			parsed, err := fxt.parseArray(a, parent)
			if err != nil {
				return make([]string, 0), err
			}
			data = append(data, parsed...)
		default:
			continue
		}
	}

	return data, nil
}

func normalizeForComparison(x string) string {
	r := strings.NewReplacer("_", "", "-", "")
	return r.Replace(strings.ToLower(x))
}

func findSimilarQueryNames(fxt *Fixture, name string) ([]string, bool) {
	keys := make([]string, 0, len(fxt.responses))
	for k := range fxt.responses {
		a := normalizeForComparison(k)
		b := normalizeForComparison(name)
		isSubstr := strings.Contains(a, b) || strings.Contains(b, a)

		if isSubstr && k != name {
			keys = append(keys, k)
		}
	}

	return keys, len(keys) > 0
}

// parseQuery checks strings for possible queries and replaces the
// corresponding value in its place. The supported query format is:
// 		$<name of fixture>:dot.path.to.field
func (fxt *Fixture) parseQuery(queryString string) (string, error) {
	value := queryString

	if query, isQuery := toFixtureQuery(queryString); isQuery {
		name := query.Name

		// Check if there is a default value specified
		if query.DefaultValue != "" {
			value = query.DefaultValue
		}

		// Catch and insert .env values
		if name == ".env" {
			// Check if env variable is present
			envValue, err := getEnvVar(query)
			if err != nil || envValue == "" {
				return value, nil
			}

			// Handle the case where only a substring of the original queryString was a query.
			// Ex: ${.env:BLAH}/blah/blah
			value = strings.ReplaceAll(queryString, query.Match, envValue)
			return value, nil
		}

		if _, ok := fxt.responses[name]; !ok {
			// An undeclared fixture name is being referenced
			var errorStrings []string
			color := ansi.Color(os.Stdout)

			referenceError := fmt.Errorf(
				"%s - an undeclared fixture name was referenced: %s",
				color.Red("✘ Validation error").String(),
				ansi.Bold(name),
			).Error()

			errorStrings = append(errorStrings, referenceError)

			if similar, exists := findSimilarQueryNames(fxt, name); exists {
				suggestions := fmt.Errorf(
					"%s: %v",
					ansi.Italic("Perhaps you meant one of the following"),
					strings.Join(similar, ", "),
				).Error()
				errorStrings = append(errorStrings, suggestions)
			}

			return "", fmt.Errorf(strings.Join(errorStrings, "\n"))
		}

		result := fxt.responses[name].Get(query.Query)
		if len(result.String()) != 0 {
			return result.String(), nil
		}

		return value, nil
	}

	return value, nil
}

// toFixtureQuery will parse a string into a fixtureQuery struct, additionally
// returning a bool indicating the value did contain a fixtureQuery.
func toFixtureQuery(value string) (fixtureQuery, bool) {
	var query fixtureQuery
	isQuery := false

	if r, didMatch := matchFixtureQuery(value); didMatch {
		isQuery = true
		match := r.FindStringSubmatch(value)
		query = fixtureQuery{Match: match[0], Name: match[1], Query: match[2], DefaultValue: match[3]}
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
