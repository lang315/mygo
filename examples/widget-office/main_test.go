package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

var fb *fake.Backend

// TestMain runs the tests on the fake backend, as package widget's do: a
// Confirm opens a window there, which a test answers.
func TestMain(m *testing.M) {
	fb = fake.New()
	platform.TestBackend = fb
	mygo.App.SetName("OfficeTest")
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		go func() { os.Exit(m.Run()) }()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// cur is the app of the test running: the helpers reach its orders.
var cur *office

// open builds the app in a form of the window's size and returns it with its
// tester. The skin is the process's: it is put back when the test ends.
func open(t *testing.T) (*office, *widget.Tester) {
	t.Helper()
	t.Cleanup(func() {
		skin.Use(nil)
		widget.SetTheme(widget.ThemeSystem)
	})
	w := widget.NewForm("Office", winW, winH)
	cur = build(w)
	return cur, widget.TestForm(t, w)
}

// tabs returns the labels of the order tabs shown.
func tabs(tt *widget.Tester) []string {
	var out []string
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleTab)) {
		if l := n.Label(); strings.HasPrefix(l, "Order ") {
			out = append(out, l)
		}
	}
	return out
}

// toasts returns the toasts of the last frame: the nodes of RoleStatus and
// RoleAlert.
func toasts(tt *widget.Tester) []ui.TestNode {
	tt.Frame()
	var out []ui.TestNode
	for _, n := range tt.Nodes() {
		if n.Role == ui.RoleStatus || n.Role == ui.RoleAlert {
			out = append(out, n)
		}
	}
	return out
}

// toast returns the one toast shown, failing without exactly one.
func toast(t *testing.T, tt *widget.Tester) ui.TestNode {
	t.Helper()
	cs := toasts(tt)
	if len(cs) != 1 {
		t.Fatalf("%d toasts, want 1: texts %q", len(cs), tt.Texts())
	}
	return cs[0]
}

// waitFake waits for the open fake window titled title: the dialog a Confirm
// shows.
func waitFake(t *testing.T, title string) *fake.Window {
	t.Helper()
	var found *fake.Window
	wait.Until(t, "an open window titled "+title, func() bool {
		for _, fw := range fb.Windows() {
			if fw.Title() == title && !fw.IsClosed() {
				found = fw
				return true
			}
		}
		return false
	})
	return found
}

// openWindow reports whether a window titled title is open.
func openWindow(title string) bool {
	for _, fw := range fb.Windows() {
		if fw.Title() == title && !fw.IsClosed() {
			return true
		}
	}
	return false
}

// press presses the button labeled label in fw's view, as a screen reader
// does, and waits for the window to close.
func press(t *testing.T, fw *fake.Window, label string) {
	t.Helper()
	var labels []string
	var nodes []platform.AccessNode
	mygo.RunOnMain(func() {
		s := fw.FakeSurface()
		s.Send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
		s.Frame()
		if tree, _ := s.Accessibility(); tree != nil {
			nodes = tree.Nodes
		}
	})
	for _, n := range nodes {
		if n.Label == label {
			mygo.RunOnMain(func() {
				s := fw.FakeSurface()
				s.Send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
				s.Frame()
			})
			wait.Until(t, "the dialog to close", fw.IsClosed)
			return
		}
		labels = append(labels, n.Label)
	}
	t.Fatalf("no %q among %q", label, labels)
}

// chooseCustomer types text in the Customer field of the order shown and
// presses Enter: the first customer it finds is chosen.
func chooseCustomer(tt *widget.Tester, text string) {
	cur.shown().customer.Focus()
	tt.Settle()
	tt.Type(text)
	tt.Settle()
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
}

// closeShown closes the order shown with Ctrl+F4, from its tab.
func closeShown(tt *widget.Tester) {
	tt.Find(widget.ByRole(ui.RoleTab).Label(fmt.Sprintf("Order %d", cur.shown().n))).Focus()
	tt.Key(ui.Ctrl, ui.KeyF4)
	tt.Settle()
}

