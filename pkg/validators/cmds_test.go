package validators

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestNoArgs(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	args := []string{}

	result := NoArgs(c, args)
	require.Nil(t, result)
}

func TestNoArgsWithArgs(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	args := []string{"foo"}

	result := NoArgs(c, args)
	require.EqualError(t, result, "`c` does not take any positional arguments. See `c --help` for supported flags and usage")
}

func TestExactArgs(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	args := []string{"foo"}

	result := ExactArgs(1)(c, args)
	require.Nil(t, result)
}

func TestExactArgsTooMany(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	args := []string{"foo", "bar"}

	result := ExactArgs(1)(c, args)
	require.EqualError(t, result, "`c` requires exactly 1 positional argument. See `c --help` for supported flags and usage")
}

func TestExactArgsTooManyMoreThan1(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	args := []string{"foo", "bar", "baz"}

	result := ExactArgs(2)(c, args)
	require.EqualError(t, result, "`c` requires exactly 2 positional arguments. See `c --help` for supported flags and usage")
}
