//go:build !js

package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func iconPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for i := range 32 {
		img.Set(i, i, color.RGBA{R: 255, A: 255})
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestMenuItemIcons(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Width: 400, Height: 300})
	w.Page().LoadHTML("<p>menus</p>", "")
	icon := iconPNG(t)
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "Icons", Submenu: []*mygo.MenuItem{
			{ID: "plain", Label: "Plain"},
			{ID: "icon", Label: "Icon", Icon: icon, Accelerator: "CmdOrCtrl+Shift+U"},
			{ID: "template", Label: "Template", Icon: icon, IconTemplate: true},
			{ID: "checked", Label: "Checked", Type: mygo.MenuItemCheckbox, Checked: true, Icon: icon},
			{ID: "bad", Label: "Bad", Icon: []byte("not a png")},
		}},
	})
	mygo.App.SetMenu(menu)
	defer mygo.App.SetMenu(nil)
	w.Focus()
	time.Sleep(100 * time.Millisecond)

	for _, tc := range []struct {
		label         string
		has, template bool
	}{{"Plain", false, false}, {"Icon", true, false}, {"Template", true, true}, {"Checked", true, false}, {"Bad", false, false}} {
		has, template, supported, err := menuIcon(w, "Icons", tc.label)
		if !supported {
			t.Skip("no menu icon hook on this platform")
		}
		if err != nil {
			t.Fatalf("%s: %v", tc.label, err)
		}
		// Only macOS reports the template flag; the other hooks return false.
		if has != tc.has || (has && runtime.GOOS == "darwin" && template != tc.template) {
			t.Errorf("%s: icon %v template %v, want %v %v", tc.label, has, template, tc.has, tc.template)
		}
	}
	if width, ok := accelWidth(w, "Icons", "Icon"); ok && width == 0 {
		t.Error("the item with an icon lost its shortcut")
	}
	// A relabeled item keeps its icon and shows its new label.
	menu.ItemByID("icon").SetLabel("Renamed")
	time.Sleep(100 * time.Millisecond)
	if has, _, _, err := menuIcon(w, "Icons", "Renamed"); err != nil || !has {
		t.Errorf("after SetLabel: icon %v, err %v", has, err)
	}
	if err := activateMenu(w, "Icons", "Checked"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for menu.ItemByID("checked").IsChecked() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if menu.ItemByID("checked").IsChecked() {
		t.Error("a checkbox with an icon did not toggle")
	}
}
