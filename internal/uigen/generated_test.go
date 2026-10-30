package main

import (
	"bytes"
	"os"
	"testing"
)

// ui/api_gen.go is what the generator makes of the files of ui/ now: a change
// to a method or a doc comment that was not followed by `go generate ./ui` is
// found here, not by whoever generates next.
func TestAPIGenIsUpToDate(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	want, err := os.ReadFile("ui/api_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := generateAPI(); !bytes.Equal(got, want) {
		t.Errorf("ui/api_gen.go is out of date: run go generate ./ui (%d bytes generated, %d committed)", len(got), len(want))
	}
}
