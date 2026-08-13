// Package errorcategory attaches stable categories to errors without changing
// their messages or unwrapping behavior.
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

// With returns an error with an explicit category. A nil error remains nil.
func With(err error, category Category) error {
	if err == nil {
		return nil
	}
	return categorizedError{err: err, category: category}
}

// UserInputErrorf formats an error categorized as UserInput.
func UserInputErrorf(format string, args ...any) error {
	return With(fmt.Errorf(format, args...), UserInput)
}

// Get returns the first explicit category in err's unwrap tree.
func Get(err error) (Category, bool) {
	var categorized Categorized
	if !errors.As(err, &categorized) {
		return "", false
	}
	return categorized.ErrorCategory(), true
}
