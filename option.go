package log

import (
	"log/slog"
	"maps"
)

// CLIHandlerOption configures a [CLIHandler]. Its zero value has no effect.
type CLIHandlerOption struct {
	apply func(*option)
}

// WithTime enables the built-in timestamp, which is disabled by default.
// A record with a zero Time still omits the timestamp and its decoration.
func WithTime() CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			o.hasTime = true
		},
	}
}

// WithTimeLayout sets the layout for built-in time and ordinary time attributes.
// It applies to ordinary attributes even without [WithTime]. The default and
// the fallback for an empty layout are [time.RFC3339].
// The built-in timestamp is written verbatim; ordinary time attributes are
// escaped when necessary. The layout should therefore contain only trusted text.
func WithTimeLayout(layout string) CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			o.timeLayout = layout
		},
	}
}

// WithLevel sets the minimum level accepted by the handler.
// The default is [slog.LevelInfo]. A nil level leaves the current setting unchanged.
// The handler calls level.Level for each enabled check, so a [slog.LevelVar]
// can change the threshold after construction.
// Custom level implementations must support concurrent calls to Level.
func WithLevel(level slog.Leveler) CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			if level != nil {
				o.level = level
			}
		},
	}
}

// WithSourcePath enables the source file path and line number.
// Source output is disabled by default and omitted when a record has no location.
// The path is used as reported by the runtime, without shortening, and escaped
// when necessary. If multiple source options are specified, the last one wins.
func WithSourcePath() CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			o.source = func(source *slog.Source) string {
				return source.File
			}
		},
	}
}

// WithSourceFunction enables the source function name and line number.
// Source output is disabled by default and omitted when a record has no location.
// The name is used as reported by the runtime, without shortening, and escaped
// when necessary. If multiple source options are specified, the last one wins.
func WithSourceFunction() CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			o.source = func(source *slog.Source) string {
				return source.Function
			}
		},
	}
}

// WithLabel sets the label displayed before the message.
// An empty label, the default, omits the label and its decoration.
// The label is written verbatim and should contain only trusted text.
func WithLabel(label string) CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			o.label = label
		},
	}
}

// WithAttrReplacer sets the replacer for ordinary attributes.
// By default, attributes are not replaced. A nil replacer leaves the current
// setting unchanged. See [AttrReplacer] for its scope and concurrency requirements.
func WithAttrReplacer(replacer AttrReplacer) CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			if replacer != nil {
				o.attrReplacer = replacer
			}
		},
	}
}

// WithStyle sets the output style, replacing [DefaultStyle].
// A nil s leaves the current setting unchanged.
func WithStyle(s *Style) CLIHandlerOption {
	return CLIHandlerOption{
		apply: func(o *option) {
			if s != nil {
				o.style = s
			}
		},
	}
}

// StyleOption configures a [Style]. Its zero value has no effect.
type StyleOption struct {
	apply func(*Style)
}

// WithTimeStyle replaces the built-in timestamp's decoration.
// It does not enable time output or change the time layout.
func WithTimeStyle(value TimeStyle) StyleOption {
	return StyleOption{
		apply: func(style *Style) {
			style.time = value
		},
	}
}

// WithLevelStyle merges value into the style's level styles, including custom levels.
// Entries with matching levels are replaced; other entries are preserved.
// The entries are copied when the option is applied. A nil or empty map has no effect.
func WithLevelStyle(value map[slog.Level]LevelStyle) StyleOption {
	return StyleOption{
		apply: func(style *Style) {
			maps.Copy(style.level, value)
		},
	}
}

// WithSourceStyle replaces the built-in source's decoration.
// It does not enable source output or select between file and function names.
func WithSourceStyle(value SourceStyle) StyleOption {
	return StyleOption{
		apply: func(style *Style) {
			style.source = value
		},
	}
}

// WithLabelStyle replaces the label's decoration and alignment.
// It does not set the label text.
func WithLabelStyle(value LabelStyle) StyleOption {
	return StyleOption{
		apply: func(style *Style) {
			style.label = value
		},
	}
}

// WithMessageStyle replaces the message's decoration.
func WithMessageStyle(value MessageStyle) StyleOption {
	return StyleOption{
		apply: func(style *Style) {
			style.message = value
		},
	}
}

// WithAttrStyle replaces the colors and key-value separator for ordinary attributes.
func WithAttrStyle(value AttrStyle) StyleOption {
	return StyleOption{
		apply: func(style *Style) {
			style.attr = value
		},
	}
}

// option holds handler options before they are compiled into a config.
type option struct {
	level        slog.Leveler
	label        string
	source       func(*slog.Source) string
	hasTime      bool
	timeLayout   string
	attrReplacer AttrReplacer
	style        *Style
}

// newOption returns the default handler options.
func newOption() option {
	return option{
		level: slog.LevelInfo,
		style: DefaultStyle(),
	}
}
