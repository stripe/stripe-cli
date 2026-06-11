package pager_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/internal/pager"
)

func TestWriter_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	w := pager.New(&buf, true)
	defer w.Close()

	_, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", buf.String())
}

func TestWriter_Disabled(t *testing.T) {
	var buf bytes.Buffer
	w := pager.New(&buf, false)
	defer w.Close()

	_, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", buf.String())
}
