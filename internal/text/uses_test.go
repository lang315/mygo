package text

import "testing"

// Touch marks the glyphs a painting used, as drawing them again would: no
// Glyph call asks for them in the frame after.
func TestTouchKeepsTheGlyphsAPaintingUsed(t *testing.T) {
	s := Shared()
	t.Cleanup(s.MakeRoom)
	l := s.Layout(Params{Text: "keep me", Style: Style{Size: 21}})
	var reqs []GlyphRequest
	for i, g := range l.Lines[0].Glyphs {
		reqs = append(reqs, GlyphRequest{Font: g.Font, ID: g.ID, X: float32(i) * 11, Shade: Flat})
	}
	var u Uses
	s.Record(&u)
	s.Glyphs(reqs, 1, false, nil)
	s.StopRecord()
	if len(u.entries) != len(reqs) {
		t.Fatalf("recorded %d entries for %d glyphs", len(u.entries), len(reqs))
	}
	s.EndFrame()
	s.mu.Lock()
	frame := s.frame
	for i, e := range u.entries {
		if e.used == frame {
			t.Errorf("entry %d is used in frame %d before Touch", i, frame)
		}
	}
	s.mu.Unlock()
	s.Touch(&u)
	s.mu.Lock()
	for i, e := range u.entries {
		if e.used != frame {
			t.Errorf("entry %d last used in frame %d after Touch, the frame is %d", i, e.used, frame)
		}
	}
	s.mu.Unlock()
}

// Recording stops: a call after StopRecord adds nothing.
func TestStopRecord(t *testing.T) {
	s := Shared()
	l := s.Layout(Params{Text: "x", Style: Style{Size: 13}})
	g := l.Lines[0].Glyphs[0]
	var u Uses
	s.Record(&u)
	s.StopRecord()
	s.Glyph(g.Font, g.ID, 1, 0, Flat, false)
	if len(u.entries) != 0 {
		t.Fatalf("%d entries recorded after StopRecord", len(u.entries))
	}
}

// A mask and a glyph run are recorded as the glyphs are.
func TestRecordSeesMasksAndRuns(t *testing.T) {
	s := Shared()
	var u Uses
	s.Record(&u)
	draw := func() (int, int, []byte) { return 2, 2, []byte{1, 2, 3, 4} }
	s.Mask(0xfeed, draw) // transient, for this frame
	s.StopRecord()
	if len(u.entries) != 0 {
		t.Fatal("a transient mask was recorded")
	}
	s.EndFrame()
	s.BeginFrame()
	s.Record(&u)
	s.Mask(0xfeed, draw) // drawn in a later frame too: lasting
	s.Mask(0xfeed, draw) // found
	s.StopRecord()
	if len(u.entries) != 2 {
		t.Fatalf("%d entries recorded for a lasting mask made and found, want 2", len(u.entries))
	}
}

// A glyph run made while recording is recorded, as one found is.
func TestRecordSeesGlyphRuns(t *testing.T) {
	s := Shared()
	t.Cleanup(s.MakeRoom)
	l := s.Layout(Params{Text: "run run run", Style: Style{Size: 17}})
	var ids []uint32
	var xs []float32
	var f *Font
	for i, g := range l.Lines[0].Glyphs {
		f = g.Font
		ids = append(ids, g.ID)
		xs = append(xs, float32(i)*9+0.37)
	}
	var u Uses
	s.Record(&u)
	g := s.GlyphRun(f, ids, xs, 1, Flat, false)
	g2 := s.GlyphRun(f, ids, xs, 1, Flat, false)
	s.StopRecord()
	if !g.OK || !g2.OK {
		t.Skip("the run is not drawn as one image")
	}
	// Each call draws its glyphs alone first (Glyph), then makes or finds the run.
	if want := 2*len(ids) + 2; len(u.entries) != want {
		t.Fatalf("%d entries recorded for a run made and found, want %d", len(u.entries), want)
	}
}
