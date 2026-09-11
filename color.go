package log

import "strconv"

// SGR (Select Graphic Rendition) codes accepted by NewColor.
// Fg and Bg select foreground and background colors; Hi selects bright colors.
const (
	CodeReset        = 0
	CodeBold         = 1
	CodeFaint        = 2
	CodeItalic       = 3
	CodeUnderline    = 4
	CodeBlinkSlow    = 5
	CodeBlinkRapid   = 6
	CodeReverseVideo = 7
	CodeConcealed    = 8
	CodeCrossedOut   = 9

	CodeFgBlack   = 30
	CodeFgRed     = 31
	CodeFgGreen   = 32
	CodeFgYellow  = 33
	CodeFgBlue    = 34
	CodeFgMagenta = 35
	CodeFgCyan    = 36
	CodeFgWhite   = 37

	CodeFgHiBlack   = 90
	CodeFgHiRed     = 91
	CodeFgHiGreen   = 92
	CodeFgHiYellow  = 93
	CodeFgHiBlue    = 94
	CodeFgHiMagenta = 95
	CodeFgHiCyan    = 96
	CodeFgHiWhite   = 97

	CodeBgBlack   = 40
	CodeBgRed     = 41
	CodeBgGreen   = 42
	CodeBgYellow  = 43
	CodeBgBlue    = 44
	CodeBgMagenta = 45
	CodeBgCyan    = 46
	CodeBgWhite   = 47

	CodeBgHiBlack   = 100
	CodeBgHiRed     = 101
	CodeBgHiGreen   = 102
	CodeBgHiYellow  = 103
	CodeBgHiBlue    = 104
	CodeBgHiMagenta = 105
	CodeBgHiCyan    = 106
	CodeBgHiWhite   = 107
)

// Color holds immutable SGR sequences for text styling.
// A nil *Color or a zero-valued Color adds no color.
type Color struct {
	prefix []byte
	suffix []byte
}

// NewColor returns a color that applies codes before text and resets SGR state after it.
// Codes are used in the given order without validation. Passing no codes disables coloring.
func NewColor(codes ...int) *Color {
	if len(codes) == 0 {
		return &Color{
			prefix: nil,
			suffix: nil,
		}
	}
	return &Color{
		prefix: buildSGR(codes),
		suffix: buildSGR([]int{CodeReset}),
	}
}

// appendText appends s with SGR decoration when enabled and a color is configured.
// Empty text appends nothing, including no escape sequences.
func (o *Color) appendText(dst []byte, s string, enabled bool) []byte {
	if s == "" {
		return dst
	}
	if !enabled || o == nil || len(o.prefix) == 0 {
		return append(dst, s...)
	}
	dst = append(dst, o.prefix...)
	dst = append(dst, s...)
	return append(dst, o.suffix...)
}

// appendPrefix appends the configured SGR prefix when enabled.
func (o *Color) appendPrefix(dst []byte, enabled bool) []byte {
	if !enabled || o == nil {
		return dst
	}
	return append(dst, o.prefix...)
}

// appendSuffix appends the configured SGR suffix when enabled.
func (o *Color) appendSuffix(dst []byte, enabled bool) []byte {
	if !enabled || o == nil {
		return dst
	}
	return append(dst, o.suffix...)
}

// buildSGR builds the SGR escape sequence for the given codes.
func buildSGR(codes []int) []byte {
	if len(codes) == 0 {
		return nil
	}
	b := make([]byte, 0, 16)
	b = append(b, '\x1b', '[')
	for i, code := range codes {
		if i > 0 {
			b = append(b, ';')
		}
		b = strconv.AppendInt(b, int64(code), 10)
	}
	b = append(b, 'm')
	return b
}
