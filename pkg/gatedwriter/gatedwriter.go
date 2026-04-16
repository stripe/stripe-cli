// Package gatedwriter provides a buffered writer that can be opened and closed
// to control the flow of data to the underlying writer.
package gatedwriter

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// GatedWriter is an io.Writer that buffers writes until opened. Once Open is
// called, buffered data is flushed and subsequent writes go directly to the
// underlying writer. Closing the gate resumes buffering. Safe for concurrent use.
type GatedWriter struct {
	mu        sync.Mutex
	out       io.Writer    // underlying destination (must be non-nil)
	buf       bytes.Buffer // buffered data while closed
	open      bool         // when true, writes go directly to out
	maxBuffer int64        // maximum number of bytes to buffer; 0 = defaultMaxBuffer
}

var _ io.Writer = (*GatedWriter)(nil)

// defaultMaxBuffer is the maximum bytes to buffer while the gate is closed (100 KB).
// Writes that would exceed this limit are dropped silently.
const defaultMaxBuffer = 100 * 1024

// NewGatedWriter creates a closed GatedWriter that writes to out once opened.
// maxBuffer caps the in-memory buffer in bytes; 0 uses defaultMaxBuffer.
// If out is nil, io.Discard is used.
func NewGatedWriter(out io.Writer, maxBuffer int64) *GatedWriter {
	if out == nil {
		out = io.Discard
	}
	if maxBuffer == 0 {
		maxBuffer = defaultMaxBuffer
	}
	return &GatedWriter{out: out, maxBuffer: maxBuffer}
}

// Write buffers p while the gate is closed. Once open, it writes directly to
// the underlying writer. Writes that would push the buffer past maxBuffer are
// dropped silently.
func (g *GatedWriter) Write(p []byte) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open {
		return g.out.Write(p)
	}
	if g.maxBuffer > 0 && int64(g.buf.Len()+len(p)) > g.maxBuffer {
		return len(p), nil // drop silently
	}
	n, _ := g.buf.Write(p) // bytes.Buffer.Write never returns error
	return n, nil
}

// Open flushes any buffered data to the underlying writer and opens the gate
// so future writes go directly to out. Calling Open on an already-open writer
// is a no-op.
func (g *GatedWriter) Open() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open {
		return nil
	}
	data := g.buf.Bytes()
	g.buf.Reset()
	g.open = true
	if len(data) > 0 {
		_, err := g.out.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes the gate, causing subsequent writes to be buffered again.
func (g *GatedWriter) Close() {
	g.mu.Lock()
	g.open = false
	g.mu.Unlock()
}

// Fd returns the file descriptor of the underlying writer if it is an *os.File,
// allowing callers to check whether the destination is a terminal. Returns
// ^uintptr(0) if the underlying writer is not an *os.File.
func (g *GatedWriter) Fd() uintptr {
	g.mu.Lock()
	out := g.out
	g.mu.Unlock()
	if f, ok := out.(*os.File); ok {
		return f.Fd()
	}
	return ^uintptr(0) // invalid fd
}
