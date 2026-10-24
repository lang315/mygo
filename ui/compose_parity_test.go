package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// An input without OnEdit must edit, undo and redo exactly as it did before
// OnEdit existed, whatever the platform's order of composition calls. The
// table is what the code of the main branch before OnEdit did: after the
// scenario, after one undo, after two, and after a redo: the text, the
// caret and anchor, the steps of undo and of redo.
type parityState struct {
	text                string
	caret, anchor, undo int
	redo                int
}

var parityWant = map[string][4]parityState{
	"tester-nosel-commit":     {{"abcXY", 5, 5, 1, 0}, {"abc", 3, 3, 0, 1}, {"abc", 3, 3, 0, 1}, {"abcXY", 5, 5, 1, 0}},
	"tester-sel-commit":       {{"1X45", 2, 2, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"1X45", 2, 2, 1, 0}},
	"tester-sel-cancel":       {{"145", 1, 1, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"145", 1, 1, 1, 0}},
	"tester-nosel-cancel":     {{"12345", 2, 2, 1, 0}, {"12345", 2, 2, 0, 1}, {"12345", 2, 2, 0, 1}, {"12345", 2, 2, 1, 0}},
	"client-nosel-commit":     {{"abcXY", 5, 5, 1, 0}, {"abc", 3, 3, 0, 1}, {"abc", 3, 3, 0, 1}, {"abcXY", 5, 5, 1, 0}},
	"client-sel-commit":       {{"1X45", 2, 2, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"1X45", 2, 2, 1, 0}},
	"client-sel-cancel":       {{"145", 1, 1, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"145", 1, 1, 1, 0}},
	"client-nosel-cancel":     {{"12345", 2, 2, 1, 0}, {"12345", 2, 2, 0, 1}, {"12345", 2, 2, 0, 1}, {"12345", 2, 2, 1, 0}},
	"gtk-sel":                 {{"1一45", 2, 2, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"1一45", 2, 2, 1, 0}},
	"gtk-nosel":               {{"1一2345", 2, 2, 1, 0}, {"12345", 1, 1, 0, 1}, {"12345", 1, 1, 0, 1}, {"1一2345", 2, 2, 1, 0}},
	"gtk-end-nocommit-reset":  {{"1yiz45", 4, 4, 2, 0}, {"1yi45", 3, 3, 1, 1}, {"12345", 3, 1, 0, 2}, {"1yi45", 3, 3, 1, 1}},
	"win-result-then-end":     {{"1一45", 2, 2, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"1一45", 2, 2, 1, 0}},
	"appkit-unmark":           {{"1yi45", 3, 3, 1, 0}, {"12345", 3, 1, 0, 1}, {"12345", 3, 1, 0, 1}, {"1yi45", 3, 3, 1, 0}},
	"appkit-unmark-insert":    {{"abđưc", 5, 5, 2, 0}, {"abđư", 4, 4, 1, 1}, {"ab", 2, 2, 0, 2}, {"abđư", 4, 4, 1, 1}},
	"telex-client":            {{"Nguyễn Văn Đức", 14, 14, 5, 0}, {"Nguyễn Văn ", 11, 11, 4, 1}, {"Nguyễn Văn", 10, 10, 3, 2}, {"Nguyễn Văn ", 11, 11, 4, 1}},
	"telex-tester":            {{"Nguyễn Văn Đức", 14, 14, 5, 0}, {"Nguyễn Văn ", 11, 11, 4, 1}, {"Nguyễn Văn", 10, 10, 3, 2}, {"Nguyễn Văn ", 11, 11, 4, 1}},
	"two-comp-noframe-client": {{"aXYb", 3, 3, 2, 0}, {"aXb", 2, 2, 1, 1}, {"ab", 1, 1, 0, 2}, {"aXb", 2, 2, 1, 1}},
	"two-comp-noframe-gtk":    {{"XY", 2, 2, 2, 0}, {"X", 1, 1, 1, 1}, {"ab", 2, 0, 0, 2}, {"X", 1, 1, 1, 1}},
	"redo-then-cancel":        {{"a", 1, 1, 2, 0}, {"a", 1, 1, 1, 1}, {"", 0, 0, 0, 2}, {"a", 1, 1, 1, 1}},
	"redo-then-backspace0":    {{"a", 1, 1, 1, 0}, {"", 0, 0, 0, 1}, {"", 0, 0, 0, 1}, {"a", 1, 1, 1, 0}},
	"unikey-burst":            {{"đá việt", 7, 7, 4, 0}, {"đá vi", 5, 5, 3, 1}, {"đá v", 4, 4, 2, 2}, {"đá vi", 5, 5, 3, 1}},
	"unikey-burst-client":     {{"đá", 2, 2, 1, 0}, {"", 0, 0, 0, 1}, {"", 0, 0, 0, 1}, {"đá", 2, 2, 1, 0}},
}

