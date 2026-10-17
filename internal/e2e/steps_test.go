package e2e

import (
	"strings"
	"testing"
	"time"
)

// steps times the steps of a test, logged only if it fails, so a failing
// wait says which step before it stalled.
type steps struct {
	now   func() time.Time
	last  time.Time
	marks []string
}

// logSteps starts timing the steps of t; they are logged if t fails.
func logSteps(t *testing.T) *steps {
	s := &steps{now: time.Now, last: time.Now()}
	t.Cleanup(func() {
		if t.Failed() {
			t.Log("steps: " + s.String())
		}
	})
	return s
}

// done ends a step, named by what it did.
func (s *steps) done(name string) {
	now := s.now()
	s.marks = append(s.marks, name+" "+now.Sub(s.last).Round(time.Millisecond).String())
	s.last = now
}

func (s *steps) String() string { return strings.Join(s.marks, ", ") }

func TestStepsNameEachStepWithItsDuration(t *testing.T) {
	// A clock of the test's own, which only the test moves: the time a step
	// took on a busy machine, or the tick of Windows' timer, is not the test's.
	clock := time.Unix(0, 0)
	s := &steps{now: func() time.Time { return clock }, last: clock}
	clock = clock.Add(20 * time.Millisecond)
	s.done("first")
	s.done("second")
	if got, want := s.String(), "first 20ms, second 0s"; got != want {
		t.Errorf("steps = %q, want %q", got, want)
	}
}
