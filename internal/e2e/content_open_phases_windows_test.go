//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/wait"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// DRAFT: where the time of opening a window with Content goes, by phase.
func TestContentWindowOpenPhases(t *testing.T) {
	const rounds = 12
	built := make(chan struct{}, 1)
	view := func(c *ui.Context) {
		win.TestMark("view")
		ui.Box(c).Fill().Padding(16).Gap(8).Children(func() {
			ui.Text(c, "Bench").FontSize(20)
			for i := range 20 {
				ui.Text(c, "Row "+strconv.Itoa(i))
			}
		})
		select {
		case built <- struct{}{}:
		default:
		}
	}
	type trace struct {
		names []string
		at    []time.Duration
	}
	d3d11.Trace = win.TestMark
	t.Cleanup(func() { d3d11.Trace = func(string) {} })
	var runs []trace
	var totals []time.Duration
	for i := 0; i < rounds; i++ {
		mygo.RunOnMain(win.TestPhasesStart)
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Bench", Width: 400, Height: 300, Content: ui.View(view)})
		select {
		case <-built:
		case <-wait.Context(t).Done():
			t.Fatal("no frame")
		}
		mygo.RunOnMain(func() { win.TestMark("built seen") })
		w.Destroy()
		mygo.RunOnMain(func() { win.TestMark("Destroy returned") })
		names, at := func() (n []string, a []time.Duration) {
			mygo.RunOnMain(func() { n, a = win.TestPhases() })
			return
		}()
		runs = append(runs, trace{names, at})
		totals = append(totals, at[len(at)-1])
	}
	var b strings.Builder
	fmt.Fprintf(&b, "DEBUG phases (ms from NewWindow call; first run, then median of the other %d)\n", rounds-1)
	first := runs[0]
	for i, n := range first.names {
		fmt.Fprintf(&b, "  first %-22s %8.2f\n", n, float64(first.at[i].Microseconds())/1000)
	}
	// Median offset of each phase name's first occurrence, in the order of the second run.
	seen := map[string]bool{}
	for i, n := range runs[1].names {
		if seen[n] {
			continue
		}
		seen[n] = true
		var v []float64
		for _, r := range runs[1:] {
			if j := slices.Index(r.names, n); j >= 0 {
				v = append(v, float64(r.at[j].Microseconds())/1000)
			}
		}
		slices.Sort(v)
		_ = i
		fmt.Fprintf(&b, "  median %-22s %8.2f (n=%d)\n", n, v[len(v)/2], len(v))
	}
	var tm []float64
	for _, d := range totals {
		tm = append(tm, float64(d.Microseconds())/1000)
	}
	fmt.Fprintf(&b, "  totals ms: %.1f\n", tm)
	t.Fatal(b.String())
}
