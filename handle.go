// Package log provides a [slog.Handler] for structured, human-readable CLI output.
//
// [CLIHandler] formats time, level, source, label, and message as built-in output
// components, followed by ordinary attributes with dot-separated group names.
// [Style] controls their appearance; [AttrReplacer] applies only to ordinary
// attributes. Configured colors are enabled for terminal files and omitted for
// other writers.
//
// Pass the result of [NewCLIHandler] to [slog.New] to create a logger.
package log

import (
	"context"
	"io"
	"log/slog"
)

var _ slog.Handler = (*CLIHandler)(nil)

// CLIHandler implements [slog.Handler] for human-readable CLI output.
// Create a handler with [NewCLIHandler]; the zero value is not ready for use.
//
// Its methods may be called concurrently. Handlers returned by [CLIHandler.WithAttrs]
// and [CLIHandler.WithGroup] share the output lock and source cache, but do not
// change the original handler's attributes or groups. Separately constructed
// handlers do not share an output lock.
type CLIHandler struct {
	config config
	attrs  []byte
	groups []string
	writer *writer
	source *source
}

// NewCLIHandler returns a handler that writes to w using opts.
// A nil w discards output. Zero-valued options are ignored.
//
// By default, the handler uses [DefaultStyle] and a minimum level of [slog.LevelInfo],
// with time, source, and label output disabled. Terminal detection requires an
// *os.File; terminal output is adapted for Windows when needed.
func NewCLIHandler(w io.Writer, opts ...CLIHandlerOption) slog.Handler {
	option := newOption()
	for _, opt := range opts {
		if opt.apply != nil {
			opt.apply(&option)
		}
	}
	writer := newWriter(w)
	config := newConfig(&option, writer.terminal)
	return &CLIHandler{
		config: config,
		writer: writer,
		source: newSource(config.source.value),
	}
}

// Enabled reports whether level meets the configured minimum.
// It reads the level threshold on each call and ignores the context.
func (o *CLIHandler) Enabled(_ context.Context, level slog.Level) bool {
	threshold := o.config.level.threshold
	if threshold == nil {
		return true
	}
	return level >= threshold.Level()
}

// Handle formats record and writes the result, followed by a newline.
// It returns any write error, including [io.ErrShortWrite] for a partial write.
// It does not check the minimum level or use the context; [slog.Logger] calls
// [CLIHandler.Enabled] before passing a record to Handle.
func (o *CLIHandler) Handle(_ context.Context, record slog.Record) error {
	config := &o.config
	s := acquireState()
	if config.time.enabled && !record.Time.IsZero() {
		s.line.appendTime(record.Time, &config.time)
	}
	s.line.appendLevel(record.Level, config.level.texts)
	if o.source != nil {
		text := o.source.resolve(record.PC)
		if text != nil {
			s.line.appendSource(text, &config.source)
		}
	}
	if config.label.value != "" {
		s.line.appendLabel(&config.label)
	}
	s.line.appendMessage(record.Message, &config.message)
	s.line.appendCachedAttrs(o.attrs)
	if record.NumAttrs() > 0 {
		s.attr.prepare(o.groups)
		yield := func(attr slog.Attr, kind slog.Kind, path []byte) {
			s.line.appendAttr(attr, kind, path, &config.attr)
		}
		record.Attrs(func(attr slog.Attr) bool {
			s.attr.resolve(attr, config.attr.replacer, yield)
			return true
		})
	}
	s.line.appendNewline()
	err := o.writer.write(s.line.buf)
	releaseState(s)
	return err
}

// WithAttrs returns a handler with attrs appended to its existing attributes.
// It resolves, replaces, and formats attrs during this call, using the current
// groups. Later records reuse the formatted output without repeating that work.
// An empty attrs slice returns the receiver.
func (o *CLIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return o
	}
	h2 := *o
	config := &h2.config.attr
	s := acquireState()
	s.line.appendCachedAttrs(o.attrs)
	s.attr.prepare(o.groups)
	yield := func(attr slog.Attr, kind slog.Kind, path []byte) {
		s.line.appendAttr(attr, kind, path, config)
	}
	for _, attr := range attrs {
		s.attr.resolve(attr, config.replacer, yield)
	}
	h2.attrs = append([]byte(nil), s.line.buf...)
	releaseState(s)
	return &h2
}

// WithGroup returns a handler that adds name to the groups of subsequent
// attributes. Previously attached attributes and built-in output are unaffected.
// An empty name returns the receiver.
func (o *CLIHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return o
	}
	h2 := *o
	h2.groups = make([]string, len(o.groups)+1)
	copy(h2.groups, o.groups)
	h2.groups[len(o.groups)] = name
	return &h2
}
