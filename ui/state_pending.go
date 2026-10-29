package ui

// pending reports whether s holds input, a request or a running build that the
// next build of its element reads or sets again: a row whose states are none
// of these builds what it built in the last frame, so it may be kept instead
// of built (retain.go). TestStatePendingPlacesEveryField places every field of
// state, and TestStatePendingReportsEachPendingField checks each one here.
func (s *state) pending() bool {
	// Input the pass applies: pointer, keys, edits, drops.
	if s.clicks != 0 || s.rightClicks != 0 || s.doubleClicks != 0 ||
		s.pressed || s.pressPending || s.released || s.rowPressed || s.pressChose ||
		s.dragX != 0 || s.dragY != 0 || s.typing || s.holding ||
		len(s.dropped) != 0 || s.hasDropped || s.dataDropped != nil || s.editor != nil {
		return true
	}
	// What the controls report or are asked, read as they build.
	if s.changed || s.submitted || s.pendingSubmit || s.pendingChange ||
		s.noticeQueued || s.pendingEvents != 0 || s.events != 0 || s.expand != 0 {
		return true
	}
	// What the build sets in every frame, which a kept row could not.
	if s.dragValue != nil || s.dragFn != nil || s.accepts != nil ||
		s.dataSource != nil || s.dataTarget != nil || s.dragRec != nil ||
		len(s.shortcuts) != 0 || len(s.delivered) != 0 {
		return true
	}
	// Builds that run on: animations, transitions, a wheel gliding.
	return len(s.anims) != 0 || len(s.motions) != 0 || len(s.colorMotions) != 0 ||
		len(s.sequences) != 0 || len(s.timelines) != 0 ||
		s.crec != nil || s.pose != nil || s.trec != nil || s.wheel != nil
}
