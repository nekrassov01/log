package log

import (
	"log/slog"
	"time"

	"github.com/mattn/go-runewidth"
)

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

// newLevelConfig preformats styled levels while retaining the dynamic threshold.
func newLevelConfig(threshold slog.Leveler, styles map[slog.Level]LevelStyle, colored bool) levelConfig {
	texts := make(map[slog.Level][]byte, len(styles))
	for lv, style := range styles {
		name := style.Text
		if name == "" {
			name = lv.String()
		}
		prefix := style.Prefix
		suffix := style.Suffix
		color := style.Color
		var text []byte
		text = prefix.Color.appendText(text, prefix.Text, colored)
		text = color.appendText(text, pad(name, style.Width), colored)
		text = suffix.Color.appendText(text, suffix.Text, colored)
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
// The value includes its padding, so the decoration does not depend on the label.
type labelConfig struct {
	prefix []byte
	suffix []byte
	value  string
	width  int
}

// newLabelConfig compiles the label configuration.
// An empty label keeps an empty value so that the label and its decoration are omitted.
func newLabelConfig(label string, style LabelStyle, colored bool) labelConfig {
	prefix, suffix := encodeAffixes(style.Prefix, style.Suffix, style.Color, colored)
	result := labelConfig{
		prefix: prefix,
		suffix: suffix,
		width:  style.Width,
	}
	if label != "" {
		result.value = pad(label, style.Width)
	}
	return result
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

// pad returns text centered within width display columns using align.
func pad(text string, width int) string {
	left, right := align(text, width)
	buf := make([]byte, 0, left+len(text)+right)
	for range left {
		buf = append(buf, ' ')
	}
	buf = append(buf, text...)
	for range right {
		buf = append(buf, ' ')
	}
	return string(buf)
}
