package gatedwriter

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuffersWhileClosed(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	n, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Empty(t, out.String(), "nothing should reach out while closed")
}

func TestFlushesOnOpen(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	w.Write([]byte("hello "))
	w.Write([]byte("world"))

	require.NoError(t, w.Open())
	assert.Equal(t, "hello world", out.String())
}

func TestWritesDirectlyWhenOpen(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	require.NoError(t, w.Open())
	w.Write([]byte("direct"))
	assert.Equal(t, "direct", out.String())
}

func TestOpenIsIdempotent(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	w.Write([]byte("once"))
	require.NoError(t, w.Open())
	require.NoError(t, w.Open()) // second Open should not re-flush
	assert.Equal(t, "once", out.String())
}

func TestCloseRebuffers(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	require.NoError(t, w.Open())
	w.Close()
	w.Write([]byte("rebuffered"))
	assert.Equal(t, "", out.String(), "closed again, should not reach out")

	require.NoError(t, w.Open())
	assert.Equal(t, "rebuffered", out.String())
}

func TestMaxBufferDropsOverflow(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 5)

	w.Write([]byte("hello")) // exactly 5 bytes — fits
	w.Write([]byte("X"))     // would exceed cap — dropped

	require.NoError(t, w.Open())
	assert.Equal(t, "hello", out.String())
}

func TestMaxBufferZeroDefaultsToDefaultMaxBuffer(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	big := strings.Repeat("a", defaultMaxBuffer)
	w.Write([]byte(big))

	require.NoError(t, w.Open())
	assert.Equal(t, big, out.String())
}

func TestNilOutDefaultsToDiscard(t *testing.T) {
	w := NewGatedWriter(nil, 0)
	w.Write([]byte("discarded"))
	// should not panic; Open flushes to io.Discard
	require.NoError(t, w.Open())
}

func TestFdWithFile(t *testing.T) {
	w := NewGatedWriter(os.Stderr, 0)
	assert.Equal(t, os.Stderr.Fd(), w.Fd())
}

func TestFdWithNonFile(t *testing.T) {
	var buf bytes.Buffer
	w := NewGatedWriter(&buf, 0)
	assert.Equal(t, ^uintptr(0), w.Fd())
}

func TestConcurrentWrites(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Write([]byte("x"))
		}()
	}
	wg.Wait()

	require.NoError(t, w.Open())
	assert.Equal(t, 100, len(out.String()))
}

func TestConcurrentWritesDuringOpen(t *testing.T) {
	var out bytes.Buffer
	w := NewGatedWriter(&out, 0)

	// pre-buffer some data
	for i := 0; i < 50; i++ {
		w.Write([]byte("x"))
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.Open()
	}()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Write([]byte("x"))
		}()
	}
	wg.Wait()

	// all 100 writes should reach out (either flushed or direct)
	assert.Equal(t, 100, len(out.String()))
}
