package validators

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
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
	requireErrorCategory(t, result, errorcategory.UserInput)
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
	requireErrorCategory(t, result, errorcategory.UserInput)
}

func TestExactArgsTooManyMoreThan1(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	args := []string{"foo", "bar", "baz"}

	result := ExactArgs(2)(c, args)
	require.EqualError(t, result, "`c` requires exactly 2 positional arguments. See `c --help` for supported flags and usage")
	requireErrorCategory(t, result, errorcategory.UserInput)
}

func TestMaximumNArgsTooMany(t *testing.T) {
	c := &cobra.Command{Use: "c"}
	err := MaximumNArgs(1)(c, []string{"foo", "bar"})

	require.EqualError(t, err, "`c` accepts at maximum 1 positional argument. See `c --help` for supported flags and usage")
	requireErrorCategory(t, err, errorcategory.UserInput)
}

func requireErrorCategory(t *testing.T, err error, expected errorcategory.Category) {
	t.Helper()
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	require.Equal(t, expected, category)
}
