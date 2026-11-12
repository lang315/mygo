package skinhook

import (
	"sync"
	"testing"
)

// Get returns what Set stored for each appearance, and nil with no skin.
func TestSetGet(t *testing.T) {
	t.Cleanup(func() { Set(nil, nil) })
	if Get(false) != nil || Get(true) != nil {
		t.Fatalf("at start, Get gives %p and %p, want nil", Get(false), Get(true))
	}
	l, d := &Tokens{ControlHeight: 36}, &Tokens{ControlHeight: 30}
	Set(l, d)
	if Get(false) != l || Get(true) != d {
		t.Errorf("after Set(l, d), Get gives %p and %p, want %p and %p", Get(false), Get(true), l, d)
	}
	Set(nil, nil)
	if Get(false) != nil || Get(true) != nil {
		t.Errorf("after Set(nil, nil), Get gives %p and %p, want nil", Get(false), Get(true))
	}
}

// Set and Get from two goroutines do not race (go test -race), and Get
// always sees a pair that Set stored.
func TestSetGetRace(t *testing.T) {
	t.Cleanup(func() { Set(nil, nil) })
	l, d := &Tokens{ControlHeight: 36}, &Tokens{ControlHeight: 30}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 1000 {
			if i%2 == 0 {
				Set(l, d)
			} else {
				Set(nil, nil)
			}
		}
	}()
	bad := 0
	go func() {
		defer wg.Done()
		for range 1000 {
			if g := Get(false); g != nil && g != l {
				bad++
			}
		}
	}()
	wg.Wait()
	if bad != 0 {
		t.Errorf("Get gave %d times a value Set never stored", bad)
	}
	if Get(false) != nil {
		t.Errorf("after the last Set(nil, nil), Get(false) is %p, want nil", Get(false))
	}
}
