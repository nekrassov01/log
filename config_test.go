package log

import (
	"log/slog"
	"testing"
	"time"
)

func Test_newConfig(t *testing.T) {
	type args struct {
		option  option
		colored bool
	}
	type want struct {
		layout string
		time   bool
		level  slog.Leveler
		label  string
		source bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "defaults",
			args: args{
				option: newOption(),
			},
			want: want{
				layout: time.RFC3339,
				level:  slog.LevelInfo,
			},
		},
		{
			name: "custom",
			args: args{
				option: option{
					level:      slog.LevelWarn,
					label:      "APP",
					hasTime:    true,
					timeLayout: time.Kitchen,
					source: func(s *slog.Source) string {
						return s.Function
					},
					style: DefaultStyle(),
				},
				colored: true,
			},
			want: want{
				layout: time.Kitchen,
				time:   true,
				level:  slog.LevelWarn,
				label:  "APP",
				source: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newConfig(&test.args.option, test.args.colored)
			assertValue(t, got.time.layout, test.want.layout, "built-in layout")
			assertValue(t, got.attr.timeLayout, test.want.layout, "attribute layout")
			assertValue(t, got.time.enabled, test.want.time, "time")
			assertValue(t, got.level.threshold, test.want.level, "threshold")
			assertValue(t, got.label.value, test.want.label, "label")
			assertValue(t, got.source.value != nil, test.want.source, "source")
		})
	}
}

func Test_newTimeConfig(t *testing.T) {
	type args struct {
		style   TimeStyle
		layout  string
		enabled bool
		colored bool
	}
	type want struct {
		prefix  string
		suffix  string
		layout  string
		enabled bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "zero",
		},
		{
			name: "decorated without color",
			args: args{
				style: TimeStyle{
					Prefix: AffixStyle{
						Text: "<",
					},
					Suffix: AffixStyle{
						Text: ">",
					},
					Color: NewColor(CodeFgRed),
				},
				layout:  time.RFC3339,
				enabled: true,
			},
			want: want{
				prefix:  "<",
				suffix:  ">",
				layout:  time.RFC3339,
				enabled: true,
			},
		},
		{
			name: "colored",
			args: args{
				style: TimeStyle{
					Prefix: AffixStyle{
						Text: "<",
					},
					Suffix: AffixStyle{
						Text: ">",
					},
					Color: NewColor(CodeFgRed),
				},
				colored: true,
			},
			want: want{
				prefix: "<\x1b[31m",
				suffix: "\x1b[0m>",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newTimeConfig(test.args.style, test.args.layout, test.args.enabled, test.args.colored)
			assertBytes(t, got.prefix, test.want.prefix, "prefix")
			assertBytes(t, got.suffix, test.want.suffix, "suffix")
			assertValue(t, got.layout, test.want.layout, "layout")
			assertValue(t, got.enabled, test.want.enabled, "enabled")
		})
	}
}

func Test_newLevelConfig(t *testing.T) {
	type args struct {
		threshold slog.Leveler
		styles    map[slog.Level]LevelStyle
		colored   bool
	}
	type want struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil",
		},
		{
			name: "plain",
			args: args{
				threshold: slog.LevelWarn,
				styles: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "INF",
					},
				},
			},
			want: want{
				val: "INF",
			},
		},
		{
			name: "empty text uses name",
			args: args{
				styles: map[slog.Level]LevelStyle{
					slog.LevelInfo: {},
				},
			},
			want: want{
				val: "INFO",
			},
		},
		{
			name: "padded",
			args: args{
				styles: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text:  "I",
						Width: 4,
						Prefix: AffixStyle{
							Text: "[",
						},
						Suffix: AffixStyle{
							Text: "]",
						},
					},
				},
			},
			want: want{
				val: "[ I  ]",
			},
		},
		{
			name: "colored padded",
			args: args{
				styles: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text:  "I",
						Width: 3,
						Color: NewColor(CodeFgRed),
					},
				},
				colored: true,
			},
			want: want{
				val: "\x1b[31m I \x1b[0m",
			},
		},
		{
			name: "width smaller",
			args: args{
				styles: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text:  "LONG",
						Width: 1,
					},
				},
			},
			want: want{
				val: "LONG",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newLevelConfig(test.args.threshold, test.args.styles, test.args.colored)
			assertValue(t, got.threshold, test.args.threshold, "threshold")
			assertBytes(t, got.texts[slog.LevelInfo], test.want.val, "level text")
			assertValue(t, len(got.texts), len(test.args.styles), "level count")
		})
	}
}

