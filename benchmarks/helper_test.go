package main

import (
	"errors"
	"io"
	"log/slog"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/nekrassov01/log"
)

// testMessage keeps a non-empty message constant across focused benchmarks.
const testMessage = "Processed input files and wrote the resulting output successfully."

// testDisabledCase describes one disabled workload.
type testDisabledCase struct {
	name  string
	attrs []slog.Attr
}

// testMessageCase describes one message workload.
type testMessageCase struct {
	name  string
	value string
}

// testBuiltInCase describes one built-in workload.
type testBuiltInCase struct {
	name string
	opts []log.CLIHandlerOption
}

// testAttrCase describes one value-formatting workload.
type testAttrCase struct {
	name string
	attr slog.Attr
}

// testReplacerCase describes one attribute-replacement workload.
type testReplacerCase struct {
	name     string
	replacer log.AttrReplacer
}

// testWriterCase describes one parallel workload.
type testWriterCase struct {
	name string
	w    io.Writer
}

// testObject models application data resolved into structured attributes on demand.
type testObject struct {
	name    string
	count   int
	enabled bool
}

// LogValue constructs a group on each resolution, making that cost part of the benchmark.
func (o testObject) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", o.name),
		slog.Int("count", o.count),
		slog.Bool("enabled", o.enabled),
	)
}

// testStringer provides an Any value with a String method.
type testStringer string

// String returns the fixture's text.
func (o testStringer) String() string {
	return string(o)
}

// testWriter is stateless and discards bytes while exercising the handler's output lock.
type testWriter struct{}

