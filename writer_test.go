package log

import (
	"bytes"
	"io"
	"testing"
)

func Test_newWriter(t *testing.T) {
	type args struct {
		w func(*testing.T) io.Writer
	}
	type want struct {
		discard  bool
		terminal bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil",
			args: args{
				w: func(*testing.T) io.Writer {
					return nil
				},
			},
			want: want{
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
				discard: true,
			},
		},
		{
			name: "buffer",
			args: args{
				w: func(*testing.T) io.Writer {
					return &bytes.Buffer{}
				},
			},
			want: want{
				discard: false,
			},
		},
		{
			name: "file",
			args: args{
				w: setupFile,
			},
			want: want{
				discard: false,
			},
		},
		{
			name: "pipe",
			args: args{
				w: setupPipe,
			},
			want: want{
				discard: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newWriter(test.args.w(t))
			assertValue(t, got.discard, test.want.discard, "discard")
			assertValue(t, got.terminal, test.want.terminal, "terminal")
			assertValue(t, got.w != nil, true, "writer available")
		})
	}
}

func Test_writer_write(t *testing.T) {
	type fields struct {
		w       func(*testing.T) *testWriter
		discard bool
	}
	type args struct {
		buf []byte
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
			name: "normal",
			fields: fields{
				w: func(*testing.T) *testWriter {
					return &testWriter{}
				},
			},
			args: args{
				buf: []byte("message"),
			},
			want: want{
				val: "message",
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "empty",
			fields: fields{
				w: func(*testing.T) *testWriter {
					return &testWriter{}
				},
			},
			want: want{
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "discard",
			fields: fields{
				w: func(*testing.T) *testWriter {
					return &testWriter{
						err: testError(),
					}
				},
				discard: true,
			},
			args: args{
				buf: []byte("message"),
			},
			want: want{
				err: func(*testWriter) error {
					return nil
				},
			},
		},
		{
			name: "error",
			fields: fields{
				w: func(*testing.T) *testWriter {
					return &testWriter{
						err: testError(),
					}
				},
			},
			args: args{
				buf: []byte("message"),
			},
			want: want{
				val: "message",
				err: func(w *testWriter) error {
					return w.err
				},
			},
		},
		{
			name: "short",
			fields: fields{
				w: func(*testing.T) *testWriter {
					n := 1
					return &testWriter{
						count: &n,
					}
				},
			},
			args: args{
				buf: []byte("message"),
			},
			want: want{
				val: "message",
				err: func(*testWriter) error {
					return io.ErrShortWrite
				},
			},
		},
		{
			name: "short with error",
			fields: fields{
				w: func(*testing.T) *testWriter {
					n := 0
					return &testWriter{
						count: &n,
						err:   testError(),
					}
				},
			},
			args: args{
				buf: []byte("message"),
			},
			want: want{
				val: "message",
				err: func(w *testWriter) error {
					return w.err
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := test.fields.w(t)
			wantErr := test.want.err(w)
			o := &writer{
				w:       w,
				discard: test.fields.discard,
			}
			got := o.write(test.args.buf)
			assertError(t, got, wantErr)
			assertBytes(t, w.Bytes(), test.want.val, "written")
		})
	}
}

func Test_resolveWriter(t *testing.T) {
	type args struct {
		w        func(*testing.T) io.Writer
		terminal bool
	}
	type want struct {
		same    bool
		discard bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil",
			args: args{
				w: func(*testing.T) io.Writer {
					return nil
				},
			},
			want: want{
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
				same:    true,
				discard: true,
			},
		},
		{
			name: "buffer",
			args: args{
				w: func(*testing.T) io.Writer {
					return &bytes.Buffer{}
				},
			},
			want: want{
				same: true,
			},
		},
		{
			name: "nonterminal file",
			args: args{
				w: setupFile,
			},
			want: want{
				same: true,
			},
		},
		{
			name: "nonterminal file with terminal flag",
			args: args{
				w:        setupFile,
				terminal: true,
			},
			want: want{
				// go-colorable checks the actual file and leaves non-terminals unchanged.
				same: true,
			},
		},
		{
			name: "pipe",
			args: args{
				w: setupPipe,
			},
			want: want{
				same: true,
			},
		},
		{
			name: "pipe with terminal flag",
			args: args{
				w:        setupPipe,
				terminal: true,
			},
			want: want{
				same: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := test.args.w(t)
			got := resolveWriter(w, test.args.terminal)
			assertValue(t, got == w, test.want.same, "same writer")
			assertValue(t, got == io.Discard, test.want.discard, "discard")
		})
	}
}

func Test_isTerminal(t *testing.T) {
	type args struct {
		w func(*testing.T) io.Writer
	}
	type want struct {
		val bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil",
			args: args{
				w: func(*testing.T) io.Writer {
					return nil
				},
			},
		},
		{
			name: "discard",
			args: args{
				w: func(*testing.T) io.Writer {
					return io.Discard
				},
			},
		},
		{
			name: "buffer",
			args: args{
				w: func(*testing.T) io.Writer {
					return &bytes.Buffer{}
				},
			},
		},
		{
			name: "file",
			args: args{
				w: setupFile,
			},
		},
		{
			name: "pipe",
			args: args{
				w: setupPipe,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isTerminal(test.args.w(t))
			assertValue(t, got, test.want.val, "terminal")
		})
	}
}
