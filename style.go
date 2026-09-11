package log

import (
	"log/slog"
	"maps"
)

// Style holds immutable decoration settings for log output.
// Use [NewStyle] or [DefaultStyle] for the default appearance, and [Style.With]
// to derive a style without modifying the original.
//
// The zero value adds no decoration and uses an empty attribute separator.
// Style does not enable or disable built-in output. Literal style text is
// written verbatim and should contain only trusted text.
type Style struct {
	time    TimeStyle
	level   map[slog.Level]LevelStyle
	source  SourceStyle
	label   LabelStyle
	message MessageStyle
	attr    AttrStyle
}

// NewStyle returns [DefaultStyle] with opts applied in order.
// Zero-valued options are ignored.
func NewStyle(opts ...StyleOption) *Style {
	style := DefaultStyle()
	for _, opt := range opts {
		if opt.apply != nil {
			opt.apply(style)
		}
	}
	return style
}

// DefaultStyle returns a new style with DBG, INF, WRN, and ERR level labels,
// angle brackets around source locations, and "=" between attribute keys and values.
// Its colors are used only for terminal output.
func DefaultStyle() *Style {
	affixColor := NewColor(CodeFgHiBlack)
	return &Style{
		level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text:  "DBG",
				Color: NewColor(38, 2, 95, 95, 255, CodeBold),
			},
			slog.LevelInfo: {
				Text:  "INF",
				Color: NewColor(38, 2, 95, 255, 215, CodeBold),
			},
			slog.LevelWarn: {
				Text:  "WRN",
				Color: NewColor(38, 2, 215, 255, 135, CodeBold),
			},
			slog.LevelError: {
				Text:  "ERR",
				Color: NewColor(38, 2, 255, 95, 135, CodeBold),
			},
		},
		source: SourceStyle{
			Prefix: AffixStyle{
				Text:  "<",
				Color: affixColor,
			},
			Suffix: AffixStyle{
				Text:  ">",
				Color: affixColor,
			},
			Color: NewColor(CodeFgHiBlack, CodeUnderline),
		},
		label: LabelStyle{
			Color: NewColor(CodeFgHiBlack, CodeBold),
		},
		attr: AttrStyle{
			KeyColor:  NewColor(CodeFgHiBlack),
			Separator: "=",
		},
	}
}

// With returns an independent copy of the style with opts applied in order.
// It also works on a zero-valued Style. Zero-valued options are ignored.
func (o *Style) With(opts ...StyleOption) *Style {
	style := *o
	style.level = maps.Clone(o.level)
	if style.level == nil {
		style.level = make(map[slog.Level]LevelStyle)
	}
	for _, opt := range opts {
		if opt.apply != nil {
			opt.apply(&style)
		}
	}
	return &style
}

// TimeStyle decorates the built-in timestamp.
// Prefix and Suffix surround the timestamp; Color applies to the timestamp itself.
// A nil Color adds no color. The layout is set by [WithTimeLayout].
type TimeStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	Color  *Color
}

// LevelStyle decorates a built-in log level.
// Prefix and Suffix surround the level text; Color applies to the text and padding.
// A nil Color adds no color.
type LevelStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	// Text replaces the level name. An empty Text uses slog.Level.String.
	Text  string
	Color *Color
	// Width is the minimum display width, excluding Prefix and Suffix.
	// Text is centered without truncation; an odd padding column goes on the right.
	// A non-positive Width adds no padding.
	Width int
}

// SourceStyle decorates the built-in source location.
// Prefix and Suffix surround the location; Color applies to the location itself.
// A nil Color adds no color. Source selection is controlled by [WithSourcePath]
// and [WithSourceFunction].
type SourceStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	Color  *Color
}

// LabelStyle decorates the text set by [WithLabel].
// Prefix and Suffix surround the label; Color applies to the label and padding.
// A nil Color adds no color. An empty label omits all decoration.
type LabelStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	Color  *Color
	// Width is the minimum display width, excluding Prefix and Suffix.
	// The label is centered without truncation; an odd padding column goes on the right.
	// A non-positive Width adds no padding.
	Width int
}

// MessageStyle decorates the log message.
// Prefix and Suffix surround the message; Color applies to the message itself.
// A nil Color adds no color.
type MessageStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	Color  *Color
}

// AttrStyle decorates ordinary attributes without affecting built-in output.
type AttrStyle struct {
	// KeyColor applies to the group path, key, and separator. Nil adds no color.
	KeyColor *Color
	// ValueColor applies to non-empty formatted values. Nil adds no color.
	ValueColor *Color
	// Separator is written verbatim between the key and value.
	// An empty Separator adds nothing; DefaultStyle uses "=".
	Separator string
}

// AffixStyle decorates a literal prefix or suffix.
// Text is written verbatim, and Color applies only to Text.
// Empty Text produces no output; a nil Color adds no color.
type AffixStyle struct {
	Text  string
	Color *Color
}
