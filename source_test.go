package log

import (
	"log/slog"
	"sync"
	"testing"
)

func Test_newSource(t *testing.T) {
	type args struct {
		value func(*slog.Source) string
	}
	type want struct {
		absent bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "disabled",
			want: want{
				absent: true,
			},
		},
		{
			name: "enabled",
			args: args{
				value: func(s *slog.Source) string {
					return s.File
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newSource(test.args.value)
			assertValue(t, got == nil, test.want.absent, "cache absent")
		})
	}
}

func Test_source_resolve(t *testing.T) {
	type fields struct {
		last    *sourceEntry
		entries map[uintptr]*sourceEntry
		value   func(*slog.Source) string
	}
	type args struct {
		pcs func(*testing.T) []uintptr
	}
	type want struct {
		val     func([]uintptr) []string
		entries int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "zero PC",
			fields: fields{
				entries: make(map[uintptr]*sourceEntry),
			},
			args: args{
				pcs: func(*testing.T) []uintptr {
					return []uintptr{0}
				},
			},
			want: want{
				val: func([]uintptr) []string {
					return []string{""}
				},
			},
		},
		{
			name: "invalid PC",
			fields: fields{
				entries: make(map[uintptr]*sourceEntry),
			},
			args: args{
				pcs: func(*testing.T) []uintptr {
					return []uintptr{^uintptr(0)}
				},
			},
			want: want{
				val: func([]uintptr) []string {
					return []string{""}
				},
			},
		},
		{
			name: "last and map hits",
			fields: fields{
				entries: map[uintptr]*sourceEntry{
					1: {
						pc:   1,
						text: []byte("one"),
					},
					2: {
						pc:   2,
						text: []byte("two"),
					},
				},
				last: &sourceEntry{
					pc:   1,
					text: []byte("one"),
				},
				value: func(*slog.Source) string {
					panic("selector called on cache hit")
				},
			},
			args: args{
				pcs: func(*testing.T) []uintptr {
					return []uintptr{1, 2, 2, 1}
				},
			},
			want: want{
				val: func([]uintptr) []string {
					return []string{"one", "two", "two", "one"}
				},
				entries: 2,
			},
		},
		{
			name: "path misses",
			fields: fields{
				entries: make(map[uintptr]*sourceEntry),
				value: func(s *slog.Source) string {
					return s.File
				},
			},
			args: args{
				pcs: setupPCs,
			},
			want: want{
				val: func(pcs []uintptr) []string {
					return []string{testSourceText(pcs[0], false), testSourceText(pcs[1], false)}
				},
				entries: 2,
			},
		},
		{
			name: "function misses",
			fields: fields{
				entries: make(map[uintptr]*sourceEntry),
				value: func(s *slog.Source) string {
					return s.Function
				},
			},
			args: args{
				pcs: setupPCs,
			},
			want: want{
				val: func(pcs []uintptr) []string {
					return []string{testSourceText(pcs[0], true), testSourceText(pcs[1], true)}
				},
				entries: 2,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &source{
				entries: test.fields.entries,
				value:   test.fields.value,
			}
			o.last.Store(test.fields.last)
			pcs := test.args.pcs(t)
			var got []string
			for _, pc := range pcs {
				got = append(got, string(o.resolve(pc)))
			}
			assertValue(t, got, test.want.val(pcs), "sources")
			assertValue(t, len(o.entries), test.want.entries, "entries")
		})
	}
}

func Test_source_resolve_concurrent(t *testing.T) {
	type fields struct {
		value func(*testing.T) func(*slog.Source) string
	}
	type args struct {
		pc func(*testing.T) uintptr
	}
	type want struct {
		entries int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "simultaneous misses share published entry",
			fields: fields{
				value: setupConcurrentSourceValue,
			},
			args: args{
				pc: func(t *testing.T) uintptr {
					return setupPCs(t)[0]
				},
			},
			want: want{
				entries: 1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := &source{
				entries: make(map[uintptr]*sourceEntry),
				value:   test.fields.value(t),
			}
			pc := test.args.pc(t)
			var results [2][]byte
			var wg sync.WaitGroup
			for i := range results {
				wg.Go(func() {
					results[i] = o.resolve(pc)
				})
			}
			wg.Wait()
			assertValue(t, len(o.entries), test.want.entries, "entries")
			for _, result := range results {
				assertBytes(t, result, testSourceText(pc, false), "source")
				assertBacking(t, result, o.entries[pc].text, true)
			}
		})
	}
}

func Test_newSourceEntry(t *testing.T) {
	type args struct {
		pc    uintptr
		value string
		line  int
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
			name: "path",
			args: args{
				pc:    1,
				value: "/src/main.go",
				line:  83,
			},
			want: want{
				val: "/src/main.go:83",
			},
		},
		{
			name: "function",
			args: args{
				pc:    1,
				value: "pkg.(*Worker).Run",
				line:  83,
			},
			want: want{
				val: "pkg.(*Worker).Run:83",
			},
		},
		{
			name: "space",
			args: args{
				pc:    1,
				value: "a b.go",
				line:  83,
			},
			want: want{
				val: "\"a b.go:83\"",
			},
		},
		{
			name: "control",
			args: args{
				pc:    1,
				value: "a\n\u001b",
				line:  83,
			},
			want: want{
				val: "\"a\\n\\x1b:83\"",
			},
		},
		{
			name: "empty",
			args: args{
				pc:    1,
				value: "",
				line:  83,
			},
			want: want{
				val: ":83",
			},
		},
		{
			name: "invalid UTF-8",
			args: args{
				pc:    1,
				value: "a\xff",
				line:  83,
			},
			want: want{
				val: `"a\xff:83"`,
			},
		},
		{
			name: "zero line",
			args: args{
				value: "file",
			},
			want: want{
				val: "file:0",
			},
		},
		{
			name: "negative line",
			args: args{
				value: "file",
				line:  -1,
			},
			want: want{
				val: "file:-1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newSourceEntry(test.args.pc, test.args.value, test.args.line)
			assertValue(t, got.pc, test.args.pc, "PC")
			assertBytes(t, got.text, test.want.val, "text")
		})
	}
}
