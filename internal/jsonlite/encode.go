package jsonlite

import (
	"math"
	"strconv"
	"unicode/utf8"
)

const hexDigits = "0123456789abcdef"

// AppendString appends s as a JSON string as encoding/json's Marshal writes it:
// <, > and & as <, > and &, U+2028 and U+2029 escaped, and
// invalid UTF-8 as �.
func AppendString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			switch {
			case c == '"' || c == '\\':
				dst = append(dst, '\\', c)
			case c == '<' || c == '>' || c == '&':
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			case c < ' ':
				dst = appendControl(dst, c)
			default:
				dst = append(dst, c)
			}
			i++
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && n == 1:
			dst = append(dst, `�`...)
		case r == ' ' || r == ' ':
			dst = append(dst, '\\', 'u', '2', '0', '2', hexDigits[r&0xf])
		default:
			dst = append(dst, s[i:i+n]...)
		}
		i += n
	}
	return append(dst, '"')
}

// AppendStringStrict appends s as encoding/json/v2's Marshal writes a string:
// only what a string cannot hold unescaped is escaped, and a string that is not
// valid UTF-8 is an error (false).
func AppendStringStrict(dst []byte, s string) ([]byte, bool) {
	if !utf8.ValidString(s) {
		return dst, false
	}
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			dst = append(dst, '\\', c)
		case c < ' ':
			dst = appendControl(dst, c)
		default:
			dst = append(dst, c)
		}
	}
	return append(dst, '"'), true
}

// AppendQuote appends s as encoding/json/jsontext's AppendQuote does: only the
// quote, the backslash and the control characters are escaped, and each byte of
// invalid UTF-8 is written as U+FFFD (AppendQuote reports an error for it, which
// the callers of MyGo ignore).
func AppendQuote(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			dst = append(dst, '\\', c)
		case c < ' ':
			dst = appendControl(dst, c)
		case c < utf8.RuneSelf:
			dst = append(dst, c)
		default:
			r, n := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && n == 1 {
				dst = append(dst, "�"...)
			} else {
				dst = append(dst, s[i:i+n]...)
			}
			i += n
			continue
		}
		i++
	}
	return append(dst, '"')
}

// appendControl appends the escape of a byte below a space.
func appendControl(dst []byte, c byte) []byte {
	switch c {
	case '\b':
		return append(dst, '\\', 'b')
	case '\f':
		return append(dst, '\\', 'f')
	case '\n':
		return append(dst, '\\', 'n')
	case '\r':
		return append(dst, '\\', 'r')
	case '\t':
		return append(dst, '\\', 't')
	}
	return append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
}

// AppendFloat appends f as encoding/json writes a float64: false for a NaN or an
// infinity, which JSON has no number for.
func AppendFloat(dst []byte, f float64) ([]byte, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return dst, false
	}
	// Exponent form for very small and very large numbers, as ES6 does, with the
	// e-09 of strconv cleaned up to e-9.
	format := byte('f')
	if abs := math.Abs(f); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	dst = strconv.AppendFloat(dst, f, format, -1, 64)
	if format == 'e' {
		if n := len(dst); n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst, true
}
