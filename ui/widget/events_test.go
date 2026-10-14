package widget

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo"
)

func TestPostRunsInOrder(t *testing.T) {
	var got []int
	for i := range 100 {
		Post(func() { got = append(got, i) })
	}
	Update(func() {}) // runs after them
	if len(got) != 100 || !slices.IsSorted(got) {
		t.Fatalf("ran %v", got)
	}
}

func TestUpdateWaits(t *testing.T) {
	x := 0
	Update(func() { x = 1 })
	if x != 1 {
		t.Fatal("Update returned before fn ran")
	}
}

func TestUpdateInAHandlerRunsAtOnce(t *testing.T) {
	ran := false
	Update(func() {
		Update(func() { ran = true }) // would deadlock if queued
		if !ran {
			t.Error("nested Update did not run at once")
		}
	})
}

func TestBlockRunsHandlersOnTheEventGoroutine(t *testing.T) {
	var order []string
	Update(func() {
		done := make(chan struct{})
		Post(func() { order = append(order, "closer"); close(done) })
		block("test", done)
		order = append(order, "returned")
	})
	if !slices.Equal(order, []string{"closer", "returned"}) {
		t.Fatalf("order %v", order)
	}
}

func TestBlockNestsThreeDeep(t *testing.T) {
	var order []string
	d1, d2, d3 := make(chan struct{}), make(chan struct{}), make(chan struct{})
	Update(func() {
		Post(func() {
			Post(func() {
				Post(func() { close(d3); close(d2); close(d1) })
				block("d3", d3)
				order = append(order, "d3")
			})
			block("d2", d2)
			order = append(order, "d2")
		})
		block("d1", d1)
		order = append(order, "d1")
	})
	if !slices.Equal(order, []string{"d3", "d2", "d1"}) {
		t.Fatalf("order %v", order)
	}
}

func TestBlockingOnTheMainThreadPanics(t *testing.T) {
	for name, call := range map[string]func(){
		"block":  func() { block("Confirm", make(chan struct{})) },
		"Update": func() { Update(func() {}) },
	} {
		var msg string
		mygo.RunOnMain(func() {
			defer func() { msg, _ = recover().(string) }()
			call()
		})
		if !strings.Contains(msg, "main thread") {
			t.Errorf("%s on the main thread: panic %q, want one naming the main thread", name, msg)
		}
	}
	posted := make(chan struct{})
	mygo.RunOnMain(func() { Post(func() { close(posted) }) })
	<-posted
}

func TestUpdateFromManyGoroutines(t *testing.T) {
	n := 0
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 1000 {
				Update(func() { n++ })
			}
		})
	}
	wg.Wait()
	if n != 8000 {
		t.Fatalf("n = %d, want 8000", n)
	}
}

// Review Focus 1: a handler changes widgets while frames build.
func TestHandlerChangesWidgetsWhileFramesBuild(t *testing.T) {
	l := Label("0")
	tt := tester(t, l)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 2000 {
			Post(func() { l.SetText("x"); Update(func() {}) })
		}
	}()
	for range 50 {
		tt.Frame()
	}
	<-done
	Update(func() {}) // every posted handler ran, none deadlocked
}
