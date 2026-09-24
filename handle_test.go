package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestNewCLIHandler(t *testing.T) {
	type args struct {
		w    func(*testing.T) io.Writer
		opts []CLIHandlerOption
	}
	type want struct {
		level   slog.Leveler
		layout  string
		time    bool
		label   string
		source  string
		discard bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "defaults",
			args: args{
				w: func(*testing.T) io.Writer {
					return &bytes.Buffer{}
				},
			},
			want: want{
				level:  slog.LevelInfo,
				layout: time.RFC3339,
			},
		},
		{
			name: "nil writer",
			args: args{
				w: func(*testing.T) io.Writer {
					return nil
				},
			},
			want: want{
				level:   slog.LevelInfo,
				layout:  time.RFC3339,
				discard: true,
			},
		},
		{
			name: "discard",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
			},
			want: want{
				level:   slog.LevelInfo,
				layout:  time.RFC3339,
				discard: true,
			},
		},
		{
			name: "zero options",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
				opts: []CLIHandlerOption{{}, WithLevel(nil), WithStyle(nil), WithAttrReplacer(nil)},
			},
			want: want{
				level:   slog.LevelInfo,
				layout:  time.RFC3339,
				discard: true,
			},
		},
		{
			name: "all options",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
				opts: []CLIHandlerOption{WithLevel(slog.LevelWarn), WithLabel("APP"), WithTime(), WithTimeLayout(time.Kitchen), WithSourcePath()},
			},
			want: want{
				level:   slog.LevelWarn,
				layout:  time.Kitchen,
				discard: true,
				time:    true,
				label:   "APP",
				source:  "file.go",
			},
		},
		{
			name: "source function wins",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
				opts: []CLIHandlerOption{WithSourcePath(), WithSourceFunction()},
			},
			want: want{
				level:   slog.LevelInfo,
				layout:  time.RFC3339,
				discard: true,
				source:  "pkg.Func",
			},
		},
		{
			name: "source path wins",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
				opts: []CLIHandlerOption{WithSourceFunction(), WithSourcePath()},
			},
			want: want{
				level:   slog.LevelInfo,
				layout:  time.RFC3339,
				discard: true,
				source:  "file.go",
			},
		},
		{
			name: "empty layout resets default",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
				opts: []CLIHandlerOption{WithTimeLayout(time.Kitchen), WithTimeLayout("")},
			},
			want: want{
				level:   slog.LevelInfo,
				layout:  time.RFC3339,
				discard: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewCLIHandler(test.args.w(t), test.args.opts...)
			assertValue(t, got.config.level.threshold, test.want.level, "level")
			assertValue(t, got.config.time.layout, test.want.layout, "time layout")
			assertValue(t, got.config.attr.timeLayout, test.want.layout, "attribute layout")
			assertValue(t, got.config.time.enabled, test.want.time, "time")
			assertValue(t, got.config.label.value, test.want.label, "label")
			assertValue(t, testSourceValue(got.config.source.value), test.want.source, "source value")
			assertValue(t, got.source != nil, test.want.source != "", "source cache")
			assertValue(t, got.writer.discard, test.want.discard, "discard")
		})
	}
}

func TestCLIHandler_Enabled(t *testing.T) {
	type fields struct {
		config config
		attrs  []byte
		groups []string
		writer *writer
		source *source
	}
	type args struct {
		ctx   context.Context
		level slog.Level
	}
	type want struct {
		val bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "nil threshold",
			args: args{
				level: slog.LevelDebug,
			},
			want: want{
				val: true,
			},
		},
		{
			name: "slog.LevelInfo/slog.LevelDebug",
			fields: fields{
				config: config{
					level: levelConfig{
						threshold: slog.LevelInfo,
					},
				},
			},
			args: args{
				level: slog.LevelDebug,
			},
			want: want{
				val: false,
			},
		},
		{
			name: "slog.LevelInfo/slog.LevelInfo",
			fields: fields{
				config: config{
					level: levelConfig{
						threshold: slog.LevelInfo,
					},
				},
			},
			args: args{
				level: slog.LevelInfo,
			},
			want: want{
				val: true,
			},
		},
		{
			name: "slog.LevelWarn/slog.LevelError",
			fields: fields{
				config: config{
					level: levelConfig{
						threshold: slog.LevelWarn,
					},
				},
			},
			args: args{
				level: slog.LevelError,
			},
			want: want{
				val: true,
			},
		},
		{
			name: "slog.Level(-8)/slog.Level(-9)",
			fields: fields{
				config: config{
					level: levelConfig{
						threshold: slog.Level(-8),
					},
				},
			},
			args: args{
				level: slog.Level(-9),
			},
			want: want{
				val: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				config: test.fields.config,
				attrs:  test.fields.attrs,
				groups: test.fields.groups,
				writer: test.fields.writer,
				source: test.fields.source,
			}
			got := o.Enabled(test.args.ctx, test.args.level)
			assertValue(t, got, test.want.val, "enabled")
		})
	}
}

