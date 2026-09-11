package log

import (
	"log/slog"
	"strconv"
	"time"
)

// sep separates output components in a log line.
const sep byte = ' '

// lineState owns the output buffer and separator state for one log line.
type lineState struct {
	buf   []byte
	wrote bool
}

// appendTime adds an unescaped timestamp and its decoration.
// The caller checks that time output is enabled and the timestamp is non-zero.
func (o *lineState) appendTime(value time.Time, config *timeConfig) {
	buf := o.appendSeparator()
	buf = append(buf, config.prefix...)
	buf = value.AppendFormat(buf, config.layout)
	buf = append(buf, config.suffix...)
	o.buf = buf
	o.wrote = true
}

// appendLevel adds a preformatted level label or falls back to slog's level notation.
func (o *lineState) appendLevel(level slog.Level, texts map[slog.Level][]byte) {
	buf := o.appendSeparator()
	if text := texts[level]; len(text) > 0 {
		buf = append(buf, text...)
	} else {
		var text string
		var offset slog.Level
		switch {
		case level < slog.LevelInfo:
			text = slog.LevelDebug.String()
			offset = level - slog.LevelDebug
		case level < slog.LevelWarn:
			text = slog.LevelInfo.String()
			offset = level - slog.LevelInfo
		case level < slog.LevelError:
			text = slog.LevelWarn.String()
			offset = level - slog.LevelWarn
		default:
			text = slog.LevelError.String()
			offset = level - slog.LevelError
		}
		buf = append(buf, text...)
		if offset != 0 {
			if offset > 0 {
				buf = append(buf, '+')
			}
			buf = strconv.AppendInt(buf, int64(offset), 10)
		}
	}
	o.buf = buf
	o.wrote = true
}

// appendSource adds pre-escaped source text and its decoration.
// The caller resolves the location and omits unavailable sources.
func (o *lineState) appendSource(text []byte, config *sourceConfig) {
	buf := o.appendSeparator()
	buf = append(buf, config.prefix...)
	buf = append(buf, text...)
	buf = append(buf, config.suffix...)
	o.buf = buf
	o.wrote = true
}

// appendLabel adds the configured label and its decoration.
// The caller omits empty labels.
func (o *lineState) appendLabel(config *labelConfig) {
	buf := o.appendSeparator()
	buf = append(buf, config.prefix...)
	buf = append(buf, config.value...)
	buf = append(buf, config.suffix...)
	o.buf = buf
	o.wrote = true
}

// appendMessage adds the escaped message and its decoration, even for an empty message.
func (o *lineState) appendMessage(message string, config *messageConfig) {
	buf := o.appendSeparator()
	buf = append(buf, config.prefix...)
	buf = escapeMessage(buf, message)
	buf = append(buf, config.suffix...)
	o.buf = buf
	o.wrote = true
}

// appendCachedAttrs adds pre-rendered handler attributes to the line.
func (o *lineState) appendCachedAttrs(attrs []byte) {
	if len(attrs) == 0 {
		return
	}
	buf := o.appendSeparator()
	buf = append(buf, attrs...)
	o.buf = buf
	o.wrote = true
}

// appendAttr formats an ordinary leaf attribute with its pre-escaped group path.
// The caller must resolve and replace attr first, with kind matching its value.
func (o *lineState) appendAttr(attr slog.Attr, kind slog.Kind, path []byte, config *attrConfig) {
	value := attr.Value
	buf := o.appendSeparator()
	styledKey := len(config.keyPrefix) > 0 && (len(path) > 0 || attr.Key != "" || config.separator != "")
	if styledKey {
		buf = append(buf, config.keyPrefix...)
	}
	buf = append(buf, path...)
	buf, _ = escapeText(buf, attr.Key)
	buf = append(buf, config.separator...)
	if styledKey {
		buf = append(buf, config.keySuffix...)
	}
	valuePrefixLen := len(config.valuePrefix)
	if valuePrefixLen > 0 {
		buf = append(buf, config.valuePrefix...)
	}
	valueStart := len(buf)
	switch kind {
	case slog.KindString:
		buf, _ = escapeText(buf, value.String())
	case slog.KindInt64:
		buf = strconv.AppendInt(buf, value.Int64(), 10)
	case slog.KindUint64:
		buf = strconv.AppendUint(buf, value.Uint64(), 10)
	case slog.KindFloat64:
		buf = strconv.AppendFloat(buf, value.Float64(), 'g', -1, 64)
	case slog.KindBool:
		buf = strconv.AppendBool(buf, value.Bool())
	case slog.KindTime:
		// Common layouts fit on the stack; longer formatted values can grow as needed.
		var scratch [64]byte
		text := value.Time().AppendFormat(scratch[:0], config.timeLayout)
		buf, _ = escapeText(buf, string(text))
	case slog.KindDuration:
		buf = append(buf, value.Duration().String()...)
	default:
		buf, _ = escapeText(buf, value.String())
	}
	if valuePrefixLen > 0 {
		if len(buf) == valueStart {
			buf = buf[:valueStart-valuePrefixLen]
		} else {
			buf = append(buf, config.valueSuffix...)
		}
	}
	o.buf = buf
	o.wrote = true
}

// appendNewline terminates the rendered line.
func (o *lineState) appendNewline() {
	o.buf = append(o.buf, '\n')
}

// appendSeparator returns a buffer with any required separator without updating state.
func (o *lineState) appendSeparator() []byte {
	if o.wrote {
		return append(o.buf, sep)
	}
	return o.buf
}

// reset clears the line and separator state while retaining buffer capacity.
func (o *lineState) reset() {
	o.buf = o.buf[:0]
	o.wrote = false
}