func Test_newSourceConfig(t *testing.T) {
	type args struct {
		style   SourceStyle
		value   func(*slog.Source) string
		colored bool
	}
	type want struct {
		prefix  string
		suffix  string
		value   string
		missing bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "zero",
			want: want{
				missing: true,
			},
		},
		{
			name: "decorated without color",
			args: args{
				style: SourceStyle{
					Prefix: AffixStyle{
						Text: "<",
					},
					Suffix: AffixStyle{
						Text: ">",
					},
					Color: NewColor(CodeFgRed),
				},
				value: func(s *slog.Source) string {
					return s.File
				},
			},
			want: want{
				prefix: "<",
				suffix: ">",
				value:  "file.go",
			},
		},
		{
			name: "colored",
			args: args{
				style: SourceStyle{
					Prefix: AffixStyle{
						Text: "<",
					},
					Suffix: AffixStyle{
						Text: ">",
					},
					Color: NewColor(CodeFgRed),
				},
				colored: true,
				value: func(s *slog.Source) string {
					return s.File
				},
			},
			want: want{
				prefix: "<\x1b[31m",
				suffix: "\x1b[0m>",
				value:  "file.go",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newSourceConfig(test.args.style, test.args.value, test.args.colored)
			assertBytes(t, got.prefix, test.want.prefix, "prefix")
			assertBytes(t, got.suffix, test.want.suffix, "suffix")
			assertValue(t, got.value == nil, test.want.missing, "selector absent")
			assertValue(t, testSourceValue(got.value), test.want.value, "selected value")
		})
	}
}

func Test_newLabelConfig(t *testing.T) {
	type args struct {
		label   string
		style   LabelStyle
		colored bool
	}
	type want struct {
		prefix string
		value  string
		suffix string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty ignores decorations",
			args: args{
				style: LabelStyle{
					Prefix: AffixStyle{
						Text: "[",
					},
					Color: NewColor(CodeFgRed),
					Width: 10,
				},
				colored: true,
			},
		},
		{
			name: "plain",
			args: args{
				label: "APP",
			},
			want: want{
				value: "APP",
			},
		},
		{
			name: "padded odd",
			args: args{
				label: "APP",
				style: LabelStyle{
					Prefix: AffixStyle{
						Text: "[",
					},
					Suffix: AffixStyle{
						Text: "]",
					},
					Width: 6,
				},
			},
			want: want{
				prefix: "[ ",
				value:  "APP",
				suffix: "  ]",
			},
		},
		{
			name: "colored padded",
			args: args{
				label: "APP",
				style: LabelStyle{
					Color: NewColor(CodeFgRed),
					Width: 5,
				},
				colored: true,
			},
			want: want{
				prefix: "\x1b[31m ",
				value:  "APP",
				suffix: " \x1b[0m",
			},
		},
		{
			name: "colored",
			args: args{
				label: "APP",
				style: LabelStyle{
					Color: NewColor(CodeFgRed),
				},
				colored: true,
			},
			want: want{
				prefix: "\x1b[31m",
				value:  "APP",
				suffix: "\x1b[0m",
			},
		},
		{
			name: "width too small",
			args: args{
				label: "APP",
				style: LabelStyle{
					Width: 1,
				},
			},
			want: want{
				value: "APP",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newLabelConfig(test.args.label, test.args.style, test.args.colored)
			assertBytes(t, got.prefix, test.want.prefix, "prefix")
			assertValue(t, got.value, test.want.value, "label")
			assertBytes(t, got.suffix, test.want.suffix, "suffix")
		})
	}
}

func Test_newMessageConfig(t *testing.T) {
	type args struct {
		style   MessageStyle
		colored bool
	}
	type want struct {
		prefix string
		suffix string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "zero",
		},
		{
			name: "decorated without color",
			args: args{
				style: MessageStyle{
					Prefix: AffixStyle{
						Text: "<",
					},
					Suffix: AffixStyle{
						Text: ">",
					},
					Color: NewColor(CodeFgRed),
				},
			},
			want: want{
				prefix: "<",
				suffix: ">",
			},
		},
		{
			name: "colored",
			args: args{
				style: MessageStyle{
					Prefix: AffixStyle{
						Text: "<",
					},
					Suffix: AffixStyle{
						Text: ">",
					},
					Color: NewColor(CodeFgRed),
				},
				colored: true,
			},
			want: want{
				prefix: "<\x1b[31m",
				suffix: "\x1b[0m>",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newMessageConfig(test.args.style, test.args.colored)
			assertBytes(t, got.prefix, test.want.prefix, "prefix")
			assertBytes(t, got.suffix, test.want.suffix, "suffix")
		})
	}
}

