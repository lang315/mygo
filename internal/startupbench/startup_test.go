// Package startupbench measures how long an app takes from its process being
// created to its window shown and its first frame (native UI) or DOM ready
// (a page), on the machine it runs on. TEMPORARY: a measurement, not a check.
package startupbench

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const runs = 15

type sample map[string]time.Duration // a milestone: time since the process was started

func build(t *testing.T, dir, name, pkg, tags string) string {
	t.Helper()
	exe := filepath.Join(dir, name)
	ld := "-s -w -X github.com/egoist/mygo.production=1"
	if runtime.GOOS == "windows" {
		exe += ".exe"
		ld += " -H=windowsgui"
	}
	cmd := exec.Command("go", "build", "-trimpath", "-tags", tags, "-ldflags", ld, "-o", exe, pkg)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s: %v\n%s", pkg, err, out)
	}
	return exe
}

func run(t *testing.T, exe, logFile string, env ...string) sample {
	t.Helper()
	os.Remove(logFile)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(append(os.Environ(), "MYGO_STARTUP_LOG="+logFile), env...)
	t0 := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	spawned := time.Since(t0)
	err := cmd.Wait()
	s := sample{"spawn call": spawned}
	f, ferr := os.Open(logFile)
	if ferr != nil {
		t.Logf("no milestones (%v, exit %v)", ferr, err)
		return s
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, ns, _ := strings.Cut(sc.Text(), " ")
		n, _ := strconv.ParseInt(ns, 10, 64)
		s[name] = time.Unix(0, n).Sub(t0)
	}
	s["exit"] = time.Since(t0)
	return s
}

func median(v []time.Duration) time.Duration {
	slices.Sort(v)
	return v[len(v)/2]
}

func ms(d time.Duration) string { return fmt.Sprintf("%6.1f", float64(d.Microseconds())/1000) }

// TestStartup writes the table of phases, in milliseconds, of each app.
func TestStartup(t *testing.T) {
	if os.Getenv("MYGO_STARTUP_BENCH") == "" {
		t.Skip("a measurement: set MYGO_STARTUP_BENCH=1")
	}
	dir := t.TempDir()
	type app struct{ name, pkg, tags, env string }
	apps := []app{
		{"native", "github.com/egoist/mygo/internal/startupbench/apps/native", "mygo_noinspector,mygo_nowebview", ""},
	}
	if runtime.GOOS == "windows" { // the prewarm of the GPU kit is Windows's
		apps = append(apps, app{"native (prewarm off)", "github.com/egoist/mygo/internal/startupbench/apps/native", "mygo_noinspector,mygo_nowebview", "MYGO_PREWARM=off"})
	}
	apps = append(apps, app{"web", "github.com/egoist/mygo/internal/startupbench/apps/web", "mygo_noinspector", ""})
	var report strings.Builder
	fmt.Fprintf(&report, "\nSTARTUP %s/%s, %d CPUs, %s\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), time.Now().UTC().Format(time.RFC3339))
	for _, a := range apps {
		exe := build(t, dir, strings.NewReplacer("+", "-", "=", "-").Replace(a.name), a.pkg, a.tags)
		var all []sample
		for i := 0; i < runs+1; i++ {
			s := run(t, exe, filepath.Join(dir, "m.log"), a.env)
			all = append(all, s)
		}
		phases := []struct{ label, from, to string }{
			{"process start -> main()", "", "main"},
			{"main() -> App ready", "main", "appready"},
			{"App ready -> window shown", "appready", "show"},
			{"window shown -> first frame / ready", "show", "ready"},
		}
		if strings.HasPrefix(a.name, "web") {
			phases = append(phases, struct{ label, from, to string }{"  (window shown -> DOM ready)", "show", "dom"})
		}
		phases = append(phases, struct{ label, from, to string }{"process start -> ready (total)", "", "ready"})
		started := 0
		for _, s := range all {
			if _, ok := s["prewarm-started"]; ok {
				started++
			}
		}
		fmt.Fprintf(&report, "%s (%s):  first run | median of %d later; prewarm started in %d of %d runs\n", a.name, a.tags, runs, started, len(all))
		for _, p := range phases {
			var v []time.Duration
			var first time.Duration
			for i, s := range all {
				to, ok := s[p.to]
				if !ok {
					continue
				}
				from := time.Duration(0)
				if p.from != "" {
					from, ok = s[p.from]
					if !ok {
						continue
					}
				}
				if i == 0 {
					first = to - from
				} else {
					v = append(v, to-from)
				}
			}
			if len(v) == 0 {
				fmt.Fprintf(&report, "  %-38s no data\n", p.label)
				continue
			}
			fmt.Fprintf(&report, "  %-38s %s | %s ms  (min %s max %s)\n", p.label, ms(first), ms(median(slices.Clone(v))), ms(slices.Min(v)), ms(slices.Max(v)))
		}
	}
	t.Log(report.String())
	if sum := os.Getenv("GITHUB_STEP_SUMMARY"); sum != "" {
		if f, err := os.OpenFile(sum, os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "```\n%s```\n", report.String())
			f.Close()
		}
	}
}
