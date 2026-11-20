package jsonlite

import "testing"

// TestOffset: after an error, Offset is the byte the reader stopped at; after
// a whole document, its length.
func TestOffset(t *testing.T) {
	doc := []byte(`{"a": tru}`)
	r := NewReader(doc)
	err := r.Object(func(string) error {
		_, err := r.Bool()
		return err
	})
	if err == nil {
		t.Fatal("tru read as a bool")
	}
	if got := r.Offset(); got != 6 {
		t.Errorf("Offset after the bad literal = %d, want 6", got)
	}

	// A value of another type is skipped: Offset is past it.
	doc = []byte(`{"a": 12}`)
	r = NewReader(doc)
	if err := r.Object(func(string) error { _, err := r.Bool(); return err }); err == nil {
		t.Fatal("12 read as a bool")
	}
	if got := r.Offset(); got != 8 {
		t.Errorf("Offset after a number read as a bool = %d, want 8", got)
	}

	doc = []byte(" {\"a\": [1, true]} \n")
	r = NewReader(doc)
	if got := r.Offset(); got != 0 {
		t.Errorf("Offset before reading = %d, want 0", got)
	}
	if err := r.Skip(); err != nil {
		t.Fatal(err)
	}
	if err := r.End(); err != nil {
		t.Fatal(err)
	}
	if got := r.Offset(); got != len(doc) {
		t.Errorf("Offset after End = %d, want %d", got, len(doc))
	}
}
