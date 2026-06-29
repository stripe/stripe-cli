package spinner

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_NonTTY_RunsFn(t *testing.T) {
	var buf bytes.Buffer
	called := false

	err := New().
		WithOutput(&buf).
		WithLabel("loading...").
		Run(func() error {
			called = true
			return nil
		})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestRun_Disabled_RunsFn(t *testing.T) {
	called := false

	err := New().
		WithDisabled(true).
		Run(func() error {
			called = true
			return nil
		})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestRun_PropagatesError(t *testing.T) {
	want := errors.New("something went wrong")

	err := New().
		WithDisabled(true).
		Run(func() error { return want })

	assert.Equal(t, want, err)
}

func TestRun_NonTTY_NoOutput(t *testing.T) {
	var buf bytes.Buffer

	_ = New().
		WithOutput(&buf).
		WithFinalMsg("✓ done").
		Run(func() error { return nil })

	// No animation or final message should be written to a non-TTY writer.
	assert.Empty(t, buf.String())
}

func TestRun_Disabled_NoOutput(t *testing.T) {
	var buf bytes.Buffer

	_ = New().
		WithOutput(&buf).
		WithDisabled(true).
		WithFinalMsg("✓ done").
		Run(func() error { return nil })

	assert.Empty(t, buf.String())
}
