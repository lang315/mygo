// Package jsonlite reads and writes the few small JSON files MyGo keeps for every
// app (the states of windows, the recent documents, the messages of a catalog)
// without linking encoding/json, which brings reflection, the unicode tables and
// 500 KB of code into an app that has no use for them. It reads what
// encoding/json reads, with the same results, and writes what it writes, byte for
// byte: files an older version wrote with encoding/json are read, and what it
// writes is read by encoding/json. TestDifferential* and FuzzReader hold it to
// that against encoding/json.
//
// A Reader walks a document without building it: the caller names what it expects
// (Object, Array, String, Int, Bool) and skips the rest (Skip). Everything a
// document holds is checked, as encoding/json checks it before it decodes: an
// error anywhere means the document is not JSON, and the caller drops what it
// had read.
package jsonlite

import (
	"errors"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// maxDepth is how deep a document nests, as encoding/json allows.
const maxDepth = 10000

// errors, with the offset of the byte that broke the document.
var (
	errSyntax = errors.New("jsonlite: invalid JSON")
	errType   = errors.New("jsonlite: value of another type")
	errDepth  = errors.New("jsonlite: exceeded max depth")
)

// A Reader reads one JSON document.
//
// By default it reads as encoding/json (v1) does: invalid UTF-8 and lone
// surrogates become U+FFFD, and a name that occurs twice in an object is read
// twice. With Strict set it reads as encoding/json/v2 does: invalid UTF-8 and lone
// surrogates are errors, and so is a duplicate name in an object. (v2 matches names
// exactly, not without regard to case; a caller that matches names compares them
// exactly.)
type Reader struct {
	// Strict makes the Reader refuse what encoding/json/v2 refuses.
	Strict bool

	b     []byte
	i     int
	depth int
}

// NewReader returns a Reader of the document b.
func NewReader(b []byte) *Reader { return &Reader{b: b} }

func (r *Reader) space() {
	for r.i < len(r.b) {
		switch r.b[r.i] {
		case ' ', '\t', '\n', '\r':
			r.i++
		default:
			return
		}
	}
}

// Peek returns the first byte of the next value, 0 at the end of the document.
func (r *Reader) Peek() byte {
	r.space()
	if r.i >= len(r.b) {
		return 0
	}
	return r.b[r.i]
}

// End reports an error unless only white space follows the value read.
func (r *Reader) End() error {
	r.space()
	if r.i != len(r.b) {
		return errSyntax
	}
	return nil
}

// literal consumes the keyword word.
func (r *Reader) literal(word string) error {
	if len(r.b)-r.i < len(word) || string(r.b[r.i:r.i+len(word)]) != word {
		return errSyntax
	}
	r.i += len(word)
	return nil
}

// Null consumes a null, reporting whether there was one.
func (r *Reader) Null() (bool, error) {
	if r.Peek() != 'n' {
		return false, nil
	}
	return true, r.literal("null")
}

// Bool reads true or false.
func (r *Reader) Bool() (bool, error) {
	switch r.Peek() {
	case 't':
		return true, r.literal("true")
	case 'f':
		return false, r.literal("false")
	}
	return false, r.typeError()
}

// typeError is the error of a value that is not the one asked for: a syntax
// error if it is not a value at all, so that a broken document is reported as one.
func (r *Reader) typeError() error {
	if err := r.Skip(); err != nil {
		return err
	}
	return errType
}

// number consumes a number and returns its text.
func (r *Reader) number() (string, error) {
	start := r.i
	if r.i < len(r.b) && r.b[r.i] == '-' {
		r.i++
	}
	if r.i >= len(r.b) {
		return "", errSyntax
	}
	switch c := r.b[r.i]; {
	case c == '0':
		r.i++
	case c >= '1' && c <= '9':
		for r.i < len(r.b) && r.b[r.i] >= '0' && r.b[r.i] <= '9' {
			r.i++
		}
	default:
		return "", errSyntax
	}
	if r.i < len(r.b) && r.b[r.i] == '.' {
		r.i++
		if !r.digits() {
			return "", errSyntax
		}
	}
	if r.i < len(r.b) && (r.b[r.i] == 'e' || r.b[r.i] == 'E') {
		r.i++
		if r.i < len(r.b) && (r.b[r.i] == '+' || r.b[r.i] == '-') {
			r.i++
		}
		if !r.digits() {
			return "", errSyntax
		}
	}
	return string(r.b[start:r.i]), nil
}

func (r *Reader) digits() bool {
	n := r.i
	for r.i < len(r.b) && r.b[r.i] >= '0' && r.b[r.i] <= '9' {
		r.i++
	}
	return r.i > n
}

// Int reads a number written as an integer that an int holds: encoding/json
// does not decode 1.0 or 1e2 into an int either.
func (r *Reader) Int() (int, error) {
	n, err := r.int(strconv.IntSize)
	return int(n), err
}

// Float64 reads a number that a float64 holds: one beyond its range is an error,
// as it is for encoding/json.
func (r *Reader) Float64() (float64, error) {
	c := r.Peek()
	if c != '-' && (c < '0' || c > '9') {
		return 0, r.typeError()
	}
	text, err := r.number()
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, errType
	}
	return f, nil
}

// Int64 is Int for an int64.
func (r *Reader) Int64() (int64, error) { return r.int(64) }

func (r *Reader) int(bits int) (int64, error) {
	c := r.Peek()
	if c != '-' && (c < '0' || c > '9') {
		return 0, r.typeError()
	}
	text, err := r.number()
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(text, 10, bits)
	if err != nil {
		return 0, errType
	}
	return n, nil
}

