//go:build windows && (amd64 || arm64)

package windows

import "fmt"

var (
	procRemoveClipboardFormatListener = user32.NewProc("RemoveClipboardFormatListener")
	procGetClipboardSequenceNumber    = user32.NewProc("GetClipboardSequenceNumber")
)

// WatchChanges shares the hidden window's listener with listenForReplacement
// (clipboard.go), which keeps it when the watch stops.
func (c clipboard) WatchChanges(changed func()) (func(), error) {
	sequence, _, _ := procGetClipboardSequenceNumber.Call()
	if err := c.listen(); err != nil {
		return nil, fmt.Errorf("mygo: watch clipboard: %w", err)
	}
	c.b.clipboardChanged = func() {
		current, _, _ := procGetClipboardSequenceNumber.Call()
		if current != sequence {
			sequence = current
			changed()
		}
	}
	return func() {
		c.b.clipboardChanged = nil
		if !clipboardReplacing && clipboardListening {
			procRemoveClipboardFormatListener.Call(c.b.appHwnd)
			clipboardListening = false
		}
	}, nil
}