func TestCLIHandler_Enabled_dynamic(t *testing.T) {
	type fields struct {
		level *slog.LevelVar
	}
	type args struct {
		thresholds []slog.Level
		level      slog.Level
	}
	type want struct {
		val []bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "observe each threshold update",
			fields: fields{
				level: new(slog.LevelVar),
			},
			args: args{
				thresholds: []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelDebug},
				level:      slog.LevelInfo,
			},
			want: want{
				val: []bool{true, false, true},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				config: config{
					level: levelConfig{
						threshold: test.fields.level,
					},
				},
			}
			var got []bool
			for _, threshold := range test.args.thresholds {
				test.fields.level.Set(threshold)
				got = append(got, o.Enabled(t.Context(), test.args.level))
			}
			assertValue(t, got, test.want.val, "dynamic threshold")
		})
	}
}

func TestCLIHandler_Handle(t *testing.T) {
	type fields struct {
		config config
		attrs  []byte
		groups []string
		writer *writer
		source *source
	}
	type args struct {
		ctx    context.Context
		record slog.Record
	}
	type want struct {
		val string
		err func(*testWriter) error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "message",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0),
			},
			want: want{
				val: "INF message\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "empty message",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "", 0),
			},
			want: want{
				val: "INF \n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "custom unstyled level",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.Level(-8), "message", 0),
			},
			want: want{
				val: "DEBUG-4 message\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "time zero omitted",
			fields: fields{
				config: testConfig(false, WithTime()),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0),
			},
			want: want{
				val: "INF message\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "source PC zero omitted",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				source: testSource("file.go", 83),
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0),
			},
			want: want{
				val: "INF message\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "all built-ins order",
			fields: fields{
				config: testConfig(false, WithTime(), WithLabel("APP")),
				writer: &writer{
					w: &testWriter{},
				},
				source: testSource("/src/main.go", 83),
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 1, slog.Int("version", 1)),
			},
			want: want{
				val: "1970-01-01T00:00:01Z INF </src/main.go:83> APP message version=1\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "quoted source",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				source: testSource("a b.go", 83),
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 1),
			},
			want: want{
				val: `INF <"a b.go:83"> message` + "\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "empty group attr",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0, slog.Group("g")),
			},
			want: want{
				val: "INF message\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "cached before record attrs",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				attrs:  []byte("before=1"),
				groups: []string{"g"},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0, slog.Int("after", 2)),
			},
			want: want{
				val: "INF message before=1 g.after=2\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "nested and inline groups",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				groups: []string{"base"},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0, slog.Group("a", slog.Group("", slog.Int("k", 1)), slog.Group("b", slog.Int("n", 2)), slog.Int("sibling", 3)), slog.Int("after", 4)),
			},
			want: want{
				val: "INF message base.a.k=1 base.a.b.n=2 base.a.sibling=3 base.after=4\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "replacer excludes built-ins",
			fields: fields{
				config: testConfig(false, WithTime(), WithLabel("APP"), WithAttrReplacer(func(_ []string, a slog.Attr) slog.Attr {
					return slog.String(a.Key, "replaced")
				})),
				writer: &writer{
					w: &testWriter{},
				},
				source: testSource("file.go", 83),
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 1, slog.Int("k", 1)),
			},
			want: want{
				val: "1970-01-01T00:00:01Z INF <file.go:83> APP message k=replaced\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "writer error",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{
						err: testError(),
					},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0),
			},
			want: want{
				val: "INF message\n",
				err: func(w *testWriter) error {
					return w.err
				},
			},
		},
		{
			name: "large message",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, strings.Repeat("m", 65537), 0),
			},
			want: want{
				val: "INF " + strings.Repeat("m", 65537) + "\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "message controls",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "a\n\x1bb", 0),
			},
			want: want{
				val: "INF \"a\\n\\x1bb\"\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "punctuation message",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, `message="value" C:\path`, 0),
			},
			want: want{
				val: `INF message="value" C:\path` + "\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "unsafe attr and group",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				groups: []string{"user group"},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message", 0, slog.String("bad=key", "line\rvalue"), slog.Any("bytes", []byte{1, 2})),
			},
			want: want{
				val: `INF message "user group"."bad=key"="line\rvalue" "user group".bytes="[1 2]"` + "\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "Japanese",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "日本語", 0, slog.String("名前", "日本語�"), slog.String("empty", "")),
			},
			want: want{
				val: "INF 日本語 名前=日本語� empty=\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "invalid UTF-8",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				groups: []string{"g\xff"},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "m\xff", 0, slog.String("k\xff", "v\xff")),
			},
			want: want{
				val: `INF "m\xff" "g\xff"."k\xff"="v\xff"` + "\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "unicode nonprinting",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "a\u2028b", 0, slog.String("k", "a\u00a0b")),
			},
			want: want{
				val: `INF "a\u2028b" k="a\u00a0b"` + "\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "colored escaping",
			fields: fields{
				config: testConfig(true, WithStyle(DefaultStyle().With(WithMessageStyle(MessageStyle{
					Color: NewColor(CodeFgGreen),
				}), WithAttrStyle(AttrStyle{
					Separator:  "=",
					KeyColor:   NewColor(CodeFgHiBlack),
					ValueColor: NewColor(CodeFgRed),
				})))),
				writer: &writer{
					w: &testWriter{},
				},
				groups: []string{"g\n"},
			},
			args: args{
				record: testRecord(time.Time{}, slog.LevelInfo, "message\n", 0, slog.String("k\r", "v\x1b")),
			},
			want: want{
				val: "\x1b[38;2;95;255;215;1mINF\x1b[0m \x1b[32m\"message\\n\"\x1b[0m \x1b[90m\"g\\n\".\"k\\r\"=\x1b[0m\x1b[31m\"v\\x1b\"\x1b[0m\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "layout default time false",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "INF message at=1970-01-01T00:00:01Z\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "layout default time true",
			fields: fields{
				config: testConfig(false, WithTime()),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "1970-01-01T00:00:01Z INF message at=1970-01-01T00:00:01Z\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "layout nano time false",
			fields: fields{
				config: testConfig(false, WithTimeLayout(time.RFC3339Nano)),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "INF message at=1970-01-01T00:00:01.123456789Z\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "layout nano time true",
			fields: fields{
				config: testConfig(false, WithTime(), WithTimeLayout(time.RFC3339Nano)),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "1970-01-01T00:00:01.123456789Z INF message at=1970-01-01T00:00:01.123456789Z\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "layout space time false",
			fields: fields{
				config: testConfig(false, WithTimeLayout(time.DateTime)),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "INF message at=\"1970-01-01 00:00:01\"\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "layout space time true",
			fields: fields{
				config: testConfig(false, WithTime(), WithTimeLayout(time.DateTime)),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "1970-01-01 00:00:01 INF message at=\"1970-01-01 00:00:01\"\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "large line and deep inherited path",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w: &testWriter{},
				},
				groups: testLargeGroups(),
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, strings.Repeat("m", 65537), 0, slog.Int("key", 1)),
			},
			want: want{
				val: "INF " + strings.Repeat("m", 65537) + " " + strings.Join(testLargeGroups(), ".") + ".key=1\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "trusted time layout and escaped time attribute",
			fields: fields{
				config: testConfig(false, WithTime(), WithTimeLayout("15:04\n\x1b[0m")),
				writer: &writer{
					w: &testWriter{},
				},
			},
			args: args{
				record: testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Time("at", testTime())),
			},
			want: want{
				val: "00:00\n\x1b[0m INF message at=\"00:00\\n\\x1b[0m\"\n",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				config: test.fields.config,
				attrs:  test.fields.attrs,
				groups: test.fields.groups,
				writer: test.fields.writer,
				source: test.fields.source,
			}
			w := o.writer.w.(*testWriter)
			wantErr := test.want.err(w)
			got := o.Handle(test.args.ctx, test.args.record)
			assertError(t, got, wantErr)
			assertBytes(t, w.Bytes(), test.want.val, "output")
		})
	}
}

