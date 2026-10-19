package stripe

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

var pathPlaceholderRegexp = regexp.MustCompile(`{\w+}`)

// EscapePathSegment encodes a raw positional argument for one API path segment.
func EscapePathSegment(value string) (string, error) {
	switch value {
	case "":
		return "", errorcategory.New(errorcategory.UserInput, "path arguments cannot be empty")
	case ".", "..":
		// PathEscape leaves bare dot segments unchanged; URL resolution removes them.
		return "", errorcategory.New(errorcategory.UserInput, "path arguments cannot be . or ..")
	default:
		return url.PathEscape(value), nil
	}
}

// FormatURLPath inserts raw positional arguments into an API path template.
func FormatURLPath(path string, values []string) (string, error) {
	segments := make([]interface{}, len(values))
	for i, value := range values {
		segment, err := EscapePathSegment(value)
		if err != nil {
			return "", err
		}
		segments[i] = segment
	}

	format := pathPlaceholderRegexp.ReplaceAllString(path, "%s")
	return fmt.Sprintf(format, segments...), nil
}