// Case 1: the window builds at 1100 x 700: ribbon tabs "Home" and "Help";
// NavBar groups "Sales" and "Settings"; "Orders" selected; one tab "Order 1".
func TestOfficeBuilds(t *testing.T) {
	o, tt := open(t)
	if winW != 1100 || winH != 700 {
		t.Fatalf("the window is %d x %d, want 1100 x 700", winW, winH)
	}
	win := ui.Rect{W: winW, H: winH}
	for _, name := range []string{"Home", "Help"} {
		r := tt.Find(widget.ByRole(ui.RoleTab).Label(name)).Bounds()
		if r.W <= 0 || r.X+r.W > win.W || r.Y+r.H > win.H {
			t.Errorf("ribbon tab %s at %v is not in the window", name, r)
		}
	}
	for _, name := range []string{"Sales", "Settings"} {
		r := tt.Find(widget.ByLabel(name)).Bounds()
		if r.W <= 0 || r.X+r.W > win.W || r.Y+r.H > win.H {
			t.Errorf("NavBar group %s at %v is not in the window", name, r)
		}
	}
	if got := o.nav.Selected(); got != "orders" {
		t.Errorf("the NavBar selects %q, want orders", got)
	}
	if got := tabs(tt); !slices.Equal(got, []string{"Order 1"}) {
		t.Errorf("order tabs %q, want [Order 1]", got)
	}
	if tt.Has(widget.ByLabel("Order 2")) {
		t.Errorf("an Order 2 is shown at the start")
	}
	if !tt.Has(widget.ByLabel("Customer")) {
		t.Errorf("Order 1 shows no Customer row: texts %q", tt.Texts())
	}
}

// Case 2: a click on "New" shows a tab "Order 2".
func TestOfficeNewOrder(t *testing.T) {
	o, tt := open(t)
	tt.Find(widget.ByLabel("New")).Click()
	if got := tabs(tt); !slices.Equal(got, []string{"Order 1", "Order 2"}) {
		t.Fatalf("order tabs %q, want [Order 1 Order 2]", got)
	}
	if got := o.docs.Page(); got != 1 {
		t.Errorf("the page shown is %d, want 1 (Order 2)", got)
	}
}

// Case 3: Acme chosen for Order 1 (typed "ac", Enter), Save shows a toast
// "Order 1 saved for Acme" of status success.
func TestOfficeSaveToast(t *testing.T) {
	_, tt := open(t)
	chooseCustomer(tt, "ac")
	tt.Find(widget.ByLabel("Save")).Click()
	c := toast(t, tt)
	if !tt.Has(widget.ByLabel("Order 1 saved for Acme")) {
		t.Errorf("no toast titled the saved order: texts %q", tt.Texts())
	}
	if c.Role != ui.RoleStatus {
		t.Errorf("the toast's role is %v, want RoleStatus (not an alert)", c.Role)
	}
	// Warning, info and none share a role with success or differ from it by
	// the fill alone: success tints the card green. Advance is past the fade-in.
	tt.Advance(time.Second)
	fill := tt.Image().RGBAAt(int(c.Bounds.X+c.Bounds.W*0.02), int(c.Bounds.Y+c.Bounds.H/2))
	if fill.G <= fill.R || fill.G <= fill.B {
		t.Errorf("the toast's fill is %v, want a green one (status success)", fill)
	}
}

// Case 4: Save with no customer shows the warning toast "Choose a customer
// first".
func TestOfficeSaveNeedsCustomer(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByLabel("Save")).Click()
	c := toast(t, tt)
	if !tt.Has(widget.ByLabel("Choose a customer first")) {
		t.Errorf("no warning toast: texts %q", tt.Texts())
	}
	if c.Role != ui.RoleAlert {
		t.Errorf("the toast's role is %v, want RoleAlert (a warning)", c.Role)
	}
}

