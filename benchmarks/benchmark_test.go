package main

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/nekrassov01/log"
)

// BenchmarkCLIHandler_Disabled measures filtering before attribute resolution.
// Attribute construction is outside the timed loop, as in the enabled cases.
func BenchmarkCLIHandler_Disabled(b *testing.B) {
	for _, test := range testDisabledCases() {
		b.Run(test.name, func(b *testing.B) {
			l := slog.New(log.NewCLIHandler(io.Discard, log.WithLevel(slog.LevelError)))
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				l.LogAttrs(ctx, slog.LevelInfo, testMessage, test.attrs...)
			}
		})
	}
}

// BenchmarkCLIHandler_Message isolates message formatting without optional built-ins.
func BenchmarkCLIHandler_Message(b *testing.B) {
	for _, test := range testMessages() {
		b.Run(test.name, func(b *testing.B) {
			l := slog.New(log.NewCLIHandler(io.Discard))
			b.ReportAllocs()
			for b.Loop() {
				l.Info(test.value)
			}
		})
	}
}

// BenchmarkCLIHandler_BuiltIns isolates the optional timestamp, source, and label.
// Source cases include warm-cache lookups from the same logging call site.
func BenchmarkCLIHandler_BuiltIns(b *testing.B) {
	for _, test := range testBuiltIns() {
		b.Run(test.name, func(b *testing.B) {
			l := slog.New(log.NewCLIHandler(io.Discard, test.opts...))
			b.ReportAllocs()
			for b.Loop() {
				l.Info(testMessage)
			}
		})
	}
}

// BenchmarkCLIHandler_AttrType measures each value's resolution and formatting.
// Values, including Any boxing, are constructed before timing begins.
func BenchmarkCLIHandler_AttrType(b *testing.B) {
	for _, test := range testAttrTypes() {
		b.Run(test.name, func(b *testing.B) {
			l := slog.New(log.NewCLIHandler(io.Discard, log.WithTimeLayout(time.RFC3339Nano)))
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				l.LogAttrs(ctx, slog.LevelInfo, testMessage, test.attr)
			}
		})
	}
}

// BenchmarkCLIHandler_AttrCount measures flat attribute lists, including slog's
// transition from five inline record attributes to additional allocated storage.
func BenchmarkCLIHandler_AttrCount(b *testing.B) {
	for _, count := range testAttrCounts() {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			attrs := testAttrs(count)
			h := log.NewCLIHandler(io.Discard)
			b.Run("AttrsAtSetup", func(b *testing.B) {
				l := slog.New(h.WithAttrs(attrs))
				b.ReportAllocs()
				for b.Loop() {
					l.Info(testMessage)
				}
			})
			b.Run("AttrsAtWrite", func(b *testing.B) {
				l := slog.New(h)
				ctx := context.Background()
				b.ReportAllocs()
				for b.Loop() {
					l.LogAttrs(ctx, slog.LevelInfo, testMessage, attrs...)
				}
			})
		})
	}
}

// BenchmarkCLIHandler_GroupDepth measures nested attribute traversal on each write.
// Construction is not timed; the input has one leaf regardless of depth.
func BenchmarkCLIHandler_GroupDepth(b *testing.B) {
	for _, depth := range testGroupDepths() {
		b.Run(strconv.Itoa(depth), func(b *testing.B) {
			l := slog.New(log.NewCLIHandler(io.Discard))
			attr := testGroup(depth)
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				l.LogAttrs(ctx, slog.LevelInfo, testMessage, attr)
			}
		})
	}
}

// BenchmarkCLIHandler_AttrReplacer separates traversal from replacement work.
func BenchmarkCLIHandler_AttrReplacer(b *testing.B) {
	attrs := testReplacementAttrs()
	for _, test := range testReplacers() {
		b.Run(test.name, func(b *testing.B) {
			l := slog.New(log.NewCLIHandler(io.Discard, log.WithAttrReplacer(test.replacer)))
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				l.LogAttrs(ctx, slog.LevelInfo, testMessage, attrs...)
			}
		})
	}
}

// BenchmarkCLIHandler_WithAttrs includes the cost of deriving a handler and
// preformatting its attributes. It does not include writing a record.
func BenchmarkCLIHandler_WithAttrs(b *testing.B) {
	for _, test := range testAttrTypes() {
		b.Run(test.name, func(b *testing.B) {
			h := log.NewCLIHandler(io.Discard, log.WithTimeLayout(time.RFC3339Nano)).
				WithAttrs(testCachedAttrs())
			attrs := []slog.Attr{test.attr}
			b.ReportAllocs()
			for b.Loop() {
				h.WithAttrs(attrs)
			}
		})
	}
}

// BenchmarkCLIHandler_Parallel measures shared-handler throughput separately from
// serial latency. Writer includes the output lock that io.Discard bypasses.
func BenchmarkCLIHandler_Parallel(b *testing.B) {
	attrs := testAttrs(5)
	for _, test := range testWriters() {
		b.Run(test.name, func(b *testing.B) {
			h := log.NewCLIHandler(test.w, testParallelOptions()...)
			b.Run("AttrsAtSetup", func(b *testing.B) {
				l := slog.New(h.WithAttrs(attrs))
				b.ReportAllocs()
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						l.Info(testMessage)
					}
				})
			})
			b.Run("AttrsAtWrite", func(b *testing.B) {
				l := slog.New(h)
				ctx := context.Background()
				b.ReportAllocs()
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						l.LogAttrs(ctx, slog.LevelInfo, testMessage, attrs...)
					}
				})
			})
		})
	}
}
