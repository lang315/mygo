package ui

import (
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/transfer"
)

// The fields of state, by what a retained row does with them (state.pending).
// A field in none of the groups, or in two, fails TestStatePendingPlacesEveryField:
// whoever adds one says which.
var statePlaced = map[string][]string{
	// pending: input or a request the next build reads, or a build that
	// still runs (an animation), or something the build sets again in every
	// frame that a kept row could not set. state.pending reports each.
	"pending": {
		"pendingSubmit", "pendingChange", "noticeQueued", "pendingEvents",
		"rowPressed", "pressChose", "clicks", "rightClicks", "doubleClicks",
		"typing", "dragX", "dragY", "pressed", "released", "pressPending",
		"changed", "submitted", "events", "expand", "holding", "dropped",
		"hasDropped", "dataDropped", "editor",
		"dragValue", "dragFn", "accepts", "dataSource", "dataTarget", "dragRec",
		"anims", "motions", "colorMotions", "sequences", "timelines", "crec", "pose", "trec",
		"shortcuts", "delivered", "wheel",
	},
	// kept: what lasts across frames, which a kept row keeps as it is: the
	// last frame's geometry and flags, scroll offsets, the focus-related
	// fields of the element, and what the app keeps in it.
	"kept": {
		"rt", "world", "clips", "rtl", "clipRect", "clipRadii", "id", "seen", "pass", "born",
		"lookedSeen", "lookedPass", "lay", "text", "x", "y", "w", "h", "vx", "vy", "vw", "vh",
		"parent", "flags", "cursor", "tip", "effectLayer", "webView",
		"scrollX", "scrollY", "contentW", "contentH", "barInset", "barsShown", "track",
		"moveFrame", "startX", "startY", "list", "cx", "cy", "cw", "ch", "role",
		"textScope", "spans", "valueBinding", "locals", "input", "textClient", "textAdapter",
		"caret", "takesText", "noIME", "copyData", "scope", "anchor", "page", "activated", "typed",
		"replay", // what the row painted (replay.go): painting memory, with its own key
		"krec",   // the record of the kept row: layout and commit memory, with its own checks
	},
	// neither: read only with a field that is pending, or set again by it.
	"neither": {
		"pressMods", "clickMods", "typedAt", "pressX", "pressY", "drag",
		"releaseVX", "releaseVY", "releaseWX", "releaseWY",
		"changeDelivered", "submitDelivered", "eventsDelivered", "submitMods",
		"holdStart", "holdSteps", "droppedValue", "dropX", "dropY",
	},
}

func TestStatePendingPlacesEveryField(t *testing.T) {
	placed := map[string]string{}
	for group, names := range statePlaced {
		for _, n := range names {
			if other, dup := placed[n]; dup {
				t.Errorf("state.%s is in %s and in %s", n, other, group)
			}
			placed[n] = group
		}
	}
	typ := reflect.TypeOf(state{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if placed[name] == "" {
			t.Errorf("state.%s is in no group of statePlaced: say what a retained row does with it", name)
		}
		delete(placed, name)
	}
	for n := range placed {
		t.Errorf("statePlaced names state.%s, which is not a field", n)
	}
}

// Each pending field makes pending report it, and a fresh state reports none.
func TestStatePendingReportsEachPendingField(t *testing.T) {
	set := map[string]func(s *state){
		"pendingSubmit": func(s *state) { s.pendingSubmit = true },
		"pendingChange": func(s *state) { s.pendingChange = true },
		"noticeQueued":  func(s *state) { s.noticeQueued = true },
		"pendingEvents": func(s *state) { s.pendingEvents = eventChecked },
		"rowPressed":    func(s *state) { s.rowPressed = true },
		"pressChose":    func(s *state) { s.pressChose = true },
		"clicks":        func(s *state) { s.clicks = 1 },
		"rightClicks":   func(s *state) { s.rightClicks = 1 },
		"doubleClicks":  func(s *state) { s.doubleClicks = 1 },
		"typing":        func(s *state) { s.typing = true },
		"dragX":         func(s *state) { s.dragX = 1 },
		"dragY":         func(s *state) { s.dragY = 1 },
		"pressed":       func(s *state) { s.pressed = true },
		"released":      func(s *state) { s.released = true },
		"pressPending":  func(s *state) { s.pressPending = true },
		"changed":       func(s *state) { s.changed = true },
		"submitted":     func(s *state) { s.submitted = true },
		"events":        func(s *state) { s.events = eventChecked },
		"expand":        func(s *state) { s.expand = 1 },
		"holding":       func(s *state) { s.holding = true },
		"dropped":       func(s *state) { s.dropped = []string{"a"} },
		"hasDropped":    func(s *state) { s.hasDropped = true },
		"dataDropped":   func(s *state) { s.dataDropped = new(transfer.Drop) },
		"editor":        func(s *state) { s.editor = new(editor) },
		"dragValue":     func(s *state) { s.dragValue = 1 },
		"dragFn":        func(s *state) { s.dragFn = func() any { return nil } },
		"accepts":       func(s *state) { s.accepts = func(any) bool { return true } },
		"dataSource":    func(s *state) { s.dataSource = new(dataSource) },
		"dataTarget":    func(s *state) { s.dataTarget = new(transfer.DropOptions) },
		"dragRec":       func(s *state) { s.dragRec = new(dragRec) },
		"anims":         func(s *state) { s.anims = map[any]*anim{1: nil} },
		"motions":       func(s *state) { s.motions = map[any]*channel{1: new(channel)} },
		"colorMotions":  func(s *state) { s.colorMotions = map[any]*colorMotion{1: nil} },
		"sequences":     func(s *state) { s.sequences = map[any]*sequence{1: nil} },
		"timelines":     func(s *state) { s.timelines = map[any]*timelineState{1: nil} },
		"crec":          func(s *state) { s.crec = new(contentRec) },
		"pose":          func(s *state) { s.pose = new(poseRec) },
		"trec":          func(s *state) { s.trec = new(transition) },
		"shortcuts":     func(s *state) { s.shortcuts = []shortcut{{}} },
		"delivered":     func(s *state) { s.delivered = []shortcut{{}} },
		"wheel":         func(s *state) { s.wheel = new(wheelGlide) },
	}
	if !slices.Equal(sortedKeys(set), slices.Sorted(slices.Values(statePlaced["pending"]))) {
		t.Fatalf("the cases name %v, statePlaced lists %v as pending", sortedKeys(set), statePlaced["pending"])
	}
	if (&state{}).pending() {
		t.Fatal("a fresh state is pending")
	}
	for name, f := range set {
		var s state
		f(&s)
		if !s.pending() {
			t.Errorf("state.%s is set, and pending does not report it", name)
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	})
}

// A value that animateRest made, at rest, does not make a state pending; one that
// moves, or waits out a delay, does; so does one made by any other call.
func TestStatePendingCountsOnlyMovingRestableMotions(t *testing.T) {
	for _, tc := range []struct {
		name string
		ch   channel
		want bool
	}{
		{"restable at rest", channel{rest: true}, false},
		{"restable moving", channel{rest: true, moving: true}, true},
		{"restable waiting out a delay", channel{rest: true, pending: true}, true},
		{"not restable at rest", channel{}, true},
	} {
		s := state{motions: map[any]*channel{1: &tc.ch}}
		if got := s.pending(); got != tc.want {
			t.Errorf("%s: pending = %v, want %v", tc.name, got, tc.want)
		}
	}
}