// Value reads a value of any kind, checks it, and returns its bytes, which are
// those of the document: the caller must not change them.
func (r *Reader) Value() ([]byte, error) {
	r.space()
	start := r.i
	if err := r.Skip(); err != nil {
		return nil, err
	}
	return r.b[start:r.i], nil
}

// String reads a string. Invalid UTF-8 and lone surrogates become U+FFFD, as
// encoding/json decodes them.
func (r *Reader) String() (string, error) {
	if r.Peek() != '"' {
		return "", r.typeError()
	}
	r.i++
	start := r.i
	// The common case: no escape, valid UTF-8.
	for r.i < len(r.b) {
		c := r.b[r.i]
		if c == '"' {
			s := r.b[start:r.i]
			if utf8.Valid(s) {
				r.i++
				return string(s), nil
			}
			if r.Strict {
				return "", errSyntax
			}
			break
		}
		if c == '\\' {
			break
		}
		if c < ' ' {
			return "", errSyntax
		}
		r.i++
	}
	buf := append([]byte(nil), r.b[start:r.i]...)
	for r.i < len(r.b) {
		c := r.b[r.i]
		switch {
		case c == '"':
			r.i++
			if r.Strict && !utf8.Valid(buf) {
				return "", errSyntax
			}
			return string(coerceUTF8(buf)), nil
		case c == '\\':
			r.i++
			if r.i >= len(r.b) {
				return "", errSyntax
			}
			e := r.b[r.i]
			r.i++
			switch e {
			case '"', '\\', '/':
				buf = append(buf, e)
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'u':
				rr, ok := r.hex4()
				if !ok {
					return "", errSyntax
				}
				if utf16.IsSurrogate(rr) {
					// A pair if a low surrogate escape follows a high one; otherwise the
					// lone surrogate is U+FFFD and the next escape is read on its own.
					save := r.i
					if r.i+1 < len(r.b) && r.b[r.i] == '\\' && r.b[r.i+1] == 'u' {
						r.i += 2
						if rr2, ok := r.hex4(); ok {
							if dec := utf16.DecodeRune(rr, rr2); dec != utf8.RuneError {
								buf = utf8.AppendRune(buf, dec)
								continue
							}
						}
					}
					if r.Strict {
						return "", errSyntax
					}
					r.i = save
					rr = utf8.RuneError
				}
				buf = utf8.AppendRune(buf, rr)
			default:
				return "", errSyntax
			}
		case c < ' ':
			return "", errSyntax
		default:
			buf = append(buf, c)
			r.i++
		}
	}
	return "", errSyntax
}

// hex4 reads the four hex digits of a \u escape.
func (r *Reader) hex4() (rune, bool) {
	if len(r.b)-r.i < 4 {
		return 0, false
	}
	var v rune
	for _, c := range r.b[r.i : r.i+4] {
		switch {
		case c >= '0' && c <= '9':
			c -= '0'
		case c >= 'a' && c <= 'f':
			c = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			c = c - 'A' + 10
		default:
			return 0, false
		}
		v = v<<4 | rune(c)
	}
	r.i += 4
	return v, true
}

// coerceUTF8 replaces each byte of invalid UTF-8 with U+FFFD.
func coerceUTF8(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	out := make([]byte, 0, len(b)+8)
	for len(b) > 0 {
		c, n := utf8.DecodeRune(b)
		out = utf8.AppendRune(out, c)
		b = b[n:]
	}
	return out
}

// Skip reads a value of any kind and checks it.
func (r *Reader) Skip() error {
	switch c := r.Peek(); {
	case c == '{':
		return r.Object(func(string) error { return r.Skip() })
	case c == '[':
		return r.Array(func() error { return r.Skip() })
	case c == '"':
		_, err := r.String()
		return err
	case c == 't':
		return r.literal("true")
	case c == 'f':
		return r.literal("false")
	case c == 'n':
		return r.literal("null")
	case c == '-' || c >= '0' && c <= '9':
		_, err := r.number()
		return err
	}
	return errSyntax
}

func (r *Reader) enter() error {
	r.depth++
	if r.depth > maxDepth {
		return errDepth
	}
	return nil
}

// Object reads an object, calling fn with each key. fn reads the value of the
// key (or Skips it) before it returns.
func (r *Reader) Object(fn func(key string) error) error {
	if r.Peek() != '{' {
		return r.typeError()
	}
	r.i++
	if err := r.enter(); err != nil {
		return err
	}
	defer func() { r.depth-- }()
	if r.Peek() == '}' {
		r.i++
		return nil
	}
	var seen []string
	for {
		if r.Peek() != '"' {
			return errSyntax
		}
		key, err := r.String()
		if err != nil {
			return err
		}
		if r.Strict {
			// shortcut: a linear scan, quadratic in the members of one object; the
			// objects of the IPC have a handful, upgrade to a map for more.
			for _, k := range seen {
				if k == key {
					return errSyntax
				}
			}
			seen = append(seen, key)
		}
		if r.Peek() != ':' {
			return errSyntax
		}
		r.i++
		if err := fn(key); err != nil {
			return err
		}
		switch r.Peek() {
		case ',':
			r.i++
		case '}':
			r.i++
			return nil
		default:
			return errSyntax
		}
	}
}

// Array reads an array, calling fn for each element, which fn reads before it
// returns.
func (r *Reader) Array(fn func() error) error {
	if r.Peek() != '[' {
		return r.typeError()
	}
	r.i++
	if err := r.enter(); err != nil {
		return err
	}
	defer func() { r.depth-- }()
	if r.Peek() == ']' {
		r.i++
		return nil
	}
	for {
		if err := fn(); err != nil {
			return err
		}
		switch r.Peek() {
		case ',':
			r.i++
		case ']':
			r.i++
			return nil
		default:
			return errSyntax
		}
	}
}