type parityCase struct {
	name string
	init string
	sel  [2]int
	run  func(tt *Tester, c platform.TextInputClient)
}

func telexClient(c platform.TextInputClient) {
	for _, syl := range [][]string{{"N", "Ng", "Ngu", "Nguy", "Nguye", "Nguyê", "Nguyên", "Nguyễn"}, {" "}, {"V", "Va", "Vă", "Văn"}, {" "}, {"Đ", "Đu", "Đư", "Đưc", "Đức"}} {
		if len(syl) == 1 && syl[0] == " " {
			c.ReplaceText(nil, " ")
			continue
		}
		for _, p := range syl {
			c.SetMarkedText(nil, p, TextInputRange{Start: len([]rune(p)), End: len([]rune(p))})
		}
		c.UnmarkText()
	}
}

var parityCases = []parityCase{
	{"tester-nosel-commit", "abc", [2]int{3, 3}, func(tt *Tester, c platform.TextInputClient) {
		tt.Compose("x", 1)
		tt.Compose("xy", 2)
		tt.Type("XY")
	}},
	{"tester-sel-commit", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		tt.Compose("x", 1)
		tt.Type("X")
	}},
	{"tester-sel-cancel", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		tt.Compose("x", 1)
		tt.Compose("", 0)
	}},
	{"tester-nosel-cancel", "12345", [2]int{2, 2}, func(tt *Tester, c platform.TextInputClient) {
		tt.Compose("x", 1)
		tt.Compose("", 0)
	}},
	{"client-nosel-commit", "abc", [2]int{3, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		tt.Frame()
		c.SetMarkedText(nil, "xy", TextInputRange{Start: 2, End: 2})
		c.ReplaceText(nil, "XY")
	}},
	{"client-sel-commit", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		tt.Frame()
		c.ReplaceText(nil, "X")
	}},
	{"client-sel-cancel", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		tt.Frame()
		c.SetMarkedText(nil, "", TextInputRange{})
	}},
	{"client-nosel-cancel", "12345", [2]int{2, 2}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		tt.Frame()
		c.SetMarkedText(nil, "", TextInputRange{})
	}},
	{"gtk-sel", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
		tt.Frame()
		var o platform.InputComposition
		o.End(c)
		o.Replace(c, nil, "一")
	}},
	{"gtk-nosel", "12345", [2]int{1, 1}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
		tt.Frame()
		var o platform.InputComposition
		o.End(c)
		o.Replace(c, nil, "一")
	}},
	{"gtk-end-nocommit-reset", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
		tt.Frame()
		var o platform.InputComposition
		o.End(c)
		tt.Frame()
		o.Reset()
		tt.Type("z")
	}},
	{"win-result-then-end", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
		tt.Frame()
		var o platform.InputComposition
		o.Replace(c, nil, "一")
		o.End(c)
	}},
	{"appkit-unmark", "12345", [2]int{1, 3}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "yi", TextInputRange{Start: 2, End: 2})
		tt.Frame()
		c.UnmarkText()
	}},
	{"appkit-unmark-insert", "ab", [2]int{2, 2}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "đư", TextInputRange{Start: 2, End: 2})
		c.UnmarkText()
		c.ReplaceText(nil, "c")
	}},
	{"telex-client", "", [2]int{0, 0}, func(tt *Tester, c platform.TextInputClient) { telexClient(c) }},
	{"telex-tester", "", [2]int{0, 0}, func(tt *Tester, c platform.TextInputClient) {
		for _, w := range []string{"Nguyễn", " ", "Văn", " ", "Đức"} {
			if w == " " {
				tt.Type(" ")
				continue
			}
			r := []rune(w)
			for i := 1; i <= len(r); i++ {
				tt.Compose(string(r[:i]), i)
			}
			tt.Type(w)
		}
	}},
	{"two-comp-noframe-client", "ab", [2]int{1, 1}, func(tt *Tester, c platform.TextInputClient) {
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		c.ReplaceText(nil, "X")
		c.SetMarkedText(nil, "y", TextInputRange{Start: 1, End: 1})
		c.ReplaceText(nil, "Y")
	}},
	{"two-comp-noframe-gtk", "ab", [2]int{0, 2}, func(tt *Tester, c platform.TextInputClient) {
		var o platform.InputComposition
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		o.End(c)
		o.Replace(c, nil, "X")
		o.Reset()
		c.SetMarkedText(nil, "y", TextInputRange{Start: 1, End: 1})
		o.End(c)
		o.Replace(c, nil, "Y")
	}},
	{"redo-then-cancel", "", [2]int{0, 0}, func(tt *Tester, c platform.TextInputClient) {
		tt.Type("a")
		tt.Advance(2 * time.Second)
		tt.Type("b")
		tt.Key(Cmd, KeyZ)
		c.SetMarkedText(nil, "x", TextInputRange{Start: 1, End: 1})
		tt.Frame()
		c.SetMarkedText(nil, "", TextInputRange{})
		tt.Frame()
		tt.Key(Cmd|Shift, KeyZ)
	}},
	{"redo-then-backspace0", "", [2]int{0, 0}, func(tt *Tester, c platform.TextInputClient) {
		tt.Type("a")
		tt.Key(Cmd, KeyZ)
		tt.Key(0, KeyBackspace)
		tt.Key(Cmd|Shift, KeyZ)
	}},
	{"unikey-burst", "", [2]int{0, 0}, func(tt *Tester, c platform.TextInputClient) {
		// Unikey/EVKey: no composition; Backspace + the new character.
		tt.Type("d")
		tt.Key(0, KeyBackspace)
		tt.Type("đ")
		tt.Type("a")
		tt.Key(0, KeyBackspace)
		tt.Type("á")
		tt.Type(" v")
		tt.Type("i")
		tt.Type("e")
		tt.Key(0, KeyBackspace)
		tt.Type("ê")
		tt.Type("t")
		tt.Key(0, KeyBackspace)
		tt.Key(0, KeyBackspace)
		tt.Type("ệt")
	}},
	{"unikey-burst-client", "", [2]int{0, 0}, func(tt *Tester, c platform.TextInputClient) {
		c.ReplaceText(nil, "d")
		tt.Key(0, KeyBackspace)
		c.ReplaceText(nil, "đ")
		c.ReplaceText(nil, "a")
		tt.Key(0, KeyBackspace)
		c.ReplaceText(nil, "á")
	}},
}

func TestInputWithoutOnEditEditsUndoesAndRedoesAsBeforeOnEdit(t *testing.T) {
	for _, k := range parityCases {
		s := k.init
		tt := inputTester(&s, false)
		ed := tt.rt.states[tt.rt.focused].editor
		ed.anchor, ed.caret = k.sel[0], k.sel[1]
		tt.Frame()
		k.run(tt, tt.h.ime.Client)
		tt.Frame()
		var got [4]parityState
		snap := func() parityState {
			return parityState{s, ed.caret, ed.anchor, len(ed.undo), len(ed.redo)}
		}
		got[0] = snap()
		tt.Key(Cmd, KeyZ)
		got[1] = snap()
		tt.Key(Cmd, KeyZ)
		got[2] = snap()
		tt.Key(Cmd|Shift, KeyZ)
		got[3] = snap()
		tt.rt.close()
		want, ok := parityWant[k.name]
		if !ok {
			t.Errorf("%s: no expected states", k.name)
			continue
		}
		for i, g := range got {
			if g != want[i] {
				t.Errorf("%s step %d: %+v, want %+v", k.name, i, g, want[i])
			}
		}
	}
}
