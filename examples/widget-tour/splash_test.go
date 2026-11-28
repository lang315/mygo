package main

import "testing"

// The splash main shows while the tour starts: its image, title and first
// status, in the default size.
func TestTourSplash(t *testing.T) {
	s := tourSplash()
	if s.Image == nil || s.Title != "Widget Tour" || s.Status != "Starting" || s.Width != 0 || s.Height != 0 {
		t.Fatalf("splash %+v, want the search icon, \"Widget Tour\", \"Starting\" and the default size", s)
	}
}
