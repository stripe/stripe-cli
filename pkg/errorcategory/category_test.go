package errorcategory

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestNew(t *testing.T) {
	err := New(UserInput, "invalid input")

	require.EqualError(t, err, "invalid input")
	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, UserInput, category)
}

func TestErrorf(t *testing.T) {
	target := &testError{message: "target"}
	err := Errorf(Network, "request failed: %w", target)

	require.EqualError(t, err, "request failed: target")
	require.ErrorIs(t, err, target)
	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, Network, category)
}

func TestWith(t *testing.T) {
	target := &testError{message: "unchanged message"}
	err := With(target, Auth)

	require.EqualError(t, err, target.Error())
	require.ErrorIs(t, err, target)

	var typed *testError
	require.ErrorAs(t, err, &typed)
	require.Same(t, target, typed)
}

func TestWithNil(t *testing.T) {
	require.NoError(t, With(nil, Internal))
}

func TestGet(t *testing.T) {
	target := errors.New("target")
	err := fmt.Errorf("outer: %w", With(fmt.Errorf("inner: %w", target), UserInput))

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, UserInput, category)
	require.ErrorIs(t, err, target)
}

func TestGetWithoutCategory(t *testing.T) {
	category, ok := Get(errors.New("uncategorized"))
	require.False(t, ok)
	require.Empty(t, category)
}

func TestUserInputErrorf(t *testing.T) {
	err := UserInputErrorf("invalid %s: %q", "value", "foo")

	require.EqualError(t, err, `invalid value: "foo"`)

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, UserInput, category)
}

func TestGetUsesOutermostCategory(t *testing.T) {
	err := With(With(errors.New("nested"), Network), API)

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, API, category)
}

// Error reporters match WrapperTypeName against the type name reflection
// produces for a wrapped error, so renaming categorizedError must not silently
// break them.
func TestWrapperTypeName(t *testing.T) {
	err := With(errors.New("wrapped"), Internal)

	require.Equal(t, reflect.TypeOf(err).String(), WrapperTypeName)
	require.Equal(t, "errorcategory.categorizedError", WrapperTypeName)
}