func Test_newAttrConfig(t *testing.T) {
	type args struct {
		style    AttrStyle
		replacer AttrReplacer
		layout   string
		colored  bool
	}
	type want struct {
		keyPrefix   string
		keySuffix   string
		valuePrefix string
		valueSuffix string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "zero",
		},
		{
			name: "plain",
			args: args{
				style: AttrStyle{
					KeyColor:   NewColor(CodeFgRed),
					ValueColor: NewColor(CodeFgGreen),
					Separator:  ":",
				},
				layout: time.Kitchen,
			},
		},
		{
			name: "colored",
			args: args{
				style: AttrStyle{
					KeyColor:   NewColor(CodeFgRed),
					ValueColor: NewColor(CodeFgGreen),
					Separator:  "=",
				},
				colored: true,
				layout:  time.RFC3339,
				replacer: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			},
			want: want{
				keyPrefix:   "\x1b[31m",
				keySuffix:   "\x1b[0m",
				valuePrefix: "\x1b[32m",
				valueSuffix: "\x1b[0m",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newAttrConfig(test.args.style, test.args.replacer, test.args.layout, test.args.colored)
			assertValue(t, got.separator, test.args.style.Separator, "separator")
			assertValue(t, got.timeLayout, test.args.layout, "layout")
			assertValue(t, got.replacer == nil, test.args.replacer == nil, "replacer absent")
			assertBytes(t, got.keyPrefix, test.want.keyPrefix, "key prefix")
			assertBytes(t, got.keySuffix, test.want.keySuffix, "key suffix")
			assertBytes(t, got.valuePrefix, test.want.valuePrefix, "value prefix")
			assertBytes(t, got.valueSuffix, test.want.valueSuffix, "value suffix")
		})
	}
}

func Test_encodeAffixes(t *testing.T) {
	type args struct {
		prefix  AffixStyle
		suffix  AffixStyle
		color   *Color
		enabled bool
	}
	type want struct {
		before string
		after  string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty",
		},
		{
			name: "disabled",
			args: args{
				prefix: AffixStyle{
					Text:  "<",
					Color: NewColor(CodeFgRed),
				},
				suffix: AffixStyle{
					Text:  ">",
					Color: NewColor(CodeFgGreen),
				},
				color: NewColor(CodeBold),
			},
			want: want{
				before: "<",
				after:  ">",
			},
		},
		{
			name: "colored",
			args: args{
				prefix: AffixStyle{
					Text:  "<",
					Color: NewColor(CodeFgRed),
				},
				suffix: AffixStyle{
					Text:  ">",
					Color: NewColor(CodeFgGreen),
				},
				color:   NewColor(CodeBold),
				enabled: true,
			},
			want: want{
				before: "\x1b[31m<\x1b[0m\x1b[1m",
				after:  "\x1b[0m\x1b[32m>\x1b[0m",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before, after := encodeAffixes(test.args.prefix, test.args.suffix, test.args.color, test.args.enabled)
			assertBytes(t, before, test.want.before, "prefix")
			assertBytes(t, after, test.want.after, "suffix")
		})
	}
}

func Test_align(t *testing.T) {
	type args struct {
		text  string
		width int
	}
	type want struct {
		left  int
		right int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "zero",
		},
		{
			name: "negative",
			args: args{
				text:  "abc",
				width: -1,
			},
		},
		{
			name: "smaller",
			args: args{
				text:  "abc",
				width: 2,
			},
		},
		{
			name: "exact",
			args: args{
				text:  "abc",
				width: 3,
			},
		},
		{
			name: "even padding",
			args: args{
				text:  "abc",
				width: 5,
			},
			want: want{
				left:  1,
				right: 1,
			},
		},
		{
			name: "odd padding",
			args: args{
				text:  "abc",
				width: 6,
			},
			want: want{
				left:  1,
				right: 2,
			},
		},
		{
			name: "wide rune",
			args: args{
				text:  "あ",
				width: 4,
			},
			want: want{
				left:  1,
				right: 1,
			},
		},
		{
			name: "combining",
			args: args{
				text:  "e\u0301",
				width: 3,
			},
			want: want{
				left:  1,
				right: 1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, right := align(test.args.text, test.args.width)
			assertValue(t, left, test.want.left, "left")
			assertValue(t, right, test.want.right, "right")
		})
	}
}
