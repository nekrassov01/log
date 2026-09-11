package log

import (
	"log/slog"
	"time"

	"github.com/mattn/go-runewidth"
)

// pad is the space used to align level and label text.
const pad byte = ' '

// config holds precomputed output settings shared by derived handlers.
// The level threshold and attribute replacer may refer to caller-managed state.
type config struct {
	time    timeConfig
	level   levelConfig
	source  sourceConfig
	label   labelConfig
	message messageConfig
	attr    attrConfig
}

// newConfig precomputes output settings, fixing color use and the shared time layout.
func newConfig(option *option, colored bool) config {
	style := option.style
	layout := option.timeLayout
	if layout == "" {
		layout = time.RFC3339
	}
	return config{
		time:    newTimeConfig(style.time, layout, option.hasTime, colored),
		level:   newLevelConfig(option.level, style.level, colored),
		source:  newSourceConfig(style.source, option.source, colored),
		label:   newLabelConfig(option.label, style.label, colored),
		message: newMessageConfig(style.message, colored),
		attr:    newAttrConfig(style.attr, option.attrReplacer, layout, colored),
	}
}

// timeConfig holds the encoded time configuration.
type timeConfig struct {
	prefix  []byte
	suffix  []byte
	layout  string
	enabled bool
}

// newTimeConfig compiles the time configuration.
func newTimeConfig(style TimeStyle, layout string, enabled, colored bool) timeConfig {
	prefix, suffix := encodeAffixes(style.Prefix, style.Suffix, style.Color, colored)
	return timeConfig{
		prefix:  prefix,
		suffix:  suffix,
		layout:  layout,
		enabled: enabled,
	}
}

// levelConfig holds the dynamic threshold and preformatted level labels.
// Levels missing from texts are formatted when a record is written.
type levelConfig struct {
	threshold slog.Leveler
	texts     map[slog.Level][]byte
}

// newLevelConfig preformats each configured level, falling back to its slog name
// when Text is empty. The threshold remains dynamic.
func newLevelConfig(threshold slog.Leveler, styles map[slog.Level]LevelStyle, colored bool) levelConfig {
	texts := make(map[slog.Level][]byte, len(styles))
	for lv, style := range styles {
		name := style.Text
		if name == "" {
			name = lv.String()
		}
		var text []byte
		text = style.Prefix.Color.appendText(text, style.Prefix.Text, colored)
		if style.Width > 0 {
			left, right := align(name, style.Width)
			text = style.Color.appendPrefix(text, colored)
			for range left {
				text = append(text, pad)
			}
			text = append(text, name...)
			for range right {
				text = append(text, pad)
			}
			text = style.Color.appendSuffix(text, colored)
		} else {
			text = style.Color.appendText(text, name, colored)
		}
		text = style.Suffix.Color.appendText(text, style.Suffix.Text, colored)
		texts[lv] = text
	}
	return levelConfig{
		threshold: threshold,
		texts:     texts,
	}
}

// sourceConfig holds the encoded source configuration.
type sourceConfig struct {
	prefix []byte
	suffix []byte
	value  func(*slog.Source) string
}

// newSourceConfig compiles the source configuration.
func newSourceConfig(style SourceStyle, value func(*slog.Source) string, colored bool) sourceConfig {
	prefix, suffix := encodeAffixes(style.Prefix, style.Suffix, style.Color, colored)
	return sourceConfig{
		prefix: prefix,
		suffix: suffix,
		value:  value,
	}
}

// labelConfig holds the encoded label configuration.
type labelConfig struct {
	prefix []byte
	suffix []byte
	value  string
}

// newLabelConfig precomputes label decoration and padding.
// An empty label discards its decoration.
func newLabelConfig(label string, style LabelStyle, colored bool) labelConfig {
	if label == "" {
		return labelConfig{}
	}
	var prefix []byte
	prefix = style.Prefix.Color.appendText(prefix, style.Prefix.Text, colored)
	var suffix []byte
	if style.Width > 0 {
		left, right := align(label, style.Width)
		prefix = style.Color.appendPrefix(prefix, colored)
		for range left {
			prefix = append(prefix, pad)
		}
		for range right {
			suffix = append(suffix, pad)
		}
		suffix = style.Color.appendSuffix(suffix, colored)
	} else {
		prefix = style.Color.appendPrefix(prefix, colored)
		suffix = style.Color.appendSuffix(suffix, colored)
	}
	suffix = style.Suffix.Color.appendText(suffix, style.Suffix.Text, colored)
	return labelConfig{
		prefix: prefix,
		value:  label,
		suffix: suffix,
	}
}

// messageConfig holds the encoded message configuration.
type messageConfig struct {
	prefix []byte
	suffix []byte
}

// newMessageConfig compiles the message configuration.
func newMessageConfig(style MessageStyle, colored bool) messageConfig {
	prefix, suffix := encodeAffixes(style.Prefix, style.Suffix, style.Color, colored)
	return messageConfig{
		prefix: prefix,
		suffix: suffix,
	}
}

// attrConfig holds the ordinary attribute configuration.
type attrConfig struct {
	replacer    AttrReplacer
	separator   string
	timeLayout  string
	keyPrefix   []byte
	keySuffix   []byte
	valuePrefix []byte
	valueSuffix []byte
}

// newAttrConfig compiles the ordinary attribute configuration.
func newAttrConfig(style AttrStyle, replacer AttrReplacer, layout string, colored bool) attrConfig {
	result := attrConfig{
		replacer:   replacer,
		separator:  style.Separator,
		timeLayout: layout,
	}
	if colored && style.KeyColor != nil {
		result.keyPrefix = style.KeyColor.prefix
		result.keySuffix = style.KeyColor.suffix
	}
	if colored && style.ValueColor != nil {
		result.valuePrefix = style.ValueColor.prefix
		result.valueSuffix = style.ValueColor.suffix
	}
	return result
}

// encodeAffixes returns the bytes before and after a dynamic value.
// Each affix uses its own color; color applies only between the affixes.
func encodeAffixes(prefix, suffix AffixStyle, color *Color, enabled bool) ([]byte, []byte) {
	var before []byte
	before = prefix.Color.appendText(before, prefix.Text, enabled)
	before = color.appendPrefix(before, enabled)
	var after []byte
	after = color.appendSuffix(after, enabled)
	after = suffix.Color.appendText(after, suffix.Text, enabled)
	return before, after
}

// align returns left and right padding to center text within width display columns.
// It never truncates text and places an extra column of odd padding on the right.
func align(text string, width int) (int, int) {
	padding := width - runewidth.StringWidth(text)
	if padding <= 0 {
		return 0, 0
	}
	left := padding / 2
	return left, padding - left
}
