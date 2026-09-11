package log

import (
	"bytes"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"
)

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
