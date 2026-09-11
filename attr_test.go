package log

import (
	"log/slog"
	"strings"
	"testing"
)

func Test_attrState_prepare(t *testing.T) {
	type fields struct {
		path   []byte
		groups []string
	}
	type args struct {
		groups []string
	}
	type want struct {
		path   string
		groups []string
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
			name: "empty",
			args: args{
				groups: []string{},
			},
		},
		{
			name: "replace previous",
			fields: fields{
				path:   []byte("old."),
				groups: []string{"old"},
			},
			args: args{
				groups: []string{"new", "inner"},
			},
			want: want{
				path:   "new.inner.",
				groups: []string{"new", "inner"},
			},
		},
		{
			name: "ignore empty groups",
			args: args{
				groups: []string{"", "a", "", "b"},
			},
			want: want{
				path:   "a.b.",
				groups: []string{"a", "b"},
			},
		},
		{
			name: "escaped group",
			args: args{
				groups: []string{"a b", "c\n"},
			},
			want: want{
				path:   "\"a b\".\"c\\n\".",
				groups: []string{"a b", "c\n"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &attrState{
				path:   test.fields.path,
				groups: test.fields.groups,
			}
			o.prepare(test.args.groups)
			assertBytes(t, o.path, test.want.path, "path")
			assertValue(t, o.groups, test.want.groups, "groups")
		})
	}
}

func Test_attrState_resolve(t *testing.T) {
	type fields struct {
		path   []byte
		groups []string
	}
	type args struct {
		attr     slog.Attr
		replacer AttrReplacer
		yield    func(*[]testAttr) func(slog.Attr, slog.Kind, []byte)
	}
	type want struct {
		attrs  []testAttr
		path   string
		groups []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty attribute",
			args: args{
				attr:  slog.Attr{},
				yield: testAttrCapture,
			},
		},
		{
			name: "nil Any with key",
			args: args{
				attr:  slog.Any("nil", nil),
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Any("nil", nil),
						kind: slog.KindAny,
						path: "",
					},
				},
			},
		},
		{
			name: "empty string with no key",
			args: args{
				attr:  slog.String("", ""),
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.String("", ""),
						kind: slog.KindString,
						path: "",
					},
				},
			},
		},
		{
			name: "ordinary",
			args: args{
				attr:  slog.Int("n", 1),
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Int("n", 1),
						kind: slog.KindInt64,
						path: "",
					},
				},
			},
		},
		{
			name: "resolve before replacing",
			args: args{
				attr: slog.Any("k", testValue(func() slog.Value {
					return slog.IntValue(2)
				})),
				replacer: func(_ []string, a slog.Attr) slog.Attr {
					return slog.Int64(a.Key, a.Value.Int64()+1)
				},
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Int("k", 3),
						kind: slog.KindInt64,
						path: "",
					},
				},
			},
		},
		{
			name: "resolve after replacing",
			args: args{
				attr: slog.Int("k", 1),
				replacer: func(_ []string, a slog.Attr) slog.Attr {
					return slog.Any(a.Key, testValue(func() slog.Value {
						return slog.StringValue("resolved")
					}))
				},
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.String("k", "resolved"),
						kind: slog.KindString,
						path: "",
					},
				},
			},
		},
		{
			name: "drop replacement",
			args: args{
				attr: slog.Int("k", 1),
				replacer: func([]string, slog.Attr) slog.Attr {
					return slog.Attr{}
				},
				yield: testAttrCapture,
			},
		},
		{
			name: "empty group",
			args: args{
				attr:  slog.Group("g"),
				yield: testAttrCapture,
			},
			want: want{
				groups: []string{},
			},
		},
		{
			name: "inline group",
			args: args{
				attr:  slog.Group("", slog.Int("a", 1)),
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Int("a", 1),
						kind: slog.KindInt64,
						path: "",
					},
				},
			},
		},
		{
			name: "nested and siblings",
			args: args{
				attr:  slog.Group("g", slog.Group("h", slog.Int("a", 1)), slog.Int("b", 2)),
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Int("a", 1),
						kind: slog.KindInt64,
						path: "g.h.",
					},
					{
						attr: slog.Int("b", 2),
						kind: slog.KindInt64,
						path: "g.",
					}},
				groups: []string{},
			},
		},
		{
			name: "replacement expands group",
			args: args{
				attr: slog.Int("k", 1),
				replacer: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == "k" {
						return slog.Group("g", slog.Int("child", 2))
					}
					return a
				},
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Int("child", 2),
						kind: slog.KindInt64,
						path: "g.",
					},
				},
				groups: []string{},
			},
		},
		{
			name: "original group names in callback",
			fields: fields{
				path:   []byte("root."),
				groups: []string{"root"},
			},
			args: args{
				attr: slog.Group("a b", slog.Int("k", 1)),
				replacer: func(groups []string, a slog.Attr) slog.Attr {
					a.Key = strings.Join(groups, "/")
					return a
				},
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.Int("root/a b", 1),
						kind: slog.KindInt64,
						path: "root.\"a b\".",
					},
				},
				path:   "root.",
				groups: []string{"root"},
			},
		},
		{
			name: "LogValuer resolves to a group",
			args: args{
				attr: slog.Any("g", testValue(func() slog.Value {
					return slog.GroupValue(slog.Any("nested", testValue(func() slog.Value {
						return slog.StringValue("value")
					})))
				})),
				yield: testAttrCapture,
			},
			want: want{
				attrs: []testAttr{
					{
						attr: slog.String("nested", "value"),
						kind: slog.KindString,
						path: "g.",
					},
				},
				groups: []string{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &attrState{
				path:   test.fields.path,
				groups: test.fields.groups,
			}
			var got []testAttr
			o.resolve(test.args.attr, test.args.replacer, test.args.yield(&got))
			assertAttrs(t, got, test.want.attrs)
			assertBytes(t, o.path, test.want.path, "path")
			assertValue(t, o.groups, test.want.groups, "groups")
		})
	}
}

