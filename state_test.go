package log

import "testing"

func Test_acquireState(t *testing.T) {
	type want struct {
		val bool
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "available",
			want: want{
				val: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := acquireState()
			assertValue(t, got != nil, test.want.val, "state available")
			assertValue(t, len(got.line.buf), 0, "line reset")
			assertValue(t, got.line.wrote, false, "wrote reset")
			assertValue(t, len(got.attr.path), 0, "path reset")
			assertValue(t, len(got.attr.groups), 0, "groups reset")
			releaseState(got)
		})
	}
}

func Test_releaseState(t *testing.T) {
	type fields struct {
		line lineState
		attr attrState
	}
	type want struct {
		lineCap       int
		pathCap       int
		groupCap      int
		lineRetained  bool
		pathRetained  bool
		groupRetained bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "zero",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 0),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 0),
					groups: make([]string, 0),
				},
			},
			want: want{
				lineCap:       0,
				pathCap:       0,
				groupCap:      0,
				lineRetained:  true,
				pathRetained:  true,
				groupRetained: true,
			},
		},
		{
			name: "ordinary",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 1024),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 256),
					groups: make([]string, 8),
				},
			},
			want: want{
				lineCap:       1024,
				pathCap:       256,
				groupCap:      8,
				lineRetained:  true,
				pathRetained:  true,
				groupRetained: true,
			},
		},
		{
			name: "below",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 65535),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 4095),
					groups: make([]string, 63),
				},
			},
			want: want{
				lineCap:       65535,
				pathCap:       4095,
				groupCap:      63,
				lineRetained:  true,
				pathRetained:  true,
				groupRetained: true,
			},
		},
		{
			name: "at",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 65536),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 4096),
					groups: make([]string, 64),
				},
			},
			want: want{
				lineCap:       65536,
				pathCap:       4096,
				groupCap:      64,
				lineRetained:  true,
				pathRetained:  true,
				groupRetained: true,
			},
		},
		{
			name: "line over",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 65537),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 256),
					groups: make([]string, 8),
				},
			},
			want: want{
				lineCap:       0,
				pathCap:       256,
				groupCap:      8,
				lineRetained:  false,
				pathRetained:  true,
				groupRetained: true,
			},
		},
		{
			name: "path over",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 1024),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 4097),
					groups: make([]string, 8),
				},
			},
			want: want{
				lineCap:       1024,
				pathCap:       0,
				groupCap:      8,
				lineRetained:  true,
				pathRetained:  false,
				groupRetained: true,
			},
		},
		{
			name: "groups over",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 1024),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 256),
					groups: make([]string, 65),
				},
			},
			want: want{
				lineCap:       1024,
				pathCap:       256,
				groupCap:      0,
				lineRetained:  true,
				pathRetained:  true,
				groupRetained: false,
			},
		},
		{
			name: "all over",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 65537),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 4097),
					groups: make([]string, 65),
				},
			},
			want: want{
				lineCap:       0,
				pathCap:       0,
				groupCap:      0,
				lineRetained:  false,
				pathRetained:  false,
				groupRetained: false,
			},
		},
		{
			name: "empty oversized",
			fields: fields{
				line: lineState{
					buf:   make([]byte, 0, 65537),
					wrote: true,
				},
				attr: attrState{
					path:   make([]byte, 0, 4097),
					groups: make([]string, 0, 65),
				},
			},
			want: want{
				lineCap:       0,
				pathCap:       0,
				groupCap:      0,
				lineRetained:  false,
				pathRetained:  false,
				groupRetained: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &state{
				line: test.fields.line,
				attr: test.fields.attr,
			}
			for i := range o.attr.groups {
				o.attr.groups[i] = "group"
			}
			line, path, groups := o.line.buf, o.attr.path, o.attr.groups
			releaseState(o)
			assertValue(t, len(o.line.buf), 0, "line length")
			assertValue(t, len(o.attr.path), 0, "path length")
			assertValue(t, len(o.attr.groups), 0, "group length")
			assertValue(t, o.line.wrote, false, "wrote")
			assertValue(t, cap(o.line.buf), test.want.lineCap, "line capacity")
			assertValue(t, cap(o.attr.path), test.want.pathCap, "path capacity")
			assertValue(t, cap(o.attr.groups), test.want.groupCap, "group capacity")
			assertBacking(t, o.line.buf, line, test.want.lineRetained)
			assertBacking(t, o.attr.path, path, test.want.pathRetained)
			assertBacking(t, o.attr.groups, groups, test.want.groupRetained)
			assertCleared(t, o.attr.groups)
		})
	}
}
