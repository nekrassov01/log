package log

import (
	"log/slog"
	"maps"
	"testing"
)

func TestNewStyle(t *testing.T) {
	custom := testDefaultStyle()
	custom.attr = AttrStyle{
		Separator: ":",
	}
	type args struct {
		opts []StyleOption
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
			name: "defaults",
			want: want{
				val: testDefaultStyle(),
			},
		},
		{
			name: "zero option",
			args: args{
				opts: []StyleOption{{}},
			},
			want: want{
				val: testDefaultStyle(),
			},
		},
		{
			name: "custom attr",
			args: args{
				opts: []StyleOption{WithAttrStyle(AttrStyle{
					Separator: ":",
				})},
			},
			want: want{
				val: custom,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewStyle(test.args.opts...)
			assertValue(t, got, test.want.val, "style")
		})
	}
}

func TestDefaultStyle(t *testing.T) {
	type want struct {
		val *Style
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "defaults",
			want: want{
				val: testDefaultStyle(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultStyle()
			assertValue(t, got, test.want.val, "default style")
			got.level[slog.LevelInfo] = LevelStyle{
				Text: "changed",
			}
			assertValue(t, DefaultStyle(), test.want.val, "independent defaults")
		})
	}
}

func TestStyle_With(t *testing.T) {
	type fields struct {
		time    TimeStyle
		level   map[slog.Level]LevelStyle
		source  SourceStyle
		label   LabelStyle
		message MessageStyle
		attr    AttrStyle
	}
	type args struct {
		opts []StyleOption
	}
	type want struct {
		val *Style
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "zero style",
			want: want{
				val: &Style{
					level: map[slog.Level]LevelStyle{},
				},
			},
		},
		{
			name: "zero option",
			args: args{
				opts: []StyleOption{{}},
			},
			want: want{
				val: &Style{
					level: map[slog.Level]LevelStyle{},
				},
			},
		},
		{
			name: "nil map option",
			args: args{
				opts: []StyleOption{WithLevelStyle(nil)},
			},
			want: want{
				val: &Style{
					level: map[slog.Level]LevelStyle{},
				},
			},
		},
		{
			name: "empty map option",
			args: args{
				opts: []StyleOption{WithLevelStyle(map[slog.Level]LevelStyle{})},
			},
			want: want{
				val: &Style{
					level: map[slog.Level]LevelStyle{},
				},
			},
		},
		{
			name: "zero style custom level",
			args: args{
				opts: []StyleOption{WithLevelStyle(map[slog.Level]LevelStyle{
					slog.Level(-8): {
						Text: "TRC",
					},
				})},
			},
			want: want{
				val: &Style{
					level: map[slog.Level]LevelStyle{
						slog.Level(-8): {
							Text: "TRC",
						},
					},
				},
			},
		},
		{
			name: "merge and preserve fields",
			fields: fields{
				time: TimeStyle{
					Color: NewColor(CodeFgRed),
				},
				level: map[slog.Level]LevelStyle{
					slog.LevelInfo: {
						Text: "INF",
					},
				},
				source: SourceStyle{
					Color: NewColor(CodeFgBlue),
				},
				label: LabelStyle{
					Width: 4,
				},
				message: MessageStyle{
					Color: NewColor(CodeFgGreen),
				},
				attr: AttrStyle{
					Separator: "=",
				},
			},
			args: args{
				opts: []StyleOption{WithLevelStyle(map[slog.Level]LevelStyle{
					slog.Level(-8): {
						Text: "TRC",
					},
				}), WithAttrStyle(AttrStyle{
					Separator: ":",
				})},
			},
			want: want{
				val: &Style{
					time: TimeStyle{
						Color: NewColor(CodeFgRed),
					},
					level: map[slog.Level]LevelStyle{
						slog.LevelInfo: {
							Text: "INF",
						},
						slog.Level(-8): {
							Text: "TRC",
						},
					},
					source: SourceStyle{
						Color: NewColor(CodeFgBlue),
					},
					label: LabelStyle{
						Width: 4,
					},
					message: MessageStyle{
						Color: NewColor(CodeFgGreen),
					},
					attr: AttrStyle{
						Separator: ":",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &Style{
				time:    test.fields.time,
				level:   test.fields.level,
				source:  test.fields.source,
				label:   test.fields.label,
				message: test.fields.message,
				attr:    test.fields.attr,
			}
			before := *o
			before.level = maps.Clone(o.level)
			got := o.With(test.args.opts...)
			assertValue(t, got, test.want.val, "style")
			got.level[slog.LevelError] = LevelStyle{
				Text: "changed",
			}
			assertValue(t, o, &before, "original style")
		})
	}
}

func TestStyle_With_input(t *testing.T) {
	type args struct {
		level map[slog.Level]LevelStyle
	}
	type want struct {
		level map[slog.Level]LevelStyle
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "input map is copied",
			args: args{
				level: map[slog.Level]LevelStyle{slog.LevelInfo: {
					Text: "INFO",
				}},
			},
			want: want{
				level: map[slog.Level]LevelStyle{slog.LevelInfo: {
					Text: "INFO",
				}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &Style{}
			got := o.With(WithLevelStyle(test.args.level))
			test.args.level[slog.LevelInfo] = LevelStyle{
				Text: "changed",
			}
			assertValue(t, got.level, test.want.level, "copied level map")
		})
	}
}
