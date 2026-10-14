package main

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestPercentile(t *testing.T) {
	var d []time.Duration
	for i := 100; i >= 1; i-- {
		d = append(d, time.Duration(i)*time.Millisecond)
	}
	before := slices.Clone(d)
	for _, tc := range []struct {
		p    int
		want time.Duration
	}{{50, 50 * time.Millisecond}, {95, 95 * time.Millisecond}, {100, 100 * time.Millisecond}} {
		if got := percentile(d, tc.p); got != tc.want {
			t.Errorf("p%d = %v, want %v", tc.p, got, tc.want)
		}
	}
	if !slices.Equal(d, before) {
		t.Error("percentile sorted its input")
	}
	if percentile(nil, 95) != 0 {
		t.Error("the percentile of nothing is not 0")
	}
}

func TestStatsLine(t *testing.T) {
	a := newApp(10)
	a.view.Sync(a.s.accounts, a.sort) // as a frame does
	t0 := time.Unix(1_800_000_000, 0)
	s := newStats(t0)
	ms := time.Millisecond
	s.frame(t0, t0.Add(2*ms))
	s.frame(t0.Add(16*ms), t0.Add(17*ms))
	s.frame(t0.Add(50*ms), t0.Add(53*ms))
	s.applied(5, ms)
	if line := s.flush(t0.Add(500*ms), a); line != "" {
		t.Fatalf("a line before a second: %q", line)
	}
	line := s.flush(t0.Add(time.Second), a)
	want := `stats t=1s rows=10 shown=10 sort="-" updates=5/s batches=1 apply_max=1.0ms frames=3 interval_p50=16.0ms p95=34.0ms max=34.0ms build_p50=2.0ms p95=3.0ms max=3.0ms heap=`
	if !strings.HasPrefix(line, want) || !strings.Contains(line, " gc=") || !strings.Contains(line, " goroutines=") {
		t.Fatalf("line\n%q\nwant it to start\n%q", line, want)
	}
	// The next second starts afresh.
	a.sort = ui.SortOrder{Column: "Bạn bè", Descending: true}
	line = s.flush(t0.Add(2*time.Second), a)
	if !strings.Contains(line, `t=2s rows=10 shown=10 sort="Bạn bè↓" updates=0/s batches=0 apply_max=0.0ms frames=0 `) {
		t.Fatalf("second line %q", line)
	}
}
