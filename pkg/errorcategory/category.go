// Package errorcategory attaches stable categories to errors without changing
// their messages or unwrapping behavior.
//
// Categorize errors at the boundary that knows their user-facing meaning, such
// as validation, authentication, API or filesystem orchestration, or command
// code. Use New or Errorf to create a categorized error, or With to classify an
// existing error without adding context. Plain errors.New remains appropriate
// for private implementation details and sentinel errors whose category is not
// intrinsic. Plain fmt.Errorf with %w can add context while preserving an
// existing category or a recognizable concrete error in the unwrap tree.
//
// Do not recategorize an error merely because it passes through another layer.
// Wrapping an already categorized error with With intentionally overrides the
// inner category.
package errorcategory

import (
	"errors"
	"fmt"
)

// Category identifies the source of an error.
type Category string

const (
	UserInput  Category = "user_input"
	Auth       Category = "auth"
	Network    Category = "network"
	API        Category = "api"
	Filesystem Category = "filesystem"
	Internal   Category = "internal"
	Panic      Category = "panic"
)

// Categorized is implemented by errors with an explicit category.
type Categorized interface {
	error
	ErrorCategory() Category
}

type categorizedError struct {
	err      error
	category Category
}

func (e categorizedError) Error() string {
	return e.err.Error()
}

func (e categorizedError) Unwrap() error {
	return e.err
}

func (e categorizedError) ErrorCategory() Category {
	return e.category
}

// New returns a new error with an explicit category.
func New(category Category, message string) error {
	return With(errors.New(message), category)
}

// Errorf returns a formatted error with an explicit category. If the format
// uses %w, the returned error preserves the wrapped error's unwrap behavior.
func Errorf(category Category, format string, args ...any) error {
	return With(fmt.Errorf(format, args...), category)
}

// With returns an error with an explicit category. A nil error remains nil.
func With(err error, category Category) error {
	if err == nil {
		return nil
	}
	return categorizedError{err: err, category: category}
}

// Get returns the first explicit category in err's unwrap tree.
func Get(err error) (Category, bool) {
	var categorized Categorized
	if !errors.As(err, &categorized) {
		return "", false
	}
	return categorized.ErrorCategory(), true
}
