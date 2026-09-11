package log

import (
	"io"
	"os"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

// writer serializes output from a handler and its derivatives through one lock.
type writer struct {
	mu       sync.Mutex
	w        io.Writer
	terminal bool
	discard  bool
}

// newWriter returns a synchronized writer for w.
func newWriter(w io.Writer) *writer {
	terminal := isTerminal(w)
	w = resolveWriter(w, terminal)
	return &writer{
		w:        w,
		terminal: terminal,
		discard:  w == io.Discard,
	}
}

// write writes buf under the shared lock, or does nothing for a discard writer.
// It converts a partial write without an error to io.ErrShortWrite.
func (o *writer) write(buf []byte) error {
	if o.discard {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	n, err := o.w.Write(buf)
	if n != len(buf) && err == nil {
		return io.ErrShortWrite
	}
	return err
}

// resolveWriter maps nil to io.Discard and adapts terminal files for Windows.
func resolveWriter(w io.Writer, terminal bool) io.Writer {
	if w == nil {
		return io.Discard
	}
	if f, ok := w.(*os.File); ok && terminal {
		return colorable.NewColorable(f)
	}
	return w
}

// isTerminal reports whether w is an *os.File attached to a native or Cygwin terminal.
// It does not probe individual color capabilities.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
