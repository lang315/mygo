package e2e

import (
	"strings"
	"testing"
	"time"
)

// steps times the steps of a test, logged only if it fails, so a failing
// wait says which step before it stalled.
type steps struct {
	last  time.Time
	marks []string
}

// logSteps starts timing the steps of t; they are logged if t fails.
func logSteps(t *testing.T) *steps {
	s := &steps{last: time.Now()}
	t.Cleanup(func() {
		if t.Failed() {
			t.Log("steps: " + s.String())
		}
	})
	return s
}

// done ends a step, named by what it did.
func (s *steps) done(name string) {
	now := time.Now()
	s.marks = append(s.marks, name+" "+now.Sub(s.last).Round(time.Millisecond).String())
	s.last = now
}

func (s *steps) String() string { return strings.Join(s.marks, ", ") }

func TestStepsNameEachStepWithItsDuration(t *testing.T) {
	s := &steps{last: time.Now()}
	time.Sleep(20 * time.Millisecond)
	s.done("first")
	s.done("second")
	got := s.String()
	if !strings.HasPrefix(got, "first 2") || !strings.Contains(got, ", second 0s") {
		t.Errorf("steps = %q, want the first with about 20ms and the second with none", got)
	}
}