func TestCLIHandler_WithAttrs(t *testing.T) {
	type fields struct {
		config config
		attrs  []byte
		groups []string
		writer *writer
		source *source
	}
	type args struct {
		attrs []slog.Attr
	}
	type want struct {
		val  string
		same bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "nil attrs",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			want: want{
				same: true,
			},
		},
		{
			name: "empty attrs",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{},
			},
			want: want{
				same: true,
			},
		},
		{
			name: "ordinary",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.Int("k", 1)},
			},
			want: want{
				val: "k=1",
			},
		},
		{
			name: "empty attr",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{{}},
			},
		},
		{
			name: "empty group",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.Group("g")},
			},
		},
		{
			name: "append cache with group",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				attrs:  []byte("before=1"),
				groups: []string{"g"},
			},
			args: args{
				attrs: []slog.Attr{slog.Int("after", 2)},
			},
			want: want{
				val: "before=1 g.after=2",
			},
		},
		{
			name: "nested escaped",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				groups: []string{"base group"},
			},
			args: args{
				attrs: []slog.Attr{slog.Group("a\n", slog.Group("", slog.Int("k", 1)), slog.Int("after", 2))},
			},
			want: want{
				val: `"base group"."a\n".k=1 "base group"."a\n".after=2`,
			},
		},
		{
			name: "replacement",
			fields: fields{
				config: testConfig(false, WithAttrReplacer(func(_ []string, a slog.Attr) slog.Attr {
					return slog.String(a.Key, "redacted")
				})),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.String("password", "secret")},
			},
			want: want{
				val: "password=redacted",
			},
		},
		{
			name: "large attribute",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.String("k", strings.Repeat("x", 65537))},
			},
			want: want{
				val: "k=" + strings.Repeat("x", 65537),
			},
		},
		{
			name: "long path",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				groups: []string{strings.Repeat("g", 4097)},
			},
			args: args{
				attrs: []slog.Attr{slog.Int("k", 1)},
			},
			want: want{
				val: strings.Repeat("g", 4097) + ".k=1",
			},
		},
		{
			name: "time time.RFC3339Nano",
			fields: fields{
				config: testConfig(false, WithTimeLayout(time.RFC3339Nano)),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.Time("at", testTime())},
			},
			want: want{
				val: "at=1970-01-01T00:00:01.123456789Z",
			},
		},
		{
			name: "time time.DateTime",
			fields: fields{
				config: testConfig(false, WithTimeLayout(time.DateTime)),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.Time("at", testTime())},
			},
			want: want{
				val: "at=\"1970-01-01 00:00:01\"",
			},
		},
		{
			name: "time time.Kitchen",
			fields: fields{
				config: testConfig(false, WithTimeLayout(time.Kitchen)),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
			},
			args: args{
				attrs: []slog.Attr{slog.Time("at", testTime())},
			},
			want: want{
				val: "at=12:00AM",
			},
		},
		{
			name: "deep inherited path",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				groups: testLargeGroups(),
			},
			args: args{
				attrs: []slog.Attr{slog.Int("key", 1)},
			},
			want: want{
				val: strings.Join(testLargeGroups(), ".") + ".key=1",
			},
		},
		{
			name: "cached control characters",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
			},
			args: args{
				attrs: []slog.Attr{slog.String("a\n", "b\x1b"), slog.Any("error", testError()), slog.String("invalid", "\xff")},
			},
			want: want{
				val: "\"a\\n\"=\"b\\x1b\" error=\"test error\" invalid=\"\\xff\"",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				config: test.fields.config,
				attrs:  test.fields.attrs,
				groups: test.fields.groups,
				writer: test.fields.writer,
				source: test.fields.source,
			}
			before := string(o.attrs)
			got := o.WithAttrs(test.args.attrs).(*CLIHandler)
			for i := range test.args.attrs {
				test.args.attrs[i] = slog.Int("changed", 0)
			}
			assertBytes(t, got.attrs, test.want.val, "cached attrs")
			assertBytes(t, o.attrs, before, "original attrs")
			assertValue(t, got == o, test.want.same, "same handler")
			assertValue(t, got.writer == o.writer, true, "shared writer")
			assertValue(t, got.source == o.source, true, "shared source")
			assertValue(t, got.groups, o.groups, "groups")
		})
	}
}

