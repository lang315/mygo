package ui

import "testing"

func TestTextInputMaxLength(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("name").MaxLength(5)
	}, 300, 100)
	tt.Click("name")

	tt.Type("abcdefgh") // typing past the limit is cut at it
	if text != "abcde" {
		t.Fatalf("typed past 5: %q", text)
	}

	tt.Key(Cmd, KeyA)
	tt.SetClipboard("héllo wörld") // runes, not bytes
	tt.Key(Cmd, KeyV)
	if text != "héllo" {
		t.Fatalf("pasted past 5: %q", text)
	}

	tt.Key(0, KeyHome)
	tt.Key(0, KeyRight)
	tt.Type("Z") // in the middle at the limit: the typed rune is refused
	if text != "héllo" {
		t.Fatalf("typed at the limit: %q", text)
	}

	tt.Key(0, KeyBackspace)
	tt.Type("ZZ") // one place free
	if text != "Zéllo" {
		t.Fatalf("typed with one free: %q", text)
	}
}

func TestTextInputNoMaxLengthIsNoLimit(t *testing.T) {
	text := ""
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Width(200).Label("name").MaxLength(0)
	}, 300, 100)
	tt.Click("name")
	tt.Type("abcdefghij")
	if text != "abcdefghij" {
		t.Fatalf("typed %q", text)
	}
}
