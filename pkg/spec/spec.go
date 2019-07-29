//go:generate vfsgendev -source="github.com/stripe/stripe-cli/pkg/spec".FS

package spec

import (
	"io/ioutil"
)

// LoadSpec loads the OpenAPI spec and returns it as a string.
func LoadSpec() (string, error) {
	file, err := FS.Open("./spec3.sdk.json")
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
