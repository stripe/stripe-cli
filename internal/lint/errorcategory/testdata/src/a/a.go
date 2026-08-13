package a

import (
	"errors"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const wrappingFormat = "context: " + "%w"

func categorized(err error) {
	_ = errorcategory.New(errorcategory.UserInput, "bad input")
	_ = errorcategory.Errorf(errorcategory.Auth, "bad profile %q", "default")
	_ = errorcategory.With(err, errorcategory.Network)
}

func invalid(err error, dynamic string) {
	_ = errors.New("leaf")             // want "errors.New creates an uncategorized error"
	_ = fmt.Errorf("leaf %s", "error") // want "fmt.Errorf creates an uncategorized error"
	_ = fmt.Errorf(dynamic, err)       // want "fmt.Errorf creates an uncategorized error"
	_ = fmt.Errorf("escaped %%w")      // want "fmt.Errorf creates an uncategorized error"
}

func wrapping(err error) {
	_ = fmt.Errorf("context: %w", err)
	_ = fmt.Errorf(wrappingFormat, err)
	_ = fmt.Errorf("context: %[1]w", err)
	_ = fmt.Errorf("context: %+w", err)
}

func suppressed() {
	//nolint:errorcategory -- this error is required by an external interface
	_ = errors.New("leaf")

	_ = fmt.Errorf("leaf") //nolint:errorcategory -- this format is required by an external interface

	//nolint:errorcategory
	_ = errors.New("leaf") // want "must include a reason"
}
