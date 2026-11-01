package mygo

import (
	"strconv"

	"github.com/egoist/mygo/internal/jsonlite"
)

// The payloads of MyGo's own events (a document's state, dropped files) are
// written by hand as encoding/json/v2 writes them for the IPC (the codec, with
// jsonOptions): windows send them to pages whether the app binds a service or
// not, and an app that binds nothing links no codec. A string that is not valid
// UTF-8 is not written, as v2 does not write it, and the event is not sent.
// TestEventPayloadsAreV2 holds them to the same bytes.

func appendStringField(b []byte, key, s string) ([]byte, bool) {
	b = append(b, key...)
	return jsonlite.AppendStringJS(b, s)
}

func appendBoolField(b []byte, key string, v bool) []byte {
	return strconv.AppendBool(append(b, key...), v)
}

func (s DocumentState) appendJSON(b []byte) ([]byte, bool) {
	var ok bool
	if b, ok = appendStringField(b, `{"name":`, s.Name); !ok {
		return nil, false
	}
	if b, ok = appendStringField(b, `,"path":`, s.Path); !ok {
		return nil, false
	}
	b = appendBoolField(b, `,"edited":`, s.Edited)
	b = appendBoolField(b, `,"readOnly":`, s.ReadOnly)
	b = appendBoolField(b, `,"conflict":`, s.Conflict)
	b = appendBoolField(b, `,"deleted":`, s.Deleted)
	return append(b, '}'), true
}

func (e FileDropEvent) appendJSON(b []byte) ([]byte, bool) {
	b = append(b, `{"paths":`...)
	if e.Paths == nil {
		b = append(b, "[]"...)
	} else {
		b = append(b, '[')
		for i, p := range e.Paths {
			if i > 0 {
				b = append(b, ',')
			}
			var ok bool
			if b, ok = jsonlite.AppendStringJS(b, p); !ok {
				return nil, false
			}
		}
		b = append(b, ']')
	}
	b = append(b, `,"x":`...)
	b = strconv.AppendInt(b, int64(e.X), 10)
	b = append(b, `,"y":`...)
	b = strconv.AppendInt(b, int64(e.Y), 10)
	return append(b, '}'), true
}
