package ui_test

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestOnPasteOnAField sees a paste into the input of a field around one, as
// SearchField and TokenField, from OnPaste called on the field.
func TestOnPasteOnAField(t *testing.T) {
	for _, kind := range []string{"SearchField", "TokenField"} {
		t.Run(kind, func(t *testing.T) {
			var query string
			var tags []string
			var seen []string
			tt := ui.NewTester(func(c *ui.Context) {
				var f ui.Element
				if kind == "SearchField" {
					f = ui.SearchField(c, &query)
				} else {
					f = ui.TokenField(c, &tags, nil)
				}
				f.Label("field").Width(300).OnPaste(func(s string) bool {
					seen = append(seen, s)
					return s == "taken"
				})
			}, 400, 100)
			if err := tt.Click("field"); err != nil {
				t.Fatal(err)
			}
			tt.SetClipboard("left")
			tt.Command("paste")
			tt.SetClipboard("taken")
			tt.Command("paste")
			if len(seen) != 2 || seen[0] != "left" || seen[1] != "taken" {
				t.Fatalf("OnPaste saw %q, want [left taken]", seen)
			}
			if kind == "SearchField" && query != "left" {
				t.Fatalf("the query is %q, want left: a paste left inserts, one taken does not", query)
			}
			if a, _ := tt.AccessByRole(ui.RoleTextField, ""); kind == "TokenField" && a.Value != "left" {
				t.Fatalf("the input holds %q, want left", a.Value)
			}
		})
	}
}

// TestPlaceholderColorOnATokenField colors the placeholder of a token
// field's input, called on the field.
func TestPlaceholderColorOnATokenField(t *testing.T) {
	frame := func(hint string, color *ui.Color) []byte {
		var tags []string
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(placeholderColorTheme())
			f := ui.TokenField(c, &tags, nil).Label("tags").Width(300).Placeholder(hint)
			if color != nil {
				f.PlaceholderColor(*color)
			}
		}, 400, 100)
		return bytes.Clone(tt.Image().Pix)
	}
	red, clear := ui.RGB(240, 40, 80), ui.Color{}
	plain := frame("Placeholder sample", nil)
	if bytes.Equal(plain, frame("", nil)) {
		t.Fatal("fixture shows no placeholder")
	}
	if bytes.Equal(plain, frame("Placeholder sample", &red)) {
		t.Fatal("PlaceholderColor on the field did not color its input's placeholder")
	}
	if !bytes.Equal(frame("Placeholder sample", &clear), frame("", nil)) {
		t.Fatal("a transparent PlaceholderColor on the field did not hide the placeholder")
	}
}
