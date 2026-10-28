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
	"strings"
)

var _ slog.Handler = (*CLIHandler)(nil)

// CLIHandler implements [slog.Handler] for human-readable CLI output.
// Create a handler with [NewCLIHandler]; the zero value is not ready for use.
//
// Its methods are safe for concurrent use. [CLIHandler.WithAttrs],
// [CLIHandler.WithGroup], and [CLIHandler.WithLabel] preserve the original
// attributes, groups, and label while sharing the output lock and source cache.
// Separately constructed handlers use separate locks.
type CLIHandler struct {
	writer *writer
	source *source
	config config
	attrs  []byte
	groups []string
}

// NewCLIHandler returns a handler that writes to w using opts.
// A nil w discards output. Zero-valued options are ignored.
//
// By default, the handler uses [DefaultStyle] and a minimum level of [slog.LevelInfo],
// with time, source, and label output disabled. Terminal detection requires an
// *os.File; terminal output is adapted for Windows when needed.
func NewCLIHandler(w io.Writer, opts ...CLIHandlerOption) *CLIHandler {
	option := newOption()
	for _, opt := range opts {
		if opt.apply != nil {
			opt.apply(&option)
		}
	}
	writer := newWriter(w)
	source := newSource(option.source)
	config := newConfig(&option, writer.terminal)
	return &CLIHandler{
		writer: writer,
		source: source,
		config: config,
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
// It ignores the context and does not check the minimum level.
func (o *CLIHandler) Handle(_ context.Context, record slog.Record) error {
	if o.config.splitLines && strings.Contains(record.Message, "\n") {
		return o.handleLines(record)
	}
	config := &o.config
	s := acquireState()
	o.appendPrefix(s, record)
	s.line.appendMessage(record.Message, &config.message)
	o.appendAttrs(s, record)
	s.line.appendNewline()
	err := o.writer.write(s.line.buf)
	releaseState(s)
	return err
}

// WithAttrs returns a handler with attrs appended to its existing attributes.
// Attributes are resolved, replaced, and formatted once under the current groups.
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

// WithLabel returns a handler that displays label before the message, replacing
// any existing label. An empty label omits the label and its decoration.
// The label keeps the receiver's [LabelStyle]. It is written verbatim and should
// contain only trusted text.
func (o *CLIHandler) WithLabel(label string) slog.Handler {
	h2 := *o
	var value string
	if label != "" {
		value = pad(label, o.config.label.width)
	}
	h2.config.label.value = value
	return &h2
}

// handleLines formats shared components once and writes all lines together.
func (o *CLIHandler) handleLines(record slog.Record) error {
	var s *state
	var prefixEnd, attrsStart, attrsEnd int
	for line := range strings.Lines(record.Message) {
		if strings.HasSuffix(line, "\n") {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if s == nil {
			s = acquireState()
			o.appendPrefix(s, record)
			prefixEnd = len(s.line.buf)
			s.line.appendMessage(line, &o.config.message)
			attrsStart = len(s.line.buf)
			o.appendAttrs(s, record)
			attrsEnd = len(s.line.buf)
		} else {
			s.line.buf = append(s.line.buf, s.line.buf[:prefixEnd]...)
			s.line.appendMessage(line, &o.config.message)
			s.line.buf = append(s.line.buf, s.line.buf[attrsStart:attrsEnd]...)
		}
		s.line.appendNewline()
	}
	if s == nil {
		return nil
	}
	err := o.writer.write(s.line.buf)
	releaseState(s)
	return err
}

// appendPrefix formats the built-in components preceding the message.
func (o *CLIHandler) appendPrefix(s *state, record slog.Record) {
	config := &o.config
	if config.time.enabled && !record.Time.IsZero() {
		s.line.appendTime(record.Time, &config.time)
	}
	s.line.appendLevel(record.Level, config.level.texts)
	if o.source != nil {
		if text := o.source.resolve(record.PC); text != nil {
			s.line.appendSource(text, &config.source)
		}
	}
	if config.label.value != "" {
		s.line.appendLabel(&config.label)
	}
}

// appendAttrs formats cached and record attributes.
func (o *CLIHandler) appendAttrs(s *state, record slog.Record) {
	s.line.appendCachedAttrs(o.attrs)
	if record.NumAttrs() == 0 {
		return
	}
	s.attr.prepare(o.groups)
	record.Attrs(func(attr slog.Attr) bool {
		s.attr.resolve(attr, o.config.attr.replacer, func(attr slog.Attr, kind slog.Kind, path []byte) {
			s.line.appendAttr(attr, kind, path, &o.config.attr)
		})
		return true
	})
}
