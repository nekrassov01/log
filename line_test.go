package log

import (
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"
)

func Test_lineState_appendTime(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		value  time.Time
		config timeConfig
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "first",
			args: args{
				value: testTime(),
				config: timeConfig{
					layout: time.RFC3339,
				},
			},
			want: want{
				buf:   "1970-01-01T00:00:01Z",
				wrote: true,
			},
		},
		{
			name: "zero time direct",
			args: args{
				value: time.Time{},
				config: timeConfig{
					layout: time.RFC3339,
				},
			},
			want: want{
				buf:   "0001-01-01T00:00:00Z",
				wrote: true,
			},
		},
		{
			name: "append decorated",
			fields: fields{
				buf:   []byte("start"),
				wrote: true,
			},
			args: args{
				value: testTime(),
				config: timeConfig{
					prefix: []byte("<"),
					suffix: []byte(">"),
					layout: time.Kitchen,
				},
			},
			want: want{
				buf:   "start <12:00AM>",
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendTime(test.args.value, &test.args.config)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendLevel(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		level slog.Level
		texts map[slog.Level][]byte
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "INFO",
			args: args{
				level: slog.Level(0),
			},
			want: want{
				buf:   "INFO",
				wrote: true,
			},
		},
		{
			name: "DEBUG",
			args: args{
				level: slog.Level(-4),
			},
			want: want{
				buf:   "DEBUG",
				wrote: true,
			},
		},
		{
			name: "WARN",
			args: args{
				level: slog.Level(4),
			},
			want: want{
				buf:   "WARN",
				wrote: true,
			},
		},
		{
			name: "ERROR",
			args: args{
				level: slog.Level(8),
			},
			want: want{
				buf:   "ERROR",
				wrote: true,
			},
		},
		{
			name: "DEBUG-4",
			args: args{
				level: slog.Level(-8),
			},
			want: want{
				buf:   "DEBUG-4",
				wrote: true,
			},
		},
		{
			name: "DEBUG+1",
			args: args{
				level: slog.Level(-3),
			},
			want: want{
				buf:   "DEBUG+1",
				wrote: true,
			},
		},
		{
			name: "INFO+3",
			args: args{
				level: slog.Level(3),
			},
			want: want{
				buf:   "INFO+3",
				wrote: true,
			},
		},
		{
			name: "WARN+3",
			args: args{
				level: slog.Level(7),
			},
			want: want{
				buf:   "WARN+3",
				wrote: true,
			},
		},
		{
			name: "ERROR+1",
			args: args{
				level: slog.Level(9),
			},
			want: want{
				buf:   "ERROR+1",
				wrote: true,
			},
		},
		{
			name: "cached text",
			fields: fields{
				buf:   []byte("start"),
				wrote: true,
			},
			args: args{
				level: slog.LevelInfo,
				texts: map[slog.Level][]byte{
					slog.LevelInfo: []byte("INF"),
				},
			},
			want: want{
				buf:   "start INF",
				wrote: true,
			},
		},
		{
			name: "empty cached text falls back",
			args: args{
				level: slog.LevelInfo,
				texts: map[slog.Level][]byte{
					slog.LevelInfo: nil,
				},
			},
			want: want{
				buf:   "INFO",
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendLevel(test.args.level, test.args.texts)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendSource(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		text   []byte
		config sourceConfig
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "source",
			args: args{
				text: []byte("main.go:83"),
				config: sourceConfig{
					prefix: []byte("<"),
					suffix: []byte(">"),
				},
			},
			want: want{
				buf:   "<main.go:83>",
				wrote: true,
			},
		},
		{
			name: "prequoted",
			fields: fields{
				buf:   []byte("start"),
				wrote: true,
			},
			args: args{
				text: []byte(`"a b.go:83"`),
			},
			want: want{
				buf:   `start "a b.go:83"`,
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendSource(test.args.text, &test.args.config)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendLabel(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		config labelConfig
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "label",
			args: args{
				config: labelConfig{
					value:  "APP",
					prefix: []byte("<"),
					suffix: []byte(">"),
				},
			},
			want: want{
				buf:   "<APP>",
				wrote: true,
			},
		},
		{
			name: "empty direct",
			want: want{
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendLabel(&test.args.config)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendMessage(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		message string
		config  messageConfig
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty",
			want: want{
				wrote: true,
			},
		},
		{
			name: "plain",
			args: args{
				message: `hello "world" C:\path`,
			},
			want: want{
				buf:   `hello "world" C:\path`,
				wrote: true,
			},
		},
		{
			name: "unsafe decorated",
			fields: fields{
				buf:   []byte("INF"),
				wrote: true,
			},
			args: args{
				message: "hello\nworld",
				config: messageConfig{
					prefix: []byte("<"),
					suffix: []byte(">"),
				},
			},
			want: want{
				buf:   `INF <"hello\nworld">`,
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendMessage(test.args.message, &test.args.config)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendCachedAttrs(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		attrs []byte
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "nil",
		},
		{
			name: "empty preserves line",
			fields: fields{
				buf:   []byte("INF"),
				wrote: true,
			},
			want: want{
				buf:   "INF",
				wrote: true,
			},
		},
		{
			name: "first",
			args: args{
				attrs: []byte("k=v"),
			},
			want: want{
				buf:   "k=v",
				wrote: true,
			},
		},
		{
			name: "append",
			fields: fields{
				buf:   []byte("INF"),
				wrote: true,
			},
			args: args{
				attrs: []byte("k=v"),
			},
			want: want{
				buf:   "INF k=v",
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendCachedAttrs(test.args.attrs)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendAttr(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type args struct {
		attr   slog.Attr
		kind   slog.Kind
		path   []byte
		config attrConfig
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "string",
			args: args{
				attr: slog.String("k", "value"),
				kind: slog.KindString,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=value",
				wrote: true,
			},
		},
		{
			name: "empty string",
			args: args{
				attr: slog.String("k", ""),
				kind: slog.KindString,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=",
				wrote: true,
			},
		},
		{
			name: "quoted string",
			args: args{
				attr: slog.String("k", "a b"),
				kind: slog.KindString,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=\"a b\"",
				wrote: true,
			},
		},
		{
			name: "quoted key",
			args: args{
				attr: slog.String("a=b", "v"),
				kind: slog.KindString,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "\"a=b\"=v",
				wrote: true,
			},
		},
		{
			name: "int min",
			args: args{
				attr: slog.Int64("k", math.MinInt64),
				kind: slog.KindInt64,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=-9223372036854775808",
				wrote: true,
			},
		},
		{
			name: "uint max",
			args: args{
				attr: slog.Uint64("k", math.MaxUint64),
				kind: slog.KindUint64,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=18446744073709551615",
				wrote: true,
			},
		},
		{
			name: "float",
			args: args{
				attr: slog.Float64("k", 1.5),
				kind: slog.KindFloat64,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=1.5",
				wrote: true,
			},
		},
		{
			name: "NaN",
			args: args{
				attr: slog.Float64("k", math.NaN()),
				kind: slog.KindFloat64,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=NaN",
				wrote: true,
			},
		},
		{
			name: "infinite",
			args: args{
				attr: slog.Float64("k", math.Inf(1)),
				kind: slog.KindFloat64,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=+Inf",
				wrote: true,
			},
		},
		{
			name: "true",
			args: args{
				attr: slog.Bool("k", true),
				kind: slog.KindBool,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=true",
				wrote: true,
			},
		},
		{
			name: "false",
			args: args{
				attr: slog.Bool("k", false),
				kind: slog.KindBool,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=false",
				wrote: true,
			},
		},
		{
			name: "time",
			args: args{
				attr: slog.Time("k", testTime()),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=1970-01-01T00:00:01.123456789Z",
				wrote: true,
			},
		},
		{
			name: "zero time",
			args: args{
				attr: slog.Time("k", time.Time{}),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=0001-01-01T00:00:00Z",
				wrote: true,
			},
		},
		{
			name: "duration",
			args: args{
				attr: slog.Duration("k", -time.Second),
				kind: slog.KindDuration,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=-1s",
				wrote: true,
			},
		},
		{
			name: "any nil",
			args: args{
				attr: slog.Any("k", nil),
				kind: slog.KindAny,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=<nil>",
				wrote: true,
			},
		},
		{
			name: "any bytes",
			args: args{
				attr: slog.Any("k", []byte{1, 2}),
				kind: slog.KindAny,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=\"[1 2]\"",
				wrote: true,
			},
		},
		{
			name: "any error",
			args: args{
				attr: slog.Any("k", testError()),
				kind: slog.KindAny,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=\"test error\"",
				wrote: true,
			},
		},
		{
			name: "any stringer",
			args: args{
				attr: slog.Any("k", testStringer("a b")),
				kind: slog.KindAny,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=\"a b\"",
				wrote: true,
			},
		},
		{
			name: "time custom layout",
			args: args{
				attr: slog.Time("k", testTime()),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.DateTime,
				},
			},
			want: want{
				buf:   `k="1970-01-01 00:00:01"`,
				wrote: true,
			},
		},
		{
			name: "long time layout",
			args: args{
				attr: slog.Time("k", testTime()),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: strings.Repeat("x", 100),
				},
			},
			want: want{
				buf:   "k=" + strings.Repeat("x", 100),
				wrote: true,
			},
		},
		{
			name: "styled",
			args: args{
				attr: slog.String("k", "v"),
				kind: slog.KindString,
				path: []byte("g."),
				config: attrConfig{
					separator:   "=",
					keyPrefix:   []byte("<"),
					keySuffix:   []byte(">"),
					valuePrefix: []byte("["),
					valueSuffix: []byte("]"),
				},
			},
			want: want{
				buf:   "<g.k=>[v]",
				wrote: true,
			},
		},
		{
			name: "styled empty value",
			args: args{
				attr: slog.String("k", ""),
				kind: slog.KindString,
				config: attrConfig{
					separator:   "=",
					keyPrefix:   []byte("<"),
					keySuffix:   []byte(">"),
					valuePrefix: []byte("["),
					valueSuffix: []byte("]"),
				},
			},
			want: want{
				buf:   "<k=>",
				wrote: true,
			},
		},
		{
			name: "no key path or separator",
			args: args{
				attr: slog.String("", "v"),
				kind: slog.KindString,
				config: attrConfig{
					keyPrefix: []byte("<"),
					keySuffix: []byte(">"),
				},
			},
			want: want{
				buf:   "v",
				wrote: true,
			},
		},
		{
			name: "empty key and separator with path",
			args: args{
				attr: slog.String("", "v"),
				kind: slog.KindString,
				path: []byte("g."),
				config: attrConfig{
					keyPrefix: []byte("<"),
					keySuffix: []byte(">"),
				},
			},
			want: want{
				buf:   "<g.>v",
				wrote: true,
			},
		},
		{
			name: "nanosecond",
			args: args{
				attr: slog.Time("k", time.Unix(1, 1).UTC()),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=1970-01-01T00:00:01.000000001Z",
				wrote: true,
			},
		},
		{
			name: "trim fraction",
			args: args{
				attr: slog.Time("k", time.Unix(1, 123000000).UTC()),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=1970-01-01T00:00:01.123Z",
				wrote: true,
			},
		},
		{
			name: "timezone offset",
			args: args{
				attr: slog.Time("k", time.Unix(1, 987654321).In(time.FixedZone("JST", 9*60*60))),
				kind: slog.KindTime,
				config: attrConfig{
					separator:  "=",
					timeLayout: time.RFC3339Nano,
				},
			},
			want: want{
				buf:   "k=1970-01-01T09:00:01.987654321+09:00",
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendAttr(test.args.attr, test.args.kind, test.args.path, &test.args.config)
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendNewline(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type want struct {
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "empty",
			want: want{
				buf: "\n",
			},
		},
		{
			name: "line",
			fields: fields{
				buf:   []byte("message"),
				wrote: true,
			},
			want: want{
				buf:   "message\n",
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.appendNewline()
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_appendSeparator(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type want struct {
		val   string
		buf   string
		wrote bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "empty",
		},
		{
			name: "first data",
			fields: fields{
				buf: []byte("prefix"),
			},
			want: want{
				val: "prefix",
				buf: "prefix",
			},
		},
		{
			name: "existing line",
			fields: fields{
				buf:   []byte("message"),
				wrote: true,
			},
			want: want{
				val:   "message ",
				buf:   "message",
				wrote: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			got := o.appendSeparator()
			assertBytes(t, got, test.want.val, "returned buffer")
			assertBytes(t, o.buf, test.want.buf, "line")
			assertValue(t, o.wrote, test.want.wrote, "wrote")
		})
	}
}

func Test_lineState_reset(t *testing.T) {
	type fields struct {
		buf   []byte
		wrote bool
	}
	type want struct {
		capacity int
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "nil",
		},
		{
			name: "retains buffer",
			fields: fields{
				buf:   []byte("abc"),
				wrote: true,
			},
			want: want{
				capacity: 3,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &lineState{
				buf:   test.fields.buf,
				wrote: test.fields.wrote,
			}
			o.reset()
			assertBytes(t, o.buf, "", "line")
			assertValue(t, o.wrote, false, "wrote")
			assertValue(t, cap(o.buf), test.want.capacity, "capacity")
		})
	}
}
