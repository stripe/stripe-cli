package origins

import (
	"errors"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const wrappingFormat = "context: %w"

func categorized(message string, err error) {
	_ = errorcategory.New(errorcategory.UserInput, message)
	_ = errorcategory.Errorf(errorcategory.Auth, "authentication failed: %s", message)
	_ = errorcategory.With(err, errorcategory.Network)
}

func invalid(message string) {
	_ = errors.New("leaf")                   // want "errors.New creates an uncategorized error"
	_ = fmt.Errorf("formatted: %s", message) // want "fmt.Errorf creates an uncategorized error"
	_ = fmt.Errorf("escaped %%w")            // want "fmt.Errorf creates an uncategorized error"
	_ = fmt.Errorf(message)                  // want "non-constant format cannot be verified as wrapping"
}

func wrapping(err error) {
	_ = fmt.Errorf("context: %w", err)
	_ = fmt.Errorf("context: %[1]w", err)
	_ = fmt.Errorf(wrappingFormat, err)
}

func suppressed(message string) {
	//nolint:errorcategory -- this fixture verifies suppression when no category is available
	_ = errors.New(message)
	_ = fmt.Errorf("leaf") //nolint:errorcategory -- this fixture verifies same-line suppression

	//nolint:errorcategory
	_ = errors.New("missing reason") // want "requires a reason"
}
