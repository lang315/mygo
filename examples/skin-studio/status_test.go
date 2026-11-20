package main

import (
	"testing"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

// The status controls under the message boxes of the Dialogs and messages
// page, with no skin and each built-in one, light and dark: a badge of each status (Running with its step), six info bars
// (Warning and Danger alerts, the rest statuses), the action and close, and
// the empty state's action, all inside the window; the action and close
// work.
func TestStatusPage(t *testing.T) {
	reset(t)
	for _, s := range []*skin.Skin{nil, skin.Calm, skin.Classic, skin.HighContrast} {
		for _, dark := range []bool{false, true} {
			skin.Use(s)
			tt := widget.TestForm(t, newStudio())
			tt.SetSize(formWidth, formHeight)
			tt.SetDark(dark)
			tt.Find(widget.ByRole(ui.RoleTab).Label("Dialogs and messages")).Click()
			tt.Frame()
			name := describe(s)
			for _, b := range []string{"Draft", "Đồng bộ", "Đang chạy, Bước 2/5", "Live", "Cần xác minh", "Sai mật khẩu"} {
				if !tt.Has(widget.ByRole(ui.RoleText).Label(b)) {
					t.Errorf("%s, dark %v: no badge %q", name, dark, b)
				}
			}
			alerts, statuses := 0, 0
			var bottom float32
			for _, n := range tt.Nodes() {
				switch n.Role {
				case ui.RoleAlert:
					alerts++
				case ui.RoleStatus:
					statuses++
				}
				if n.Bounds.H > 0 {
					bottom = max(bottom, n.Bounds.Y+n.Bounds.H)
				}
			}
			if alerts != 2 || statuses < 4 {
				t.Errorf("%s, dark %v: %d alerts and %d statuses, want 2 and at least 4", name, dark, alerts, statuses)
			}
			if bottom > formHeight {
				t.Errorf("%s, dark %v: the page ends at %v, below the window's %d", name, dark, bottom, formHeight)
			}
			for _, b := range []string{"Xác minh", "Close", "Thêm tài khoản"} {
				if !tt.Has(widget.ByRole(ui.RoleButton).Label(b)) {
					t.Errorf("%s, dark %v: no button %q", name, dark, b)
				}
			}
			for _, c := range clipped(t, tt, formWidth, formHeight) {
				t.Errorf("%s, dark %v: %s", name, dark, c)
			}
		}
	}
	skin.Use(skin.Calm)
	tt := widget.TestForm(t, newStudio())
	tt.Find(widget.ByRole(ui.RoleTab).Label("Dialogs and messages")).Click()
	tt.Find(widget.ByRole(ui.RoleButton).Label("Xác minh")).Click()
	wait.Until(t, "the action to show", func() bool { tt.Frame(); return tt.Has(widget.ByLabel("Verify clicked")) })
	tt.Find(widget.ByRole(ui.RoleButton).Label("Close")).Click()
	wait.Until(t, "the close to show", func() bool { tt.Frame(); return tt.Has(widget.ByLabel("Info bar closed")) })
	if tt.Has(widget.ByLabel("Đã lưu")) {
		t.Error("the closed info bar still shows")
	}
}