func TestCLIHandler_WithAttrs_independent(t *testing.T) {
	type fields struct {
		attrs  func() []byte
		groups []string
	}
	type args struct {
		attrs [][]slog.Attr
	}
	type want struct {
		attrs  []string
		parent string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "siblings retain independent caches",
			fields: fields{
				attrs: func() []byte {
					value := make([]byte, len("base=1"), 128)
					copy(value, "base=1")
					return value
				},
				groups: []string{"g"},
			},
			args: args{
				attrs: [][]slog.Attr{{slog.Int("left", 2)}, {slog.Int("right", 3)}},
			},
			want: want{
				attrs:  []string{"base=1 g.left=2", "base=1 g.right=3"},
				parent: "base=1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				config: testConfig(false),
				attrs:  test.fields.attrs(),
				groups: test.fields.groups,
			}
			var got []*CLIHandler
			for _, attrs := range test.args.attrs {
				got = append(got, o.WithAttrs(attrs).(*CLIHandler))
			}
			for i, handler := range got {
				assertBytes(t, handler.attrs, test.want.attrs[i], "independent cache")
			}
			assertBytes(t, o.attrs, test.want.parent, "parent cache")
		})
	}
}

func TestCLIHandler_WithGroup(t *testing.T) {
	type fields struct {
		config config
		attrs  []byte
		groups []string
		writer *writer
		source *source
	}
	type args struct {
		name string
	}
	type want struct {
		val  []string
		same bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty name",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				attrs:  []byte("cached=1"),
				groups: []string{"base"},
			},
			want: want{
				val:  []string{"base"},
				same: true,
			},
		},
		{
			name: "first",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				attrs:  []byte("cached=1"),
			},
			args: args{
				name: "g",
			},
			want: want{
				val: []string{"g"},
			},
		},
		{
			name: "nested",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				attrs:  []byte("cached=1"),
				groups: []string{"a"},
			},
			args: args{
				name: "b",
			},
			want: want{
				val: []string{"a", "b"},
			},
		},
		{
			name: "retain original name",
			fields: fields{
				config: testConfig(false),
				writer: &writer{
					w:       io.Discard,
					discard: true,
				},
				source: testSource("file", 1),
				attrs:  []byte("cached=1"),
			},
			args: args{
				name: "a b\n",
			},
			want: want{
				val: []string{"a b\n"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				config: test.fields.config,
				attrs:  test.fields.attrs,
				groups: test.fields.groups,
				writer: test.fields.writer,
				source: test.fields.source,
			}
			before := append([]string(nil), o.groups...)
			got := o.WithGroup(test.args.name).(*CLIHandler)
			assertValue(t, got.groups, test.want.val, "groups")
			assertValue(t, o.groups, before, "original groups")
			assertValue(t, got == o, test.want.same, "same handler")
			assertValue(t, got.writer == o.writer, true, "shared writer")
			assertValue(t, got.source == o.source, true, "shared source")
			assertBytes(t, got.attrs, string(o.attrs), "cached attrs")
		})
	}
}

func TestCLIHandler_WithGroup_independent(t *testing.T) {
	type fields struct {
		groups func() []string
	}
	type args struct {
		names []string
	}
	type want struct {
		groups [][]string
		parent []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "siblings retain independent paths",
			fields: fields{
				groups: func() []string {
					value := make([]string, 1, 8)
					value[0] = "base"
					return value
				},
			},
			args: args{
				names: []string{"left", "right"},
			},
			want: want{
				groups: [][]string{{"base", "left"}, {"base", "right"}},
				parent: []string{"base"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &CLIHandler{
				groups: test.fields.groups(),
			}
			var got []*CLIHandler
			for _, name := range test.args.names {
				got = append(got, o.WithGroup(name).(*CLIHandler))
			}
			for i, handler := range got {
				assertValue(t, handler.groups, test.want.groups[i], "independent path")
			}
			assertValue(t, o.groups, test.want.parent, "parent path")
		})
	}
}
