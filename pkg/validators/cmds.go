package validators

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func getCommandPath(cmd *cobra.Command) string {
	var commandPath string
	if cmd.Annotations["scope"] == "plugin" {
		commandPath = fmt.Sprintf("stripe %s", cmd.CommandPath())
	} else {
		commandPath = cmd.CommandPath()
	}

	return commandPath
}

// NoArgs is a validator for commands to print an error when an argument is provided
func NoArgs(cmd *cobra.Command, args []string) error {
	commandPath := getCommandPath(cmd)
	errorMessage := fmt.Sprintf(
		"`%s` does not take any positional arguments. See `%s --help` for supported flags and usage",
		commandPath,
		commandPath,
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
		commandPath := getCommandPath(cmd)
		argument := "positional argument"
		if num != 1 {
			argument = "positional arguments"
		}

		errorMessage := fmt.Sprintf(
			"`%s` requires exactly %d %s. See `%s --help` for supported flags and usage",
			commandPath,
			num,
			argument,
			commandPath,
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
		commandPath := getCommandPath(cmd)
		argument := "positional argument"
		if num > 1 {
			argument = "positional arguments"
		}

		errorMessage := fmt.Sprintf(
			"`%s` accepts at maximum %d %s. See `%s --help` for supported flags and usage",
			commandPath,
			num,
			argument,
			commandPath,
		)

		if len(args) > num {
			return errors.New(errorMessage)
		}
		return nil
	}
}

// MinimumNArgs is a validator for commands to print an error when the provided
// args are less than the minimum amount
func MinimumNArgs(num int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		commandPath := getCommandPath(cmd)
		argument := "positional argument"
		if num > 1 {
			argument = "positional arguments"
		}

		errorMessage := fmt.Sprintf(
			"`%s` requires at least %d %s. See `%s --help` for supported flags and usage",
			commandPath,
			num,
			argument,
			commandPath,
		)

		if len(args) < num {
			return errors.New(errorMessage)
		}
		return nil
	}
}
