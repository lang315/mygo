package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourCustomDialogEnterAcceptsAndEscapeCancels(t *testing.T) {
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeyEscape} {
		d := newCustomDialog(nil)
		dt := widget.TestDialog(t, d)
		done := make(chan struct{})
		go func() { d.Run(); close(done) }()
		node := dt.Find(widget.ByRole(ui.RoleTextField))
		node.Type("x").Key(0, key)
		<-done
		if !dt.Closed() {
			t.Fatalf("key %v did not close the dialog", key)
		}
	}
}
