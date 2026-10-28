package text

// Uses are the atlas entries a painting drew from (Record), so that a later
// frame that shows what it painted without painting it again marks them used
// again (Touch): MakeRoom forgets the entries no frame marked.
type Uses struct {
	entries []*atlasEntry
}

// Reset empties u, keeping its room.
func (u *Uses) Reset() {
	clear(u.entries)
	u.entries = u.entries[:0]
}

// Record has every atlas entry marked used from now on, by Glyph, Glyphs,
// GlyphRun and Mask, also put in u, which it empties first, until StopRecord.
func (s *System) Record(u *Uses) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u.Reset()
	s.rec = u
}

// StopRecord ends Record.
func (s *System) StopRecord() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rec = nil
}

// Touch marks the entries of u used in this frame, as drawing them would.
func (s *System) Touch(u *Uses) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range u.entries {
		e.used = s.frame
	}
}

// use marks e used in this frame, recording it when a painting records.
// s.mu is held.
func (s *System) use(e *atlasEntry) {
	e.used = s.frame
	if s.rec != nil {
		s.rec.entries = append(s.rec.entries, e)
	}
}
