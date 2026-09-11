package log

import (
	"log/slog"
	"testing"
)

func TestWithTime(t *testing.T) {
	type want struct {
		val bool
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "enabled",
			want: want{
				val: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			WithTime().apply(&o)
			assertValue(t, o.hasTime, test.want.val, "time")
		})
	}
}

func TestWithTimeLayout(t *testing.T) {
	type args struct {
		layout string
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
			name: "empty",
		},
		{
			name: "value",
			args: args{
				layout: "value",
			},
			want: want{
				val: "value",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			WithTimeLayout(test.args.layout).apply(&o)
			assertValue(t, o.timeLayout, test.want.val, "timeLayout")
		})
	}
}

func TestWithLevel(t *testing.T) {
	type args struct {
		level slog.Leveler
	}
	type want struct {
		val slog.Leveler
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil preserves default",
			want: want{
				val: slog.LevelInfo,
			},
		},
		{
			name: "debug",
			args: args{
				level: slog.LevelDebug,
			},
			want: want{
				val: slog.LevelDebug,
			},
		},
		{
			name: "custom",
			args: args{
				level: slog.Level(-8),
			},
			want: want{
				val: slog.Level(-8),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			WithLevel(test.args.level).apply(&o)
			assertValue(t, o.level, test.want.val, "level")
		})
	}
}

func TestWithSourcePath(t *testing.T) {
	type want struct {
		val string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "select",
			want: want{
				val: "file.go",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			WithSourcePath().apply(&o)
			got := o.source(&slog.Source{
				File:     "file.go",
				Function: "pkg.Func",
			})
			assertValue(t, got, test.want.val, "source")
		})
	}
}

func TestWithSourceFunction(t *testing.T) {
	type want struct {
		val string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "select",
			want: want{
				val: "pkg.Func",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			WithSourceFunction().apply(&o)
			got := o.source(&slog.Source{
				File:     "file.go",
				Function: "pkg.Func",
			})
			assertValue(t, got, test.want.val, "source")
		})
	}
}

func TestWithLabel(t *testing.T) {
	type args struct {
		label string
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
			name: "empty",
		},
		{
			name: "value",
			args: args{
				label: "value",
			},
			want: want{
				val: "value",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			WithLabel(test.args.label).apply(&o)
			assertValue(t, o.label, test.want.val, "label")
		})
	}
}

func TestWithAttrReplacer(t *testing.T) {
	type args struct {
		replacer AttrReplacer
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
			name: "nil preserves previous",
			want: want{
				val: "before",
			},
		},
		{
			name: "replace",
			args: args{
				replacer: func(_ []string, a slog.Attr) slog.Attr {
					return slog.String(a.Key, "after")
				},
			},
			want: want{
				val: "after",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			o.attrReplacer = func(_ []string, a slog.Attr) slog.Attr {
				return slog.String(a.Key, "before")
			}
			WithAttrReplacer(test.args.replacer).apply(&o)
			got := o.attrReplacer(nil, slog.String("k", "v"))
			assertValue(t, got.Value.String(), test.want.val, "replacement")
		})
	}
}

func TestWithStyle(t *testing.T) {
	type args struct {
		style *Style
	}
	type want struct {
		val *Style
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil preserves previous",
			want: want{
				val: &Style{},
			},
		},
		{
			name: "replace",
			args: args{
				style: &Style{
					attr: AttrStyle{
						Separator: ":",
					},
				},
			},
			want: want{
				val: &Style{
					attr: AttrStyle{
						Separator: ":",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			o.style = &Style{}
			WithStyle(test.args.style).apply(&o)
			assertValue(t, o.style, test.want.val, "style")
		})
	}
}

func TestWithTimeStyle(t *testing.T) {
	type args struct {
		value TimeStyle
	}
	type want struct {
		val TimeStyle
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
			name: "replace",
			args: args{
				value: TimeStyle{
					Color: NewColor(CodeFgRed),
				},
			},
			want: want{
				val: TimeStyle{
					Color: NewColor(CodeFgRed),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := Style{}
			WithTimeStyle(test.args.value).apply(&o)
			assertValue(t, o.time, test.want.val, "time")
		})
	}
}

func TestWithLevelStyle(t *testing.T) {
	type args struct {
		value map[slog.Level]LevelStyle
	}
	type want struct {
		val map[slog.Level]LevelStyle
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil",
			want: want{
				val: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "old",
					},
				},
			},
		},
		{
			name: "merge",
			args: args{
				value: map[slog.Level]LevelStyle{
					slog.LevelWarn: {
						Text: "warn",
					},
				},
			},
			want: want{
				val: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "old",
					},
					slog.LevelWarn: {
						Text: "warn",
					},
				},
			},
		},
		{
			name: "overwrite",
			args: args{
				value: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "new",
					},
				},
			},
			want: want{
				val: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "new",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := Style{
				level: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "old",
					},
				},
			}
			WithLevelStyle(test.args.value).apply(&o)
			assertValue(t, o.level, test.want.val, "levels")
		})
	}
}

func TestWithSourceStyle(t *testing.T) {
	type args struct {
		value SourceStyle
	}
	type want struct {
		val SourceStyle
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
			name: "replace",
			args: args{
				value: SourceStyle{
					Color: NewColor(CodeFgRed),
				},
			},
			want: want{
				val: SourceStyle{
					Color: NewColor(CodeFgRed),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := Style{}
			WithSourceStyle(test.args.value).apply(&o)
			assertValue(t, o.source, test.want.val, "source")
		})
	}
}

func TestWithLabelStyle(t *testing.T) {
	type args struct {
		value LabelStyle
	}
	type want struct {
		val LabelStyle
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
			name: "replace",
			args: args{
				value: LabelStyle{
					Color: NewColor(CodeFgRed),
				},
			},
			want: want{
				val: LabelStyle{
					Color: NewColor(CodeFgRed),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := Style{}
			WithLabelStyle(test.args.value).apply(&o)
			assertValue(t, o.label, test.want.val, "label")
		})
	}
}

func TestWithMessageStyle(t *testing.T) {
	type args struct {
		value MessageStyle
	}
	type want struct {
		val MessageStyle
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
			name: "replace",
			args: args{
				value: MessageStyle{
					Color: NewColor(CodeFgRed),
				},
			},
			want: want{
				val: MessageStyle{
					Color: NewColor(CodeFgRed),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := Style{}
			WithMessageStyle(test.args.value).apply(&o)
			assertValue(t, o.message, test.want.val, "message")
		})
	}
}

func TestWithAttrStyle(t *testing.T) {
	type args struct {
		value AttrStyle
	}
	type want struct {
		val AttrStyle
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
			name: "replace",
			args: args{
				value: AttrStyle{
					Separator: ":",
				},
			},
			want: want{
				val: AttrStyle{
					Separator: ":",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := Style{}
			WithAttrStyle(test.args.value).apply(&o)
			assertValue(t, o.attr, test.want.val, "attr")
		})
	}
}

func Test_newOption(t *testing.T) {
	type want struct {
		level      slog.Leveler
		label      string
		hasTime    bool
		timeLayout string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "defaults",
			want: want{
				level: slog.LevelInfo,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newOption()
			assertValue(t, o.level, test.want.level, "level")
			assertValue(t, o.label, test.want.label, "label")
			assertValue(t, o.hasTime, test.want.hasTime, "time")
			assertValue(t, o.timeLayout, test.want.timeLayout, "layout")
			assertValue(t, o.source == nil, true, "source disabled")
			assertValue(t, o.attrReplacer == nil, true, "replacer absent")
			assertValue(t, o.style, DefaultStyle(), "default style")
		})
	}
}
