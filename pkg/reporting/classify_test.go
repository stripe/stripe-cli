package reporting

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
)

type testNetworkError struct{}

func (testNetworkError) Error() string   { return "network error" }
func (testNetworkError) Timeout() bool   { return false }
func (testNetworkError) Temporary() bool { return true }

func TestClassifyError(t *testing.T) {
	unknownFlag := func() error {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		return flags.Parse([]string{"--unknown"})
	}()
	missingFlagValue := func() error {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("name", "", "")
		return flags.Parse([]string{"--name"})
	}()

	tests := []struct {
		name     string
		err      error
		expected errorcategory.Category
	}{
		{name: "explicit", err: errorcategory.With(errors.New("input"), errorcategory.UserInput), expected: errorcategory.UserInput},
		{name: "explicit takes precedence", err: errorcategory.With(&os.PathError{Op: "open", Path: "file", Err: errors.New("failed")}, errorcategory.Auth), expected: errorcategory.Auth},
		{name: "unknown flag", err: unknownFlag, expected: errorcategory.UserInput},
		{name: "missing flag value", err: missingFlagValue, expected: errorcategory.UserInput},
		{name: "API unauthorized", err: requests.RequestError{StatusCode: 401}, expected: errorcategory.Auth},
		{name: "API forbidden", err: &requests.RequestError{StatusCode: 403}, expected: errorcategory.Auth},
		{name: "API client error", err: requests.RequestError{StatusCode: 400}, expected: errorcategory.API},
		{name: "API server error", err: requests.RequestError{StatusCode: 500}, expected: errorcategory.API},
		{name: "filesystem", err: &os.PathError{Op: "open", Path: "file", Err: errors.New("failed")}, expected: errorcategory.Filesystem},
		{name: "URL", err: &url.Error{Op: "Get", URL: "https://example.com", Err: errors.New("failed")}, expected: errorcategory.Network},
		{name: "network", err: testNetworkError{}, expected: errorcategory.Network},
		{name: "canceled", err: context.Canceled, expected: errorcategory.UserInput},
		{name: "wrapped", err: fmt.Errorf("outer: %w", &os.PathError{Op: "open", Path: "file", Err: errors.New("failed")}), expected: errorcategory.Filesystem},
		{name: "fallback", err: errors.New("unexpected"), expected: errorcategory.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, classifyError(test.err))
		})
	}
}
