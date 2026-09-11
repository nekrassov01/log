package log

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testValue allows each case to control value resolution.
type testValue func() slog.Value

// LogValue delegates to the case-specific resolver.
func (o testValue) LogValue() slog.Value {
	return o()
}

// testRecursiveValue exercises slog's bounded LogValuer resolution.
type testRecursiveValue struct{}

// LogValue returns the receiver again to exercise slog's resolution limit.
func (o testRecursiveValue) LogValue() slog.Value {
	return slog.AnyValue(o)
}

// testStringer supplies the Stringer branch of ordinary Any values.
type testStringer string

// String returns the fixture's underlying text.
func (o testStringer) String() string {
	return string(o)
}

// testAttr records a yielded attribute and an owned copy of its group path.
type testAttr struct {
	attr slog.Attr
	kind slog.Kind
	path string
}

// testWriter records writes and can return short writes or errors.
type testWriter struct {
	bytes.Buffer

	count *int
	err   error
}

// Write records all bytes, then reports the configured count and error.
// A nil count reports the number of bytes recorded.
func (o *testWriter) Write(p []byte) (int, error) {
	n, _ := o.Buffer.Write(p)
	if o.count != nil {
		n = *o.count
	}
	return n, o.err
}

// setupPCs supplies valid, distinct runtime locations without hard-coded addresses.
func setupPCs(t *testing.T) []uintptr {
	t.Helper()
	pcs := make([]uintptr, 2)
	if runtime.Callers(1, pcs) != len(pcs) {
		t.Fatal("could not capture source locations")
	}
	return pcs
}

// setupConcurrentSourceValue returns a source-value selector for concurrent cache-miss tests.
// It waits for both lookups before returning the file path, ensuring that both
// miss the cache and attempt to publish an entry for the same location.
func setupConcurrentSourceValue(t *testing.T) func(*slog.Source) string {
	t.Helper()
	var calls atomic.Int32
	ready := make(chan struct{})
	return func(source *slog.Source) string {
		if calls.Add(1) == 2 {
			close(ready)
		}
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			t.Error("concurrent cache misses did not reach the barrier")
		}
		return source.File
	}
}

// setupFile supplies a non-terminal file with automatic cleanup.
func setupFile(t *testing.T) io.Writer {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "log")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})
	return f
}

// testDefaultStyle records the public default independently of DefaultStyle.
func testDefaultStyle() *Style {
	return &Style{
		level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text:  "DBG",
				Color: NewColor(38, 2, 95, 95, 255, CodeBold),
			},
			slog.LevelInfo: {
				Text:  "INF",
				Color: NewColor(38, 2, 95, 255, 215, CodeBold),
			},
			slog.LevelWarn: {
				Text:  "WRN",
				Color: NewColor(38, 2, 215, 255, 135, CodeBold),
			},
			slog.LevelError: {
				Text:  "ERR",
				Color: NewColor(38, 2, 255, 95, 135, CodeBold),
			},
		},
		source: SourceStyle{
			Prefix: AffixStyle{
				Text:  "<",
				Color: NewColor(CodeFgHiBlack),
			},
			Suffix: AffixStyle{
				Text:  ">",
				Color: NewColor(CodeFgHiBlack),
			},
			Color: NewColor(CodeFgHiBlack, CodeUnderline),
		},
		label: LabelStyle{
			Color: NewColor(CodeFgHiBlack, CodeBold),
		},
		attr: AttrStyle{
			KeyColor:  NewColor(CodeFgHiBlack),
			Separator: "=",
		},
	}
}

// testConfig constructs configuration for tests of its consumers.
// Unlike NewCLIHandler, it requires every supplied option to have an apply function.
func testConfig(colored bool, opts ...CLIHandlerOption) config {
	o := newOption()
	for _, opt := range opts {
		opt.apply(&o)
	}
	return newConfig(&o, colored)
}

// testRecord fixes all record metadata instead of depending on the current time.
func testRecord(at time.Time, level slog.Level, message string, pc uintptr, attrs ...slog.Attr) slog.Record {
	record := slog.NewRecord(at, level, message, pc)
	record.AddAttrs(attrs...)
	return record
}

// testTime supplies a fixed timestamp for built-in and ordinary time attributes.
func testTime() time.Time {
	return time.Unix(1, 123456789).UTC()
}

