package widget

import (
	"sync"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// measureThreads is a control that records, for each measure of its view, whether
// it ran on the main thread, where the windows draw: a measure builds with the
// process's text system, whose glyph atlas is for one thread only.
type measureThreads struct {
	mu     sync.Mutex
	onMain []bool
}

func (m *measureThreads) control() *CustomWidget {
	return Custom(func(c *ui.Context) ui.Element {
		if c.Measuring() {
			main := mygo.IsMainThread()
			m.mu.Lock()
			m.onMain = append(m.onMain, main)
			m.mu.Unlock()
		}
		return ui.Text(c, "measured")
	})
}

func (m *measureThreads) assert(t *testing.T, what string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.onMain) == 0 {
		t.Fatalf("%s: the view was never measured", what)
	}
	for _, main := range m.onMain {
		if !main {
			t.Fatalf("%s: the view was measured off the main thread: %v", what, m.onMain)
		}
	}
}

func TestRefitOfAShownFormMeasuresOnTheMainThread(t *testing.T) {
	var m measureThreads
	f := NewForm("Fit thread", 420, 200)
	f.SetContent(Label("splash"))
	f.Show()
	t.Cleanup(f.Close)
	waitFake(t, "Fit thread")
	f.SetContent(m.control())
	f.SetSize(600, 0)
	drain()
	m.assert(t, "SetSize(w, 0) of a shown form")
}

func TestFirstShowOfADialogMeasuresOnTheMainThread(t *testing.T) {
	var m measureThreads
	d := NewDialog(nil, "Fit thread dialog", "", 500, 0, m.control())
	done := make(chan struct{})
	go func() { defer close(done); d.Show() }() // a goroutine, as an app's worker or handler is
	wait.For(t, "Show to return", done)
	t.Cleanup(d.Close)
	m.assert(t, "the first Show of a dialog")
}
