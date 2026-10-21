//go:build windows

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// fatalRecorder is a test whose Fatalf records its message and ends the
// goroutine it is called in, so that a hook failing its test can be tried.
type fatalRecorder struct {
	testing.TB
	msg string
}

func (r *fatalRecorder) Fatalf(format string, args ...any) {
	r.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// TestAppClickUnderAnotherAppFailsAtTheClick: a click that another app's
// window would take does not go to it: the test fails at the click, saying
// which window was there, rather than going on to wait for what the click
// would have done. A CI runner showed it: its agent's window was over the
// point, and the test timed out waiting for the effect of the click.
func TestAppClickUnderAnotherAppFailsAtTheClick(t *testing.T) {
	var clicks, frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Under", X: 300, Y: 300, Width: 400, Height: 300,
		Content: ui.View(func(c *ui.Context) {
			frames.Add(1)
			ui.Column(c).Fill().OnClick(func() { clicks.Add(1) })
		})})
	eventually(t, "the window's first frame", func() bool { return frames.Load() > 0 })

	// Another app's window over the whole window, topmost: bringing w to the
	// front does not put it above.
	ctx, cancel := context.WithTimeout(context.Background(), wait.Guard)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "MYGO_E2E_COVER_PEER=1", "MYGO_COVER_BOUNDS=250,250,500,400")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = stdin.Write([]byte("\n"))
		_ = stdin.Close()
		_ = cmd.Wait()
	})
	if ready, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || ready != "ready\n" {
		t.Fatalf("cover peer startup %q: %v", ready, err)
	}
	eventually(t, "another app's window over the point", func() bool { return !ownWindowAt(w, 60, 15) })

	rec := &fatalRecorder{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		appClick(rec, w, 60, 15)
	}()
	wait.For(t, "the click to return", done)
	if !strings.Contains(rec.msg, "the click at (60, 15) did not reach the app; the window there: ") || !strings.Contains(rec.msg, `title "Cover"`) {
		t.Fatalf("the click failed the test with %q, want the point and the Cover window in it", rec.msg)
	}
	if n := clicks.Load(); n != 0 {
		t.Fatalf("the click went through to the app's window %d times", n)
	}
}
