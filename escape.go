package log

import (
	"strconv"
	"unicode/utf8"
)

// safeASCII avoids repeated character comparisons for ordinary attribute text.
var safeASCII = func() [256]bool {
	var safe [256]bool
	for b := byte('!'); b < '\x7f'; b++ {
		safe[b] = b != '=' && b != '\\' && b != '"'
	}
	return safe
}()

// escapeText appends s, quoting the whole value when it contains attribute
// delimiters, control characters, non-printing runes, or invalid UTF-8.
// It reports whether quotes were added; an empty s appends nothing.
func escapeText(buf []byte, s string) ([]byte, bool) {
	for i := 0; i < len(s); {
		b := s[i]
		if safeASCII[b] {
			i++
			continue
		}
		if b < utf8.RuneSelf {
			return strconv.AppendQuote(buf, s), true
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 || !strconv.IsPrint(r) {
			return strconv.AppendQuote(buf, s), true
		}
		i += size
	}
	return append(buf, s...), false
}

// escapeMessage appends s, preserving printable spaces and punctuation.
// Control characters, non-printing runes, or invalid UTF-8 cause the whole message
// to be quoted using Go string escapes.
func escapeMessage(buf []byte, s string) []byte {
	for i := 0; i < len(s); {
		b := s[i]
		if b >= ' ' && b < '\x7f' {
			i++
			continue
		}
		if b < utf8.RuneSelf {
			return strconv.AppendQuote(buf, s)
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 || !strconv.IsPrint(r) {
			return strconv.AppendQuote(buf, s)
		}
		i += size
	}
	return append(buf, s...)
}
