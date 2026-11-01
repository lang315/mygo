package mygo

import (
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/jsonlite"
)

// The window-state.json file, read and written without encoding/json, whose
// reflection and tables a native app has no other use for. It reads what
// json.Unmarshal reads into a map[string]WindowState, with the same results
// (TestWindowStatesAgreeWithJSON holds it to that), and writes what
// json.MarshalIndent(states, "", "  ") writes, byte for byte.

// decodeWindowStates reads the states of a window-state.json file.
func decodeWindowStates(data []byte) (map[string]WindowState, error) {
	r := jsonlite.NewReader(data)
	if null, err := r.Null(); err != nil {
		return nil, err
	} else if null {
		return nil, r.End()
	}
	states := map[string]WindowState{}
	err := r.Object(func(key string) error {
		var s WindowState
		if null, err := r.Null(); err != nil || null {
			states[key] = s
			return err
		}
		err := r.Object(func(field string) error { return s.decodeField(r, field) })
		states[key] = s
		return err
	})
	if err == nil {
		err = r.End()
	}
	if err != nil {
		return nil, err
	}
	return states, nil
}

// decodeField reads the value of a field of a state: its name matches
// regardless of case, as encoding/json matches it, and a null leaves the field
// as it was.
func (s *WindowState) decodeField(r *jsonlite.Reader, name string) error {
	var n *int
	var b *bool
	switch {
	case strings.EqualFold(name, "x"):
		n = &s.X
	case strings.EqualFold(name, "y"):
		n = &s.Y
	case strings.EqualFold(name, "width"):
		n = &s.Width
	case strings.EqualFold(name, "height"):
		n = &s.Height
	case strings.EqualFold(name, "maximized"):
		b = &s.Maximized
	case strings.EqualFold(name, "fullScreen"):
		b = &s.FullScreen
	default:
		return r.Skip()
	}
	if null, err := r.Null(); err != nil || null {
		return err
	}
	var err error
	if n != nil {
		*n, err = r.Int()
	} else {
		*b, err = r.Bool()
	}
	return err
}

// encodeWindowStates writes states as json.MarshalIndent(states, "", "  ") does.
func encodeWindowStates(states map[string]WindowState) []byte {
	if states == nil {
		return []byte("null")
	}
	if len(states) == 0 {
		return []byte("{}")
	}
	keys := make([]string, 0, len(states))
	for k := range states {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []byte("{")
	for i, k := range keys {
		if i > 0 {
			out = append(out, ',')
		}
		s := states[k]
		out = append(out, "\n  "...)
		out = jsonlite.AppendString(out, k)
		out = append(out, ": {\n    \"x\": "...)
		out = strconv.AppendInt(out, int64(s.X), 10)
		out = append(out, ",\n    \"y\": "...)
		out = strconv.AppendInt(out, int64(s.Y), 10)
		out = append(out, ",\n    \"width\": "...)
		out = strconv.AppendInt(out, int64(s.Width), 10)
		out = append(out, ",\n    \"height\": "...)
		out = strconv.AppendInt(out, int64(s.Height), 10)
		if s.Maximized {
			out = append(out, ",\n    \"maximized\": true"...)
		}
		if s.FullScreen {
			out = append(out, ",\n    \"fullScreen\": true"...)
		}
		out = append(out, "\n  }"...)
	}
	return append(out, "\n}"...)
}
