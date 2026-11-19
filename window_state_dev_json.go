//go:build !js

package mygo

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/jsonlite"
)

// mygo dev's file of windows (window_state_dev.go), read and written without
// encoding/json, as window-state.json is (window_state_json.go): its
// reflection and tables would be linked into every app, which has no other
// use for them. It reads and writes what json.Unmarshal and json.Marshal do
// for a []devWindow (TestDevWindowsAgreeWithJSON).

// decodeDevWindows reads the windows of mygo dev's file.
func decodeDevWindows(data []byte) ([]devWindow, error) {
	r := jsonlite.NewReader(data)
	if null, err := r.Null(); err != nil || null {
		if err == nil {
			err = r.End()
		}
		return nil, err
	}
	list := []devWindow{}
	err := r.Array(func() error {
		var w devWindow
		defer func() { list = append(list, w) }()
		if null, err := r.Null(); err != nil || null {
			return err
		}
		return r.Object(func(key string) error {
			switch {
			case strings.EqualFold(key, "asked"):
				if null, err := r.Null(); err != nil || null {
					return err
				}
				return r.Object(func(field string) error { return w.Asked.decodeField(r, field) })
			case strings.EqualFold(key, "state"):
				if null, err := r.Null(); err != nil || null {
					return err
				}
				return r.Object(func(field string) error { return decodeDevState(r, &w.State, field) })
			}
			return r.Skip()
		})
	})
	if err == nil {
		err = r.End()
	}
	if err != nil {
		return nil, err
	}
	return list, nil
}

// decodeDevState reads a field of a window's state: its pixel bounds too,
// which the file keeps as json.Marshal writes them.
func decodeDevState(r *jsonlite.Reader, s *WindowState, name string) error {
	var n *int
	switch {
	case strings.EqualFold(name, "pixelX"):
		n = &s.PixelX
	case strings.EqualFold(name, "pixelY"):
		n = &s.PixelY
	case strings.EqualFold(name, "pixelWidth"):
		n = &s.PixelWidth
	case strings.EqualFold(name, "pixelHeight"):
		n = &s.PixelHeight
	default:
		return s.decodeField(r, name)
	}
	if null, err := r.Null(); err != nil || null {
		return err
	}
	var err error
	*n, err = r.Int()
	return err
}

// decodeField reads the value of a field of the frame a window's code asked
// for, its name matched regardless of case, as encoding/json matches it.
func (a *devWindowAsk) decodeField(r *jsonlite.Reader, name string) error {
	var n *int
	var b *bool
	switch {
	case strings.EqualFold(name, "x"):
		n = &a.X
	case strings.EqualFold(name, "y"):
		n = &a.Y
	case strings.EqualFold(name, "width"):
		n = &a.Width
	case strings.EqualFold(name, "height"):
		n = &a.Height
	case strings.EqualFold(name, "center"):
		b = &a.Center
	case strings.EqualFold(name, "contentSize"):
		b = &a.ContentSize
	case strings.EqualFold(name, "maximized"):
		b = &a.Maximized
	case strings.EqualFold(name, "fullScreen"):
		b = &a.FullScreen
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

// encodeDevWindows writes list as json.Marshal does.
func encodeDevWindows(list []devWindow) []byte {
	if list == nil {
		return []byte("null")
	}
	out := []byte{'['}
	for i, w := range list {
		if i > 0 {
			out = append(out, ',')
		}
		a := w.Asked
		out = append(out, `{"asked":{"x":`...)
		out = strconv.AppendInt(out, int64(a.X), 10)
		out = append(out, `,"y":`...)
		out = strconv.AppendInt(out, int64(a.Y), 10)
		out = append(out, `,"width":`...)
		out = strconv.AppendInt(out, int64(a.Width), 10)
		out = append(out, `,"height":`...)
		out = strconv.AppendInt(out, int64(a.Height), 10)
		for _, f := range [...]struct {
			on   bool
			name string
		}{{a.Center, "center"}, {a.ContentSize, "contentSize"}, {a.Maximized, "maximized"}, {a.FullScreen, "fullScreen"}} {
			if f.on {
				out = append(out, `,"`...)
				out = append(out, f.name...)
				out = append(out, `":true`...)
			}
		}
		s := w.State
		out = append(out, `},"state":{"x":`...)
		out = strconv.AppendInt(out, int64(s.X), 10)
		out = append(out, `,"y":`...)
		out = strconv.AppendInt(out, int64(s.Y), 10)
		out = append(out, `,"width":`...)
		out = strconv.AppendInt(out, int64(s.Width), 10)
		out = append(out, `,"height":`...)
		out = strconv.AppendInt(out, int64(s.Height), 10)
		if s.Maximized {
			out = append(out, `,"maximized":true`...)
		}
		if s.FullScreen {
			out = append(out, `,"fullScreen":true`...)
		}
		for _, f := range [...]struct {
			v    int
			name string
		}{{s.PixelX, "pixelX"}, {s.PixelY, "pixelY"}, {s.PixelWidth, "pixelWidth"}, {s.PixelHeight, "pixelHeight"}} {
			if f.v != 0 {
				out = append(out, `,"`...)
				out = append(out, f.name...)
				out = append(out, `":`...)
				out = strconv.AppendInt(out, int64(f.v), 10)
			}
		}
		out = append(out, "}}"...)
	}
	return append(out, ']')
}
