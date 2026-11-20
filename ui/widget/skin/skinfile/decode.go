package skinfile

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/jsonlite"
	"github.com/egoist/mygo/ui"
)

var colorType = reflect.TypeFor[ui.Color]()

// fileError is an error of a file, naming its path and its line.
type fileError struct{ msg string }

func (e *fileError) Error() string { return e.msg }

// decoder reads the values of file from r, whose bytes start at the offset
// base of file.
type decoder struct {
	file []byte
	base int
	r    *jsonlite.Reader
}

// fail returns the error msg at path, at the reader's offset off.
func (d *decoder) fail(path string, off int, msg string) error {
	off = d.base + off
	line := 1 + bytes.Count(d.file[:off], []byte{'\n'})
	if path != "" {
		msg = path + ": " + msg
	}
	return &fileError{fmt.Sprintf("%s (line %d)", msg, line)}
}

// wrap returns err as an error at path: one of jsonlite's where the reader
// stopped, the decoder's own as it is.
func (d *decoder) wrap(path string, err error) error {
	if _, ok := err.(*fileError); ok || err == nil {
		return err
	}
	return d.fail(path, d.r.Offset(), strings.TrimPrefix(err.Error(), "jsonlite: "))
}

// read reads a value at path and returns its text and its offset.
func (d *decoder) read(path string) ([]byte, int, error) {
	d.r.Peek()
	off := d.r.Offset()
	v, err := d.r.Value()
	return v, off, d.wrap(path, err)
}

// got returns the error of the value v at off, which is not what is wanted.
func (d *decoder) got(path string, off int, what string, v []byte) error {
	return d.fail(path, off, "want "+what+", got "+cut(string(v)))
}

// want reads a value at path that is not what is wanted and returns the
// error naming it.
func (d *decoder) want(path, what string) error {
	v, off, err := d.read(path)
	if err != nil {
		return err
	}
	return d.got(path, off, what, v)
}

// object reads an object at path, calling fn with each key, its path and
// the offset after it. A key twice is an error naming it.
func (d *decoder) object(path string, fn func(key, path string, off int) error) error {
	if d.r.Peek() != '{' {
		return d.want(path, "an object")
	}
	seen := map[string]bool{}
	err := d.r.Object(func(key string) error {
		off := d.r.Offset()
		p := key
		if path != "" {
			p = path + "." + key
		}
		if seen[key] {
			return d.fail(p, off, "duplicate key")
		}
		seen[key] = true
		return d.wrap(p, fn(key, p, off))
	})
	return d.wrap(path, err)
}

// str reads a string at path.
func (d *decoder) str(path string) (string, error) {
	v, off, err := d.read(path)
	if err != nil {
		return "", err
	}
	if v[0] != '"' {
		return "", d.got(path, off, "a string", v)
	}
	return jsonlite.NewReader(v).String()
}

const wantColor = "a color #rrggbb or #rrggbbaa"

// color reads a color at path.
func (d *decoder) color(path string) (ui.Color, error) {
	v, off, err := d.read(path)
	if err != nil {
		return ui.Color{}, err
	}
	if s, err := jsonlite.NewReader(v).String(); err == nil {
		if c, ok := parseColor(s); ok {
			return c, nil
		}
	}
	return ui.Color{}, d.got(path, off, wantColor, v)
}

// parseColor reads "#rrggbb" or "#rrggbbaa", and nothing else.
func parseColor(s string) (ui.Color, bool) {
	if len(s) != 7 && len(s) != 9 || s[0] != '#' {
		return ui.Color{}, false
	}
	for _, c := range []byte(s[1:]) {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
			return ui.Color{}, false
		}
	}
	return ui.Hex(s), true
}

// value reads the value at path into v, a field of Tokens or a part of one.
func (d *decoder) value(v reflect.Value, path string) error {
	if v.Type() == colorType {
		c, err := d.color(path)
		if err == nil {
			v.Set(reflect.ValueOf(c))
		}
		return err
	}
	switch v.Kind() {
	case reflect.Struct:
		return d.object(path, func(key, p string, off int) error {
			f, ok := v.Type().FieldByName(key)
			if !ok || !f.IsExported() || len(f.Index) != 1 {
				return d.fail(p, off, "unknown key")
			}
			return d.value(v.Field(f.Index[0]), p)
		})
	case reflect.Array:
		if d.r.Peek() != '[' {
			return d.want(path, "an array of "+strconv.Itoa(v.Len()))
		}
		off := d.r.Offset()
		n := 0
		err := d.r.Array(func() error {
			n++
			if n > v.Len() {
				return d.r.Skip()
			}
			return d.value(v.Index(n-1), fmt.Sprintf("%s[%d]", path, n-1))
		})
		if err != nil {
			return d.wrap(path, err)
		}
		if n != v.Len() {
			return d.fail(path, off, fmt.Sprintf("want %d colors, got %d", v.Len(), n))
		}
		return nil
	}
	val, off, err := d.read(path)
	if err != nil {
		return err
	}
	text := string(val)
	switch v.Kind() {
	case reflect.Float32:
		if !isNumber(val) {
			return d.got(path, off, "a number", val)
		}
		f, err := strconv.ParseFloat(text, 32)
		if err != nil {
			return d.got(path, off, "a number a float32 holds", val)
		}
		v.SetFloat(f)
	case reflect.Int:
		n, err := strconv.ParseInt(text, 10, 0)
		if err != nil {
			return d.got(path, off, "an integer", val)
		}
		v.SetInt(n)
	case reflect.Bool:
		if text != "true" && text != "false" {
			return d.got(path, off, "true or false", val)
		}
		v.SetBool(text == "true")
	case reflect.String:
		if val[0] != '"' {
			return d.got(path, off, "a string", val)
		}
		s, err := jsonlite.NewReader(val).String()
		if err != nil {
			return d.wrap(path, err)
		}
		v.SetString(s)
	default:
		return d.fail(path, off, "a field of kind "+v.Kind().String())
	}
	return nil
}

// isNumber reports whether the JSON value v is a number.
func isNumber(v []byte) bool { return v[0] == '-' || '0' <= v[0] && v[0] <= '9' }
