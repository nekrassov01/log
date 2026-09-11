package log

import "testing"

func TestNewColor(t *testing.T) {
	type args struct {
		codes []int
	}
	type want struct {
		val *Color
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil",
			want: want{
				val: &Color{},
			},
		},
		{
			name: "empty",
			args: args{
				codes: []int{},
			},
			want: want{
				val: &Color{},
			},
		},
		{
			name: "single",
			args: args{
				codes: []int{31},
			},
			want: want{
				val: &Color{
					prefix: []byte("\x1b[31m"),
					suffix: []byte("\x1b[0m"),
				},
			},
		},
		{
			name: "multiple",
			args: args{
				codes: []int{1, 34, 45},
			},
			want: want{
				val: &Color{
					prefix: []byte("\x1b[1;34;45m"),
					suffix: []byte("\x1b[0m"),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewColor(test.args.codes...)
			assertValue(t, got, test.want.val, "color")
		})
	}
}

func TestColor_appendText(t *testing.T) {
	type fields struct {
		prefix []byte
		suffix []byte
	}
	type args struct {
		dst     []byte
		s       string
		enabled bool
	}
	type want struct {
		val string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty color",
			args: args{
				dst:     []byte("start"),
				s:       "text",
				enabled: true,
			},
			want: want{
				val: "starttext",
			},
		},
		{
			name: "disabled",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				dst:     []byte("start"),
				s:       "text",
				enabled: false,
			},
			want: want{
				val: "starttext",
			},
		},
		{
			name: "enabled",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				dst:     []byte("start"),
				s:       "text",
				enabled: true,
			},
			want: want{
				val: "start<text>",
			},
		},
		{
			name: "empty text",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				enabled: true,
			},
		},
		{
			name: "suffix only",
			fields: fields{
				suffix: []byte(">"),
			},
			args: args{
				s:       "text",
				enabled: true,
			},
			want: want{
				val: "text",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &Color{
				prefix: test.fields.prefix,
				suffix: test.fields.suffix,
			}
			got := o.appendText(test.args.dst, test.args.s, test.args.enabled)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

func TestColor_appendText_nil(t *testing.T) {
	type args struct {
		dst     []byte
		s       string
		enabled bool
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
			name: "nil receiver",
			args: args{
				dst:     []byte("start"),
				s:       "text",
				enabled: true,
			},
			want: want{
				val: "starttext",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := (*Color)(nil).appendText(test.args.dst, test.args.s, test.args.enabled)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

func TestColor_appendPrefix(t *testing.T) {
	type fields struct {
		prefix []byte
		suffix []byte
	}
	type args struct {
		dst     []byte
		enabled bool
	}
	type want struct {
		val string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty color",
			args: args{
				dst:     []byte("start"),
				enabled: true,
			},
			want: want{
				val: "start",
			},
		},
		{
			name: "disabled",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				dst:     []byte("start"),
				enabled: false,
			},
			want: want{
				val: "start",
			},
		},
		{
			name: "enabled",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				dst:     []byte("start"),
				enabled: true,
			},
			want: want{
				val: "start<",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &Color{
				prefix: test.fields.prefix,
				suffix: test.fields.suffix,
			}
			got := o.appendPrefix(test.args.dst, test.args.enabled)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

func TestColor_appendPrefix_nil(t *testing.T) {
	type args struct {
		dst     []byte
		enabled bool
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
			name: "nil receiver",
			args: args{
				dst:     []byte("start"),
				enabled: true,
			},
			want: want{
				val: "start",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := (*Color)(nil).appendPrefix(test.args.dst, test.args.enabled)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

func TestColor_appendSuffix(t *testing.T) {
	type fields struct {
		prefix []byte
		suffix []byte
	}
	type args struct {
		dst     []byte
		enabled bool
	}
	type want struct {
		val string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty color",
			args: args{
				dst:     []byte("start"),
				enabled: true,
			},
			want: want{
				val: "start",
			},
		},
		{
			name: "disabled",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				dst:     []byte("start"),
				enabled: false,
			},
			want: want{
				val: "start",
			},
		},
		{
			name: "enabled",
			fields: fields{
				prefix: []byte("<"),
				suffix: []byte(">"),
			},
			args: args{
				dst:     []byte("start"),
				enabled: true,
			},
			want: want{
				val: "start>",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &Color{
				prefix: test.fields.prefix,
				suffix: test.fields.suffix,
			}
			got := o.appendSuffix(test.args.dst, test.args.enabled)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

func TestColor_appendSuffix_nil(t *testing.T) {
	type args struct {
		dst     []byte
		enabled bool
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
			name: "nil receiver",
			args: args{
				dst:     []byte("start"),
				enabled: true,
			},
			want: want{
				val: "start",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := (*Color)(nil).appendSuffix(test.args.dst, test.args.enabled)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

func Test_buildSGR(t *testing.T) {
	type args struct {
		codes []int
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
			name: "empty",
			args: args{
				codes: []int{},
			},
		},
		{
			name: "reset",
			args: args{
				codes: []int{0},
			},
			want: want{
				val: "\x1b[0m",
			},
		},
		{
			name: "multiple",
			args: args{
				codes: []int{1, 34, 45},
			},
			want: want{
				val: "\x1b[1;34;45m",
			},
		},
		{
			name: "extended",
			args: args{
				codes: []int{38, 2, 255, 0, 128},
			},
			want: want{
				val: "\x1b[38;2;255;0;128m",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildSGR(test.args.codes)
			assertBytes(t, got, test.want.val, "SGR")
		})
	}
}
