package log

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestCLIHandler_Nonterminal checks that terminal detection disables configured
// colors in both cached and per-record output through the public handler API.
func TestCLIHandler_Nonterminal(t *testing.T) {
	type args struct {
		open func(*testing.T) (*os.File, *os.File)
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
			name: "file",
			args: args{
				open: setupFileOutput,
			},
			want: want{
				val: "1970-01-01T00:00:01Z INF APP message job.cached=value job.count=1\n",
			},
		},
		{
			name: "pipe",
			args: args{
				open: setupPipeOutput,
			},
			want: want{
				val: "1970-01-01T00:00:01Z INF APP message job.cached=value job.count=1\n",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r, w := test.args.open(t)
			handler := NewCLIHandler(w, WithTime(), WithLabel("APP"), WithStyle(NewStyle(
				WithTimeStyle(TimeStyle{
					Color: NewColor(CodeFgRed),
				}),
				WithMessageStyle(MessageStyle{
					Color: NewColor(CodeFgGreen),
				}),
				WithAttrStyle(AttrStyle{
					KeyColor:   NewColor(CodeFgBlue),
					ValueColor: NewColor(CodeFgYellow),
					Separator:  "=",
				}),
			))).WithGroup("job").WithAttrs([]slog.Attr{slog.String("cached", "value")})
			err := handler.Handle(t.Context(), testRecord(testTime(), slog.LevelInfo, "message", 0, slog.Int("count", 1)))
			assertError(t, err, nil)
			assertError(t, w.Close(), nil)
			got, err := io.ReadAll(r)
			assertError(t, err, nil)
			assertBytes(t, got, test.want.val, "output")
		})
	}
}

// TestCLIHandler_Concurrent checks that derived handlers serialize complete records
// through their shared writer, including when source caching is enabled.
func TestCLIHandler_Concurrent(t *testing.T) {
	type args struct {
		opts  []CLIHandlerOption
		count int
	}
	type want struct {
		lines int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "shared writer",
			args: args{
				count: 64,
			},
			want: want{
				lines: 64,
			},
		},
		{
			name: "shared writer and source",
			args: args{
				opts:  []CLIHandlerOption{WithSourcePath()},
				count: 64,
			},
			want: want{
				lines: 64,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			parent := NewCLIHandler(buf, test.args.opts...)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := range test.args.count {
				logger := slog.New(parent.WithGroup("worker" + strconv.Itoa(i)).WithAttrs([]slog.Attr{slog.String("cached", "value")}))
				wg.Go(func() {
					<-start
					logger.LogAttrs(t.Context(), slog.LevelInfo, "message", slog.Int("id", i))
				})
			}
			close(start)
			wg.Wait()
			assertConcurrentOutput(t, buf.String(), test.want.lines)
		})
	}
}

// TestCLIHandler_Reentrant checks that attribute replacement can log through the
// parent without deadlocking or overwriting the outer operation's pooled buffers.
func TestCLIHandler_Reentrant(t *testing.T) {
	type args struct {
		replacer func(**slog.Logger) AttrReplacer
		cached   []slog.Attr
		attrs    []slog.Attr
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
			name: "record replacement",
			args: args{
				replacer: testReentrantReplacer,
				attrs:    []slog.Attr{slog.Int("trigger", 1)},
			},
			want: want{
				val: "INF inner\nINF outer trigger=1\n",
			},
		},
		{
			name: "cached replacement",
			args: args{
				replacer: testReentrantReplacer,
				cached:   []slog.Attr{slog.Int("trigger", 1)},
			},
			want: want{
				val: "INF inner\nINF outer trigger=1\n",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			var logger *slog.Logger
			parent := NewCLIHandler(buf, WithAttrReplacer(test.args.replacer(&logger)))
			logger = slog.New(parent)
			done := make(chan struct{})
			go func() {
				handler := parent.WithAttrs(test.args.cached)
				slog.New(handler).LogAttrs(t.Context(), slog.LevelInfo, "outer", test.args.attrs...)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("reentrant logging did not complete")
			}
			assertBytes(t, buf.Bytes(), test.want.val, "output")
		})
	}
}