// Case 5, answered No: a changed order, Ctrl+F4, a Confirm "Discard Order
// 1?": No keeps Order 1.
func TestOfficeCloseChangedAnswerNo(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByRole(ui.RoleTextField)).Type("two boxes")
	closeShown(tt)
	fw := waitFake(t, "Discard Order 1?")
	if got := tabs(tt); !slices.Equal(got, []string{"Order 1"}) {
		t.Fatalf("order tabs %q while the question shows, want [Order 1]", got)
	}
	press(t, fw, "No")
	tt.Settle()
	if got := tabs(tt); !slices.Equal(got, []string{"Order 1"}) {
		t.Errorf("order tabs %q after No, want [Order 1]", got)
	}
}

// Case 5, answered Yes: the order closes.
func TestOfficeCloseChangedAnswerYes(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByRole(ui.RoleTextField)).Type("two boxes")
	closeShown(tt)
	fw := waitFake(t, "Discard Order 1?")
	press(t, fw, "Yes")
	tt.Settle()
	if got := tabs(tt); len(got) != 0 {
		t.Errorf("order tabs %q after Yes, want none", got)
	}
}

// Case 5, unchanged: an order closes with no question.
func TestOfficeCloseUnchangedAsksNothing(t *testing.T) {
	_, tt := open(t)
	closeShown(tt)
	if openWindow("Discard Order 1?") {
		t.Errorf("an unchanged order asked to be discarded")
	}
	if got := tabs(tt); len(got) != 0 {
		t.Errorf("order tabs %q after the close, want none", got)
	}
}

// A saved order is unchanged again: no question when it closes.
func TestOfficeSavedOrderClosesWithoutAsking(t *testing.T) {
	_, tt := open(t)
	chooseCustomer(tt, "ac")
	tt.Find(widget.ByRole(ui.RoleTextField)).Type("fragile")
	tt.Find(widget.ByLabel("Save")).Click()
	closeShown(tt)
	if openWindow("Discard Order 1?") {
		t.Errorf("a saved order asked to be discarded")
	}
	if got := tabs(tt); len(got) != 0 {
		t.Errorf("order tabs %q after the close, want none", got)
	}
}

// A changed customer is a change too, and the ribbon's Close asks as Ctrl+F4
// does.
func TestOfficeCloseButtonAsksWhenCustomerChanged(t *testing.T) {
	_, tt := open(t)
	chooseCustomer(tt, "ac")
	tt.Find(widget.ByLabel("Close")).Click()
	fw := waitFake(t, "Discard Order 1?")
	press(t, fw, "Yes")
	tt.Settle()
	if got := tabs(tt); len(got) != 0 {
		t.Errorf("order tabs %q after Yes, want none", got)
	}
}

// The counter goes on after a close: the next order is Order 2, never Order
// 1 again.
func TestOfficeOrderNumbersGoOn(t *testing.T) {
	_, tt := open(t)
	closeShown(tt)
	tt.Find(widget.ByLabel("New")).Click()
	if got := tabs(tt); !slices.Equal(got, []string{"Order 2"}) {
		t.Errorf("order tabs %q, want [Order 2]", got)
	}
}

// Save is of the order shown: the second order's toast names its own number.
func TestOfficeSaveIsOfTheOrderShown(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByLabel("New")).Click()
	chooseCustomer(tt, "birch")
	tt.Find(widget.ByLabel("Save")).Click()
	if !tt.Has(widget.ByLabel("Order 2 saved for Birch")) {
		t.Errorf("no toast of Order 2: texts %q", tt.Texts())
	}
}

