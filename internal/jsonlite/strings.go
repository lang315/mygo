package jsonlite

import "errors"

// DecodeStrings reads a JSON array of strings, the file of the recent
// documents: what encoding/json decodes into a []string, or an error (and no
// strings) where it reports one. A null is no strings.
func DecodeStrings(b []byte) ([]string, error) {
	r := NewReader(b)
	if null, err := r.Null(); err != nil {
		return nil, err
	} else if null {
		return nil, r.End()
	}
	var out []string
	err := r.Array(func() error {
		// An element that is null is an empty string, as encoding/json decodes it.
		if null, err := r.Null(); err != nil || null {
			out = append(out, "")
			return err
		}
		s, err := r.String()
		out = append(out, s)
		return err
	})
	if err == nil {
		err = r.End()
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

var errInvalidUTF8 = errors.New("jsonlite: string is not valid UTF-8")

// EncodeStrings writes strings as a JSON array, as encoding/json/v2's Marshal
// does: no white space, [] for none, and an error for a string that is not valid
// UTF-8.
func EncodeStrings(strs []string) ([]byte, error) {
	out := []byte{'['}
	for i, s := range strs {
		if i > 0 {
			out = append(out, ',')
		}
		var ok bool
		if out, ok = AppendStringStrict(out, s); !ok {
			return nil, errInvalidUTF8
		}
	}
	return append(out, ']'), nil
}
