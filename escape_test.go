package log

import "testing"

func Test_escapeText(t *testing.T) {
	type args struct {
		buf []byte
		s   string
	}
	type want struct {
		val    string
		quoted bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty",
			args: args{
				buf: []byte("prefix:"),
				s:   "",
			},
			want: want{
				val:    "prefix:",
				quoted: false,
			},
		},
		{
			name: "plain",
			args: args{
				buf: []byte("prefix:"),
				s:   "hello",
			},
			want: want{
				val:    "prefix:hello",
				quoted: false,
			},
		},
		{
			name: "Japanese",
			args: args{
				buf: []byte("prefix:"),
				s:   "日本語�",
			},
			want: want{
				val:    "prefix:日本語�",
				quoted: false,
			},
		},
		{
			name: "space",
			args: args{
				buf: []byte("prefix:"),
				s:   "a b",
			},
			want: want{
				val:    "prefix:\"a b\"",
				quoted: true,
			},
		},
		{
			name: "equals",
			args: args{
				buf: []byte("prefix:"),
				s:   "a=b",
			},
			want: want{
				val:    "prefix:\"a=b\"",
				quoted: true,
			},
		},
		{
			name: "quote",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\"b",
			},
			want: want{
				val:    "prefix:\"a\\\"b\"",
				quoted: true,
			},
		},
		{
			name: "backslash",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\\b",
			},
			want: want{
				val:    "prefix:\"a\\\\b\"",
				quoted: true,
			},
		},
		{
			name: "newline",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\nb",
			},
			want: want{
				val:    "prefix:\"a\\nb\"",
				quoted: true,
			},
		},
		{
			name: "controls",
			args: args{
				buf: []byte("prefix:"),
				s:   "\r\t\u0000\x7f",
			},
			want: want{
				val:    "prefix:\"\\r\\t\\x00\\x7f\"",
				quoted: true,
			},
		},
		{
			name: "escape",
			args: args{
				buf: []byte("prefix:"),
				s:   "\u001b[2J",
			},
			want: want{
				val:    "prefix:\"\\x1b[2J\"",
				quoted: true,
			},
		},
		{
			name: "nonbreaking space",
			args: args{
				buf: []byte("prefix:"),
				s:   "a b",
			},
			want: want{
				val:    "prefix:\"a\\u00a0b\"",
				quoted: true,
			},
		},
		{
			name: "line separator",
			args: args{
				buf: []byte("prefix:"),
				s:   "a b",
			},
			want: want{
				val:    "prefix:\"a\\u2028b\"",
				quoted: true,
			},
		},
		{
			name: "format",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\u200bb",
			},
			want: want{
				val:    "prefix:\"a\\u200bb\"",
				quoted: true,
			},
		},
		{
			name: "invalid UTF-8",
			args: args{
				s: "a\xff",
			},
			want: want{
				val:    `"a\xff"`,
				quoted: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, quoted := escapeText(test.args.buf, test.args.s)
			assertBytes(t, got, test.want.val, "text")
			assertValue(t, quoted, test.want.quoted, "quoted")
		})
	}
}

func Test_escapeMessage(t *testing.T) {
	type args struct {
		buf []byte
		s   string
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
			args: args{
				buf: []byte("prefix:"),
				s:   "",
			},
			want: want{
				val: "prefix:",
			},
		},
		{
			name: "plain",
			args: args{
				buf: []byte("prefix:"),
				s:   "hello",
			},
			want: want{
				val: "prefix:hello",
			},
		},
		{
			name: "Japanese",
			args: args{
				buf: []byte("prefix:"),
				s:   "日本語�",
			},
			want: want{
				val: "prefix:日本語�",
			},
		},
		{
			name: "space",
			args: args{
				buf: []byte("prefix:"),
				s:   "a b",
			},
			want: want{
				val: "prefix:a b",
			},
		},
		{
			name: "equals",
			args: args{
				buf: []byte("prefix:"),
				s:   "a=b",
			},
			want: want{
				val: "prefix:a=b",
			},
		},
		{
			name: "quote",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\"b",
			},
			want: want{
				val: "prefix:a\"b",
			},
		},
		{
			name: "backslash",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\\b",
			},
			want: want{
				val: "prefix:a\\b",
			},
		},
		{
			name: "newline",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\nb",
			},
			want: want{
				val: "prefix:\"a\\nb\"",
			},
		},
		{
			name: "controls",
			args: args{
				buf: []byte("prefix:"),
				s:   "\r\t\u0000\x7f",
			},
			want: want{
				val: "prefix:\"\\r\\t\\x00\\x7f\"",
			},
		},
		{
			name: "escape",
			args: args{
				buf: []byte("prefix:"),
				s:   "\u001b[2J",
			},
			want: want{
				val: "prefix:\"\\x1b[2J\"",
			},
		},
		{
			name: "nonbreaking space",
			args: args{
				buf: []byte("prefix:"),
				s:   "a b",
			},
			want: want{
				val: "prefix:\"a\\u00a0b\"",
			},
		},
		{
			name: "line separator",
			args: args{
				buf: []byte("prefix:"),
				s:   "a b",
			},
			want: want{
				val: "prefix:\"a\\u2028b\"",
			},
		},
		{
			name: "format",
			args: args{
				buf: []byte("prefix:"),
				s:   "a\u200bb",
			},
			want: want{
				val: "prefix:\"a\\u200bb\"",
			},
		},
		{
			name: "invalid UTF-8",
			args: args{
				s: "a\xff",
			},
			want: want{
				val: `"a\xff"`,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := escapeMessage(test.args.buf, test.args.s)
			assertBytes(t, got, test.want.val, "message")
		})
	}
}