// An order closed leaves the others to their own tabs: after Order 1 goes,
// Save and the close question are of Order 2.
func TestOfficeClosedOrderLeavesTheOthers(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByLabel("New")).Click()
	tt.Find(widget.ByRole(ui.RoleTab).Label("Order 1")).Click()
	closeShown(tt) // Order 1, unchanged: no question
	if openWindow("Discard Order 1?") {
		t.Fatalf("an unchanged order asked to be discarded")
	}
	if got := tabs(tt); !slices.Equal(got, []string{"Order 2"}) {
		t.Fatalf("order tabs %q, want [Order 2]", got)
	}
	chooseCustomer(tt, "birch")
	tt.Find(widget.ByLabel("Save")).Click()
	if !tt.Has(widget.ByLabel("Order 2 saved for Birch")) {
		t.Fatalf("no toast of Order 2: texts %q", tt.Texts())
	}
	closeShown(tt) // saved: no question
	if openWindow("Discard Order 2?") {
		t.Errorf("a saved order asked to be discarded")
	}
	if got := tabs(tt); len(got) != 0 {
		t.Errorf("order tabs %q after the close, want none", got)
	}
}

// Case 6: "Customers" shows a tree with the header Customer, City and
// Orders; a region opens on its customers and their city.
func TestOfficeCustomersTree(t *testing.T) {
	o, tt := open(t)
	tt.Find(widget.ByLabel("Customers")).Click()
	if got := o.nav.Selected(); got != "customers" {
		t.Fatalf("the NavBar selects %q, want customers", got)
	}
	for _, h := range []string{"Customer", "City", "Orders"} {
		if !tt.Has(widget.ByRole(ui.RoleColumnHeader).Label(h)) {
			t.Errorf("no column header %q: texts %q", h, tt.Texts())
		}
	}
	if tt.Has(widget.ByLabel("Acme")) {
		t.Errorf("a customer shows before its region is open")
	}
	tt.Find(widget.ByRole(ui.RoleTreeItem).Label("North")).Key(0, ui.KeyRight)
	row := tt.Find(widget.ByRole(ui.RoleTreeItem).Label("Acme"))
	city := tt.Find(widget.ByLabel("Hanoi"))
	if r, c := row.Bounds(), city.Bounds(); c.Y < r.Y || c.Y+c.H > r.Y+r.H {
		t.Errorf("Hanoi at %v is not on Acme's row %v", c, r)
	}
	if tt.Has(widget.ByRole(ui.RoleTab).Label("Order 1")) {
		t.Errorf("the order tabs show on the Customers page")
	}
}

// Case 7: Navigation off takes the NavBar away and the page its width; on
// again brings it back.
func TestOfficeNavigationToggle(t *testing.T) {
	_, tt := open(t)
	page := func() ui.Rect { return tt.Find(widget.ByRole(ui.RoleTab).Label("Order 1")).Bounds() }
	with := page()
	if !tt.Has(widget.ByLabel("Sales")) {
		t.Fatalf("no NavBar at the start")
	}
	tt.Find(widget.ByLabel("Navigation")).Click()
	if tt.Has(widget.ByLabel("Sales")) || tt.Has(widget.ByLabel("Settings")) {
		t.Errorf("the NavBar shows with Navigation off: texts %q", tt.Texts())
	}
	without := page()
	if without.X >= with.X || without.X > 8 {
		t.Errorf("the page starts at x=%v without the NavBar, %v with it: it did not take the width", without.X, with.X)
	}
	tt.Find(widget.ByLabel("Navigation")).Click()
	if !tt.Has(widget.ByLabel("Sales")) {
		t.Errorf("the NavBar is not back with Navigation on: texts %q", tt.Texts())
	}
	if again := page(); again != with {
		t.Errorf("the page is at %v, was %v before", again, with)
	}
}

// Case 8: F10, the Home tab's letter, New's letter open a new order tab.
// Key tips are off on macOS by design.
func TestOfficeKeyTips(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("key tips are off on macOS, whose menu bar has the access keys (ui/widget/menustrip.go:12 macMenus)")
	}
	_, tt := open(t)
	tt.Find(widget.ByLabel("Order 1")).Focus() // not in the ribbon: the tips take the focus from here
	tt.Key(0, ui.KeyF10)
	tt.Key(0, ui.KeyH)
	tt.Key(0, ui.KeyN)
	tt.Settle()
	if got := tabs(tt); !slices.Equal(got, []string{"Order 1", "Order 2"}) {
		t.Errorf("order tabs %q after F10 H N, want [Order 1 Order 2]", got)
	}
}

