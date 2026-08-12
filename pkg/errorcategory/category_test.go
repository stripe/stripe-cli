package errorcategory

import (
	"errors"
	"fmt"
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
	err := New(UserInput, "unchanged message")

	require.EqualError(t, err, "unchanged message")

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, UserInput, category)
}

func TestErrorf(t *testing.T) {
	err := Errorf(API, "request failed with status %d", 500)

	require.EqualError(t, err, "request failed with status 500")

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, API, category)
}

func TestErrorfWraps(t *testing.T) {
	target := &testError{message: "unchanged message"}
	err := Errorf(Filesystem, "reading configuration: %w", target)

	require.EqualError(t, err, "reading configuration: unchanged message")
	require.ErrorIs(t, err, target)

	var typed *testError
	require.ErrorAs(t, err, &typed)
	require.Same(t, target, typed)

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, Filesystem, category)
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

func TestGetUsesOutermostCategory(t *testing.T) {
	err := With(With(errors.New("nested"), Network), API)

	category, ok := Get(err)
	require.True(t, ok)
	require.Equal(t, API, category)
}
