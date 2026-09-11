package log

import (
	"log/slog"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
)

// source shares immutable, escaped source text across a handler and its derivatives.
// Its cache supports concurrent lookups; the source selector is fixed at construction.
type source struct {
	// last avoids the map lock for repeated calls from the same location.
	last    atomic.Pointer[sourceEntry]
	mu      sync.RWMutex
	entries map[uintptr]*sourceEntry
	value   func(*slog.Source) string
}

// newSource binds a value selector to a cache, or returns nil when disabled.
func newSource(value func(*slog.Source) string) *source {
	if value == nil {
		return nil
	}
	return &source{
		entries: make(map[uintptr]*sourceEntry),
		value:   value,
	}
}

// resolve returns shared, escaped source text for pc, without style decoration.
// The result must not be modified. A zero or unresolvable pc returns nil.
func (o *source) resolve(pc uintptr) []byte {
	if pc == 0 {
		return nil
	}
	if entry := o.last.Load(); entry != nil && entry.pc == pc {
		return entry.text
	}
	o.mu.RLock()
	entry := o.entries[pc]
	o.mu.RUnlock()
	if entry != nil {
		o.last.Store(entry)
		return entry.text
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if frame == (runtime.Frame{}) {
		return nil
	}
	value := o.value(&slog.Source{
		Function: frame.Function,
		File:     frame.File,
		Line:     frame.Line,
	})
	entry = newSourceEntry(pc, value, frame.Line)
	o.mu.Lock()
	// Another call may have cached the location while this entry was formatted.
	if cached := o.entries[pc]; cached != nil {
		o.mu.Unlock()
		o.last.Store(cached)
		return cached.text
	}
	o.entries[pc] = entry
	o.mu.Unlock()
	o.last.Store(entry)
	return entry.text
}

// sourceEntry associates a program counter with its formatted source text.
type sourceEntry struct {
	pc   uintptr
	text []byte
}

// newSourceEntry escapes the source value and includes its line inside any quotes.
func newSourceEntry(pc uintptr, value string, line int) *sourceEntry {
	text, quoted := escapeText(nil, value)
	if quoted {
		text = text[:len(text)-1]
	}
	text = append(text, ':')
	text = strconv.AppendInt(text, int64(line), 10)
	if quoted {
		text = append(text, '"')
	}
	return &sourceEntry{
		pc:   pc,
		text: text,
	}
}