// testSource supplies one deterministic, already-cached source at PC 1.
// Other non-zero program counters are unsupported because no selector is installed.
func testSource(value string, line int) *source {
	return &source{
		entries: map[uintptr]*sourceEntry{
			1: newSourceEntry(1, value, line),
		},
	}
}

// testSourceValue observes either an absent selector or its selected source text.
func testSourceValue(value func(*slog.Source) string) string {
	if value == nil {
		return ""
	}
	return value(&slog.Source{
		File:     "file.go",
		Function: "pkg.Func",
	})
}

// testSourceText derives runtime-dependent expectations independently of the cache.
func testSourceText(pc uintptr, function bool) string {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	value := frame.File
	if function {
		value = frame.Function
	}
	return value + ":" + strconv.Itoa(frame.Line)
}

// testAttrCapture returns a callback that snapshots attributes and copies each path
// before traversal reuses the path buffer.
func testAttrCapture(got *[]testAttr) func(slog.Attr, slog.Kind, []byte) {
	return func(attr slog.Attr, kind slog.Kind, path []byte) {
		*got = append(*got, testAttr{
			attr: attr,
			kind: kind,
			path: string(path),
		})
	}
}

// testLargeGroups exceeds both retained group and path capacities.
func testLargeGroups() []string {
	groups := make([]string, 65)
	groups[0] = strings.Repeat("g", 4097)
	for i := 1; i < len(groups); i++ {
		groups[i] = "g"
	}
	return groups
}

// testReentrantReplacer returns a replacer that logs through the parent on a trigger.
// The inner record has no attributes, so replacement does not recurse.
func testReentrantReplacer(logger **slog.Logger) AttrReplacer {
	return func(_ []string, attr slog.Attr) slog.Attr {
		if attr.Key == "trigger" {
			(*logger).Info("inner")
		}
		return attr
	}
}

// testError creates a generic error for tests.
func testError() error {
	return errors.New("test error")
}

// assertConcurrentOutput checks that every identifier occupies one complete line.
func assertConcurrentOutput(t *testing.T, output string, count int) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	assertValue(t, len(lines), count, "line count")
	seen := make(map[int]bool, count)
	for _, line := range lines {
		index := strings.LastIndex(line, "id=")
		if index < 0 {
			t.Errorf("missing identifier in %q", line)
			continue
		}
		id, err := strconv.Atoi(line[index+3:])
		if err != nil || id < 0 || id >= count || seen[id] {
			t.Errorf("invalid or duplicate identifier in %q", line)
		}
		seen[id] = true
	}
}

// assertBytes compares output without distinguishing empty and nil buffers.
func assertBytes(t *testing.T, got []byte, want string, name string) {
	t.Helper()
	if string(got) != want {
		t.Errorf("%s mismatch\ngot=%q\nwant=%q", name, got, want)
	}
}

// assertAttrs compares slog values by their public equality operation.
func assertAttrs(t *testing.T, got, want []testAttr) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("attribute count mismatch: got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if !got[i].attr.Equal(want[i].attr) || got[i].kind != want[i].kind || got[i].path != want[i].path {
			t.Errorf("attribute %d mismatch\ngot=%+v\nwant=%+v", i, got[i], want[i])
		}
	}
}

// assertCleared checks that retained group slots do not keep strings alive.
func assertCleared(t *testing.T, groups []string) {
	t.Helper()
	for _, group := range groups[:cap(groups)] {
		assertValue(t, group, "", "cleared group")
	}
}

// assertBacking checks retention without depending on sync.Pool's next result.
func assertBacking[T any](t *testing.T, got, before []T, retained bool) {
	t.Helper()
	if !retained {
		assertValue(t, got == nil, true, "discarded backing array")
		return
	}
	if cap(before) > 0 && (cap(got) == 0 || &got[:cap(got)][0] != &before[:cap(before)][0]) {
		t.Error("reusable backing array was replaced")
	}
}

// assertError preserves error identity, including a nil expected error.
func assertError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("error mismatch\ngot=%v\nwant=%v", got, want)
	}
}

// assertValue compares structural state without normalizing nil collections.
func assertValue(t *testing.T, got, want any, name string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s mismatch\ngot=%#v\nwant=%#v", name, got, want)
	}
}
