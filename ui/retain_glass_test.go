package ui

import "testing"

type testGlass struct{}

func (testGlass) PaintMaterial(*Painter, Rect, [4]float32) {}
func (testGlass) NativeGlass() (NativeGlassSpec, bool)     { return NativeGlassSpec{}, true }

type testPlainMaterial struct{}

func (testPlainMaterial) PaintMaterial(*Painter, Rect, [4]float32) {}

// The system's glass of an element is placed as the frame commits it
// (noteNativeGlass), and the places are cleared in every frame: a kept row
// would lose its pane, so a row holding one is never kept.
func TestRetainableRefusesRowsWithNativeGlass(t *testing.T) {
	glass := &node{}
	glass.material = testGlass{}
	if retainable(glass) {
		t.Error("a row with a native glass material is retainable")
	}
	plain := &node{}
	plain.material = testPlainMaterial{}
	if !retainable(plain) {
		t.Error("a row with a material of our own drawing is not retainable")
	}
}
