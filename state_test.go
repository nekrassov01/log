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
			assertValue(t, len(got.attr.path), 0, "path reset")
			assertValue(t, len(got.attr.groups), 0, "groups reset")
			assertValue(t, len(got.line.buf), 0, "line reset")
			assertValue(t, got.line.wrote, false, "wrote reset")
			releaseState(got)
		})
	}
}

func Test_releaseState(t *testing.T) {
	type fields struct {
		attr attrState
		line lineState
	}
	type want struct {
		pathCap       int
		groupCap      int
		lineCap       int
		pathRetained  bool
		groupRetained bool
		lineRetained  bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "zero",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 0),
					groups: make([]string, 0),
				},
				line: lineState{
					buf:   make([]byte, 0),
					wrote: true,
				},
			},
			want: want{
				pathCap:       0,
				groupCap:      0,
				lineCap:       0,
				pathRetained:  true,
				groupRetained: true,
				lineRetained:  true,
			},
		},
		{
			name: "ordinary",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 256),
					groups: make([]string, 8),
				},
				line: lineState{
					buf:   make([]byte, 1024),
					wrote: true,
				},
			},
			want: want{
				pathCap:       256,
				groupCap:      8,
				lineCap:       1024,
				pathRetained:  true,
				groupRetained: true,
				lineRetained:  true,
			},
		},
		{
			name: "below",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 4095),
					groups: make([]string, 63),
				},
				line: lineState{
					buf:   make([]byte, 65535),
					wrote: true,
				},
			},
			want: want{
				pathCap:       4095,
				groupCap:      63,
				lineCap:       65535,
				pathRetained:  true,
				groupRetained: true,
				lineRetained:  true,
			},
		},
		{
			name: "at",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 4096),
					groups: make([]string, 64),
				},
				line: lineState{
					buf:   make([]byte, 65536),
					wrote: true,
				},
			},
			want: want{
				pathCap:       4096,
				groupCap:      64,
				lineCap:       65536,
				pathRetained:  true,
				groupRetained: true,
				lineRetained:  true,
			},
		},
		{
			name: "path over",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 4097),
					groups: make([]string, 8),
				},
				line: lineState{
					buf:   make([]byte, 1024),
					wrote: true,
				},
			},
			want: want{
				pathCap:       0,
				groupCap:      8,
				lineCap:       1024,
				pathRetained:  false,
				groupRetained: true,
				lineRetained:  true,
			},
		},
		{
			name: "groups over",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 256),
					groups: make([]string, 65),
				},
				line: lineState{
					buf:   make([]byte, 1024),
					wrote: true,
				},
			},
			want: want{
				pathCap:       256,
				groupCap:      0,
				lineCap:       1024,
				pathRetained:  true,
				groupRetained: false,
				lineRetained:  true,
			},
		},
		{
			name: "line over",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 256),
					groups: make([]string, 8),
				},
				line: lineState{
					buf:   make([]byte, 65537),
					wrote: true,
				},
			},
			want: want{
				pathCap:       256,
				groupCap:      8,
				lineCap:       0,
				pathRetained:  true,
				groupRetained: true,
				lineRetained:  false,
			},
		},
		{
			name: "all over",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 4097),
					groups: make([]string, 65),
				},
				line: lineState{
					buf:   make([]byte, 65537),
					wrote: true,
				},
			},
			want: want{
				pathCap:       0,
				groupCap:      0,
				lineCap:       0,
				pathRetained:  false,
				groupRetained: false,
				lineRetained:  false,
			},
		},
		{
			name: "empty oversized",
			fields: fields{
				attr: attrState{
					path:   make([]byte, 0, 4097),
					groups: make([]string, 0, 65),
				},
				line: lineState{
					buf:   make([]byte, 0, 65537),
					wrote: true,
				},
			},
			want: want{
				pathCap:       0,
				groupCap:      0,
				lineCap:       0,
				pathRetained:  false,
				groupRetained: false,
				lineRetained:  false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &state{
				attr: test.fields.attr,
				line: test.fields.line,
			}
			for i := range o.attr.groups {
				o.attr.groups[i] = "group"
			}
			path, groups, line := o.attr.path, o.attr.groups, o.line.buf
			releaseState(o)
			assertValue(t, len(o.attr.path), 0, "path length")
			assertValue(t, len(o.attr.groups), 0, "group length")
			assertValue(t, len(o.line.buf), 0, "line length")
			assertValue(t, o.line.wrote, false, "wrote")
			assertValue(t, cap(o.attr.path), test.want.pathCap, "path capacity")
			assertValue(t, cap(o.attr.groups), test.want.groupCap, "group capacity")
			assertValue(t, cap(o.line.buf), test.want.lineCap, "line capacity")
			assertBacking(t, o.attr.path, path, test.want.pathRetained)
			assertBacking(t, o.attr.groups, groups, test.want.groupRetained)
			assertBacking(t, o.line.buf, line, test.want.lineRetained)
			assertCleared(t, o.attr.groups)
		})
	}
}
