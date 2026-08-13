package errorcategory

import (
	"errors"
	"fmt"
)

type Category string

const (
	UserInput Category = "user_input"
	Auth      Category = "auth"
	Network   Category = "network"
)

func New(_ Category, _ string) error              { return nil }
func Errorf(_ Category, _ string, _ ...any) error { return nil }
func With(_ error, _ Category) error              { return nil }

var (
	uncategorizedLeaf      = errors.New("allowed inside category package")
	uncategorizedFormatted = fmt.Errorf("allowed inside category package")
)
