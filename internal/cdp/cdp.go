// Package cdp writes the parameters of the few DevTools protocol methods the
// Windows backend calls, and reads their replies, without linking encoding/json:
// its reflection and its tables were 600 KB of every app on Windows, native UI
// ones included, which have no use for them. What it writes decodes as what
// json.Marshal wrote, and it reads what json.Unmarshal read, names matched
// without regard to case as it does.
// TestReadersMatchEncodingJSON and FuzzReaders hold it to that.
package cdp

import (
	"errors"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/jsonlite"
)

// InsertText is the parameters of Input.insertText.
func InsertText(text string) string {
	return `{"text":` + quote(text) + `}`
}

// Evaluate is the parameters of Runtime.evaluate for an expression; userGesture
// runs it as a user's gesture (requestPictureInPicture() needs one).
func Evaluate(expression string, userGesture bool) string {
	return `{"expression":` + quote(expression) + `,"userGesture":` + strconv.FormatBool(userGesture) + `}`
}

// EvaluateAsync is the parameters of Runtime.evaluate for an expression that
// returns a promise, whose value it awaits and returns.
func EvaluateAsync(expression string) string {
	return `{"expression":` + quote(expression) + `,"awaitPromise":true,"returnByValue":true}`
}

// PrintToPDF is the parameters of Page.printToPDF. A size that is not a number
// is an error.
func PrintToPDF(landscape, background bool, width, height, top, right, bottom, left float64) (string, error) {
	b := append([]byte(`{"landscape":`), strconv.FormatBool(landscape)...)
	b = append(b, `,"printBackground":`...)
	b = strconv.AppendBool(b, background)
	for _, n := range []struct {
		name string
		v    float64
	}{{"paperWidth", width}, {"paperHeight", height}, {"marginTop", top}, {"marginRight", right}, {"marginBottom", bottom}, {"marginLeft", left}} {
		b = append(b, `,"`+n.name+`":`...)
		var ok bool
		if b, ok = jsonlite.AppendFloat(b, n.v); !ok {
			return "", errors.New("mygo: printing to PDF: " + n.name + " is not a number")
		}
	}
	return string(append(b, `,"preferCSSPageSize":false}`...)), nil
}

func quote(s string) string { return string(jsonlite.AppendString(nil, s)) }

// Value reads the reply of Runtime.evaluate: the string it returned (empty for any
// other value), or the exception it raised, as an error whose message is the
// exception's description.
func Value(reply string) (string, error) {
	var value []byte
	var text, description string
	var raised bool
	r := jsonlite.NewReader([]byte(reply))
	err := object(r, func(key string) error {
		switch {
		case fold(key, "result"):
			return object(r, func(key string) error {
				if !fold(key, "value") {
					return r.Skip()
				}
				v, err := r.Value()
				value = v
				return err
			})
		case fold(key, "exceptionDetails"):
			if null, err := r.Null(); null || err != nil {
				raised = false
				return err
			}
			raised = true
			return object(r, func(key string) error {
				switch {
				case fold(key, "text"):
					return str(r, &text)
				case fold(key, "exception"):
					if null, err := r.Null(); null || err != nil {
						description = ""
						return err
					}
					return object(r, func(key string) error {
						if fold(key, "description") {
							return str(r, &description)
						}
						return r.Skip()
					})
				}
				return r.Skip()
			})
		}
		return r.Skip()
	})
	if err == nil {
		err = r.End()
	}
	if err != nil {
		return "", err
	}
	if raised {
		if description != "" {
			text = description
		}
		return "", errors.New(text)
	}
	if value == nil {
		return "", nil
	}
	s, _ := jsonlite.NewReader(value).String() // a value that is no string is ""
	return s, nil
}

// Data reads the data of the reply of Page.printToPDF.
func Data(reply string) (string, error) {
	var data string
	r := jsonlite.NewReader([]byte(reply))
	err := object(r, func(key string) error {
		if fold(key, "data") {
			return str(r, &data)
		}
		return r.Skip()
	})
	if err == nil {
		err = r.End()
	}
	if err != nil {
		return "", err
	}
	return data, nil
}

// Title reads a title: the reply of Runtime.evaluate of document.title, which is
// a string.
func Title(reply string) (string, error) {
	var title string
	r := jsonlite.NewReader([]byte(reply))
	err := str(r, &title)
	if err == nil {
		err = r.End()
	}
	if err != nil {
		return "", err
	}
	return title, nil
}

// fold reports whether key names field: encoding/json matches names without regard
// to case when there is no exact match.
func fold(key, field string) bool { return strings.EqualFold(key, field) }

// object reads an object, or a null, which encoding/json reads into a struct as
// nothing.
func object(r *jsonlite.Reader, fn func(key string) error) error {
	if null, err := r.Null(); null || err != nil {
		return err
	}
	return r.Object(fn)
}

// str reads a string into dst; a null leaves it as it is, as encoding/json does.
func str(r *jsonlite.Reader, dst *string) error {
	if null, err := r.Null(); null || err != nil {
		return err
	}
	s, err := r.String()
	if err == nil {
		*dst = s
	}
	return err
}

// Strings is a JavaScript array of strings, as json.Marshal writes a []string (null
// for nil).
func Strings(strs []string) string {
	if strs == nil {
		return "null"
	}
	b := []byte{'['}
	for i, s := range strs {
		if i > 0 {
			b = append(b, ',')
		}
		b = jsonlite.AppendString(b, s)
	}
	return string(append(b, ']'))
}

// JSString is s as a JavaScript string literal, as json.Marshal writes it (the
// characters that would end a script element or a line are escaped).
func JSString(s string) string { return quote(s) }
