package coopcmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return captureOutput(t, &os.Stdout, fn)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return captureOutput(t, &os.Stderr, fn)
}

func captureOutput(t *testing.T, target **os.File, fn func()) string {
	t.Helper()

	orig := *target
	r, w, err := os.Pipe()
	require.NoError(t, err)
	*target = w

	var buf bytes.Buffer
	readErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, r)
		readErr <- err
	}()

	closed := false
	defer func() {
		*target = orig
		if !closed {
			_ = w.Close()
		}
		_ = r.Close()
	}()

	fn()

	closed = true
	require.NoError(t, w.Close())
	*target = orig

	require.NoError(t, <-readErr)
	require.NoError(t, r.Close())
	return strings.TrimSpace(buf.String())
}

func TestCaptureStdoutDrainsLargeOutput(t *testing.T) {
	output := captureStdout(t, func() {
		_, err := os.Stdout.WriteString(strings.Repeat("x", 128*1024))
		require.NoError(t, err)
	})

	assert.Len(t, output, 128*1024)
}
