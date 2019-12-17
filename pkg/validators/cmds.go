package validators

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// NoArgs is a validator for commands to print an error when an argument is provided
func NoArgs(cmd *cobra.Command, args []string) error {
	errorMessage := fmt.Sprintf(
		"%s does not take any arguments. See `stripe %s --help` for supported flags and usage",
		cmd.Name(),
		cmd.Name(),
	)

	if len(args) > 0 {
		return errors.New(errorMessage)
	}

	return nil
}

// ExactArgs is a validator for commands to print an error when the number provided
// is different than the arguments passed in
func ExactArgs(num int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		argument := "argument"
		if num > 1 {
			argument = "arguments"
		}

		errorMessage := fmt.Sprintf(
			"%s only takes %d %s. See `stripe %s --help` for supported flags and usage",
			cmd.Name(),
			num,
			argument,
			cmd.Name(),
		)

		if len(args) != num {
			return errors.New(errorMessage)
		}
		return nil
	}
}

// MaximumNArgs is a validator for commands to print an error when the provided
// args are greater than the maximum amount
func MaximumNArgs(num int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		argument := "argument"
		if num > 1 {
			argument = "arguments"
		}

		errorMessage := fmt.Sprintf(
			"%s only takes %d %s (or less). See `stripe %s --help` for supported flags and usage",
			cmd.Name(),
			num,
			argument,
			cmd.Name(),
		)

		if len(args) > num {
			return errors.New(errorMessage)
		}
		return nil
	}
}