// Case 9: every built-in skin and none, light and dark: the window builds,
// and cases 1 and 2 hold.
func TestOfficeEverySkin(t *testing.T) {
	skins := append([]*skin.Skin{nil}, skin.Builtin()...)
	for _, s := range skins {
		name := "none"
		if s != nil {
			name = s.Name()
		}
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dark %v", name, dark), func(t *testing.T) {
				t.Cleanup(func() { skin.Use(nil) })
				skin.Use(s)
				o, tt := open(t)
				tt.SetDark(dark)
				tt.Frame()
				for _, tab := range []string{"Home", "Help"} {
					tt.Find(widget.ByRole(ui.RoleTab).Label(tab))
				}
				for _, group := range []string{"Sales", "Settings"} {
					tt.Find(widget.ByLabel(group))
				}
				if o.nav.Selected() != "orders" || !slices.Equal(tabs(tt), []string{"Order 1"}) {
					t.Errorf("selected %q, tabs %q", o.nav.Selected(), tabs(tt))
				}
				tt.Find(widget.ByLabel("New")).Click()
				if got := tabs(tt); !slices.Equal(got, []string{"Order 1", "Order 2"}) {
					t.Errorf("order tabs %q, want [Order 1 Order 2]", got)
				}
			})
		}
	}
}

// Case 10: a double click on the Home tab collapses the ribbon, the NavBar
// and the pages move up; another expands it.
func TestOfficeRibbonCollapse(t *testing.T) {
	o, tt := open(t)
	nav := func() float32 { return tt.Find(widget.ByLabel("Sales")).Bounds().Y }
	page := func() float32 { return tt.Find(widget.ByRole(ui.RoleTab).Label("Order 1")).Bounds().Y }
	navY, pageY := nav(), page()
	tt.Find(widget.ByRole(ui.RoleTab).Label("Home")).DoubleClick()
	if !o.ribbon.Collapsed() {
		t.Fatalf("the ribbon is not collapsed after a double click")
	}
	if nav() >= navY || page() >= pageY {
		t.Errorf("after the collapse the NavBar is at y=%v (was %v) and the page at y=%v (was %v): they did not move up", nav(), navY, page(), pageY)
	}
	tt.Find(widget.ByRole(ui.RoleTab).Label("Home")).DoubleClick()
	if o.ribbon.Collapsed() {
		t.Fatalf("the ribbon is still collapsed after the second double click")
	}
	if nav() != navY || page() != pageY {
		t.Errorf("after the expand the NavBar is at y=%v (was %v) and the page at y=%v (was %v)", nav(), navY, page(), pageY)
	}
}

// The Skin page applies the skin chosen, and the window keeps working.
func TestOfficeSkinPage(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByLabel("Skin")).Click()
	tt.Find(widget.ByRole(ui.RoleListItem).Label("Classic")).Click()
	if got := skin.Active(); got == nil || got.Name() != "Classic" {
		t.Fatalf("the active skin is %v, want Classic", got)
	}
	tt.Find(widget.ByLabel("Orders")).Click()
	tt.Find(widget.ByLabel("New")).Click()
	if got := tabs(tt); !slices.Equal(got, []string{"Order 1", "Order 2"}) {
		t.Errorf("order tabs %q after the skin, want [Order 1 Order 2]", got)
	}
}

// The About command shows a toast.
func TestOfficeAbout(t *testing.T) {
	_, tt := open(t)
	tt.Find(widget.ByRole(ui.RoleTab).Label("Help")).Click()
	tt.Find(widget.ByRole(ui.RoleButton).Label("About")).Click()
	c := toast(t, tt)
	if c.Role != ui.RoleStatus {
		t.Errorf("the About toast's role is %v, want RoleStatus", c.Role)
	}
}