// Write reports a successful full write without performing operating-system I/O.
func (o testWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

// testDisabledCases supplies disabled benchmark inputs.
func testDisabledCases() []testDisabledCase {
	return []testDisabledCase{
		{
			name: "Message",
		},
		{
			name:  "Attrs",
			attrs: testAttrs(5),
		},
	}
}

// testMessages supplies message benchmark inputs.
func testMessages() []testMessageCase {
	return []testMessageCase{
		{
			name: "Empty",
		},
		{
			name:  "Text",
			value: testMessage,
		},
		{
			name:  "QuotesAndBackslashes",
			value: `copy "input.txt" to C:\output\result.txt`,
		},
		{
			name:  "Controls",
			value: "line one\nline two\t\x1b[31m",
		},
		{
			name:  "Unicode",
			value: "日本語のメッセージと絵文字 📦",
		},
		{
			name:  "NonPrintingUnicode",
			value: "before\u200bafter\u2028next",
		},
		{
			name:  "InvalidUTF8",
			value: "before\xffafter",
		},
		{
			name:  "Bytes/1024",
			value: strings.Repeat("x", 1024),
		},
		{
			name:  "Bytes/32768",
			value: strings.Repeat("x", 32768),
		},
		{
			name:  "Bytes/65536",
			value: strings.Repeat("x", 65536),
		},
	}
}

// testBuiltIns supplies built-in benchmark inputs.
func testBuiltIns() []testBuiltInCase {
	return []testBuiltInCase{
		{
			name: "None",
		},
		{
			name: "Time",
			opts: []log.CLIHandlerOption{log.WithTime()},
		},
		{
			name: "SourcePath",
			opts: []log.CLIHandlerOption{log.WithSourcePath()},
		},
		{
			name: "SourceFunction",
			opts: []log.CLIHandlerOption{log.WithSourceFunction()},
		},
		{
			name: "Label",
			opts: []log.CLIHandlerOption{log.WithLabel("APP")},
		},
		{
			name: "All",
			opts: []log.CLIHandlerOption{log.WithTime(), log.WithSourceFunction(), log.WithLabel("APP")},
		},
	}
}

// testAttrTypes supplies scalar, structured, and Any workloads for writing and setup.
func testAttrTypes() []testAttrCase {
	at := time.Date(2025, time.April, 1, 12, 34, 56, 123456789, time.UTC)
	err := errors.New("input could not be processed")
	return []testAttrCase{
		{
			name: "String/Empty",
			attr: slog.String("key", ""),
		},
		{
			name: "String/Text",
			attr: slog.String("key", "value"),
		},
		{
			name: "String/Quoted",
			attr: slog.String("key", "a b=c\n\t\"d\"\\e"),
		},
		{
			name: "String/Unicode",
			attr: slog.String("key", "日本語の値 📦"),
		},
		{
			name: "String/InvalidUTF8",
			attr: slog.String("key", "a\xffb"),
		},
		{
			name: "String/Long",
			attr: slog.String("key", strings.Repeat("x", 4096)),
		},
		{
			name: "Bool",
			attr: slog.Bool("key", true),
		},
		{
			name: "Int64",
			attr: slog.Int64("key", math.MinInt64),
		},
		{
			name: "Uint64",
			attr: slog.Uint64("key", math.MaxUint64),
		},
		{
			name: "Float64/Finite",
			attr: slog.Float64("key", -1.23456789012345),
		},
		{
			name: "Float64/NaN",
			attr: slog.Float64("key", math.NaN()),
		},
		{
			name: "Float64/Inf",
			attr: slog.Float64("key", math.Inf(1)),
		},
		{
			name: "Time/Zero",
			attr: slog.Time("key", time.Time{}),
		},
		{
			name: "Time/Nanoseconds",
			attr: slog.Time("key", at),
		},
		{
			name: "Duration",
			attr: slog.Duration("key", time.Hour+2*time.Minute+3*time.Second+4*time.Nanosecond),
		},
		{
			name: "Group",
			attr: slog.Group("key", slog.String("name", "input"), slog.Int("count", 42), slog.Bool("enabled", true)),
		},
		{
			name: "LogValuer/Object",
			attr: slog.Any("key", testObject{
				name:    "input",
				count:   42,
				enabled: true,
			}),
		},
		{
			name: "LogValuer/Objects",
			attr: slog.Group("key",
				slog.Any("first", testObject{
					name:    "first",
					count:   1,
					enabled: true,
				}),
				slog.Any("second", testObject{
					name:    "second",
					count:   2,
					enabled: false,
				}),
				slog.Any("third", testObject{
					name:    "third",
					count:   3,
					enabled: true,
				}),
			),
		},
		{
			name: "Any/Nil",
			attr: slog.Any("key", nil),
		},
		{
			name: "Any/Bytes",
			attr: slog.Any("key", []byte{0, 1, 31, 127, 128, 255}),
		},
		{
			name: "Any/Bools",
			attr: slog.Any("key", []bool{true, false, true, false}),
		},
		{
			name: "Any/Ints",
			attr: slog.Any("key", []int{0, -1, 42, 1234567890}),
		},
		{
			name: "Any/Floats",
			attr: slog.Any("key", []float64{0, -1.25, math.SmallestNonzeroFloat64, math.MaxFloat64}),
		},
		{
			name: "Any/Strings",
			attr: slog.Any("key", []string{"plain", "with spaces", "with\ncontrols", "日本語"}),
		},
		{
			name: "Any/Times",
			attr: slog.Any("key", []time.Time{at, at.Add(time.Second)}),
		},
		{
			name: "Any/Durations",
			attr: slog.Any("key", []time.Duration{time.Nanosecond, time.Second, time.Hour}),
		},
		{
			name: "Any/Error",
			attr: slog.Any("key", err),
		},
		{
			name: "Any/Errors",
			attr: slog.Any("key", []error{err, nil, err}),
		},
		{
			name: "Any/Stringer",
			attr: slog.Any("key", testStringer("formatted application value")),
		},
		{
			name: "Any/Struct",
			attr: slog.Any("key", struct {
				Name    string
				Count   int
				Enabled bool
			}{
				Name:    "input",
				Count:   42,
				Enabled: true,
			}),
		},
		{
			name: "Any/Map",
			attr: slog.Any("key", map[string]any{
				"name":    "input",
				"count":   42,
				"enabled": true,
			}),
		},
		{
			name: "Any/MixedSlice",
			attr: slog.Any("key", []any{"input", 42, true, at, err}),
		},
		{
			name: "Any/IPAddr",
			attr: slog.Any("key", netip.MustParseAddr("2001:db8::1")),
		},
		{
			name: "Any/IPPrefix",
			attr: slog.Any("key", netip.MustParsePrefix("2001:db8::/32")),
		},
	}
}

// testAttrCounts includes small records and larger flat attribute lists.
func testAttrCounts() []int {
	return []int{0, 1, 4, 5, 6, 16, 64, 256}
}

// testAttrs returns a flat list with stable keys, constructed outside timed loops.
func testAttrs(count int) []slog.Attr {
	attrs := make([]slog.Attr, count)
	for i := range attrs {
		attrs[i] = slog.Int("key"+strconv.Itoa(i), i)
	}
	return attrs
}

// testGroupDepths supplies nesting depths.
func testGroupDepths() []int {
	return []int{0, 1, 2, 4, 8, 16, 32, 64}
}

// testGroup wraps one leaf in depth groups.
func testGroup(depth int) slog.Attr {
	attr := slog.Int("key", 1)
	for range depth {
		attr = slog.GroupAttrs("group", attr)
	}
	return attr
}

// testReplacers supplies attribute-replacement benchmark inputs.
func testReplacers() []testReplacerCase {
	return []testReplacerCase{
		{
			name: "None",
		},
		{
			name: "Identity",
			replacer: func(_ []string, attr slog.Attr) slog.Attr {
				return attr
			},
		},
		{
			name: "Redact",
			replacer: func(_ []string, attr slog.Attr) slog.Attr {
				if attr.Key == "password" {
					return slog.String(attr.Key, "***")
				}
				return attr
			},
		},
		{
			name: "Drop",
			replacer: func(_ []string, _ slog.Attr) slog.Attr {
				return slog.Attr{}
			},
		},
		{
			name: "ExpandGroup",
			replacer: func(_ []string, attr slog.Attr) slog.Attr {
				if attr.Key == "password" {
					return slog.Group("credentials", slog.String("value", "***"), slog.Bool("redacted", true))
				}
				return attr
			},
		},
	}
}

// testReplacementAttrs includes unchanged and replaced leaves in a group.
func testReplacementAttrs() []slog.Attr {
	return []slog.Attr{
		slog.Group("request",
			slog.String("user", "alice"),
			slog.String("password", "secret"),
		),
	}
}

// testCachedAttrs supplies the existing attributes for handler derivation.
func testCachedAttrs() []slog.Attr {
	return []slog.Attr{slog.String("cached", "value")}
}

// testWriters supplies parallel benchmark inputs.
func testWriters() []testWriterCase {
	return []testWriterCase{
		{
			name: "Discard",
			w:    io.Discard,
		},
		{
			name: "Writer",
			w:    testWriter{},
		},
	}
}

// testParallelOptions enables built-in output for shared-handler workloads.
func testParallelOptions() []log.CLIHandlerOption {
	return []log.CLIHandlerOption{
		log.WithTime(),
		log.WithSourceFunction(),
		log.WithLabel("APP"),
	}
}