func Test_attrState_resolve_failure(t *testing.T) {
	type fields struct {
		path   []byte
		groups []string
	}
	type args struct {
		attr slog.Attr
	}
	type want struct {
		message string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "panicking LogValuer",
			args: args{
				attr: slog.Any("k", testValue(func() slog.Value {
					panic("failure")
				})),
			},
			want: want{
				message: "LogValue panicked",
			},
		},
		{
			name: "recursive LogValuer",
			args: args{
				attr: slog.Any("k", testRecursiveValue{}),
			},
			want: want{
				message: "LogValue called too many times",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &attrState{
				path:   test.fields.path,
				groups: test.fields.groups,
			}
			var got []testAttr
			o.resolve(test.args.attr, nil, testAttrCapture(&got))
			assertValue(t, len(got), 1, "attribute count")
			assertValue(t, got[0].kind, slog.KindAny, "kind")
			assertValue(t, strings.Contains(got[0].attr.Value.String(), test.want.message), true, "resolution error")
		})
	}
}

func Test_attrState_pushGroup(t *testing.T) {
	type fields struct {
		path   []byte
		groups []string
	}
	type args struct {
		group string
	}
	type want struct {
		path   string
		groups []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty",
		},
		{
			name: "first",
			args: args{
				group: "a",
			},
			want: want{
				path:   "a.",
				groups: []string{"a"},
			},
		},
		{
			name: "nested",
			fields: fields{
				path:   []byte("a."),
				groups: []string{"a"},
			},
			args: args{
				group: "b",
			},
			want: want{
				path:   "a.b.",
				groups: []string{"a", "b"},
			},
		},
		{
			name: "escaped",
			args: args{
				group: "a=b",
			},
			want: want{
				path:   "\"a=b\".",
				groups: []string{"a=b"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &attrState{
				path:   test.fields.path,
				groups: test.fields.groups,
			}
			o.pushGroup(test.args.group)
			assertBytes(t, o.path, test.want.path, "path")
			assertValue(t, o.groups, test.want.groups, "groups")
		})
	}
}

func Test_attrState_popGroup(t *testing.T) {
	type fields struct {
		path   []byte
		groups []string
	}
	type args struct {
		group   string
		pathLen int
	}
	type want struct {
		path   string
		groups []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "empty group",
		},
		{
			name: "last",
			fields: fields{
				path:   []byte("a."),
				groups: []string{"a"},
			},
			args: args{
				group: "a",
			},
			want: want{
				groups: []string{},
			},
		},
		{
			name: "nested escaped",
			fields: fields{
				path:   []byte("a.\"b c\"."),
				groups: []string{"a", "b c"},
			},
			args: args{
				group:   "b c",
				pathLen: 2,
			},
			want: want{
				path:   "a.",
				groups: []string{"a"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &attrState{
				path:   test.fields.path,
				groups: test.fields.groups,
			}
			o.popGroup(test.args.group, test.args.pathLen)
			assertBytes(t, o.path, test.want.path, "path")
			assertValue(t, o.groups, test.want.groups, "groups")
		})
	}
}

func Test_attrState_reset(t *testing.T) {
	type fields struct {
		path   []byte
		groups []string
	}
	type want struct {
		pathCap  int
		groupCap int
		groups   []string
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
			name: "populated",
			fields: fields{
				path:   []byte("a.b."),
				groups: []string{"a", "b"},
			},
			want: want{
				pathCap:  4,
				groupCap: 2,
				groups:   []string{"", ""},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &attrState{
				path:   test.fields.path,
				groups: test.fields.groups,
			}
			groups := o.groups
			o.reset()
			assertValue(t, len(o.path), 0, "path length")
			assertValue(t, len(o.groups), 0, "group length")
			assertValue(t, cap(o.path), test.want.pathCap, "path capacity")
			assertValue(t, cap(o.groups), test.want.groupCap, "group capacity")
			assertValue(t, groups, test.want.groups, "cleared references")
		})
	}
}
