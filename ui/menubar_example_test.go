package ui_test

import "github.com/egoist/mygo/ui"

type barExampleApp struct {
	open  [2]bool
	first int
}

func (app *barExampleApp) items(m *ui.Menu, i int) {}

// The menu bar snippet of docs/ui/menu-button.md, which must compile.
func ExampleMenuParts_PopupMenu() {
	app := &barExampleApp{first: -1}
	view := func(c *ui.Context) {
		ui.Row(c).Role(ui.RoleMenuBar).Children(func() {
			for i, title := range []string{"File", "Edit"} {
				m := ui.MenuBase(c, &app.open[i])
				m.Trigger.Unfocusable().KeepFocus().Role(ui.RoleMenuItem).Label(title).Padding(4, 8)
				m.Trigger.Children(func() { ui.Text(c, title) })
				if app.first == i {
					app.first = -1
					m.OpenFirst() // on its first item, as the keys open it
				}
				m.PopupMenu(func(m *ui.Menu) { app.items(m, i) })
				if d := m.Across(); d != 0 { // Left or Right: the menu before or after
					app.open[i], app.first = false, (i+d+2)%2
				}
			}
		})
	}
	_ = view
}
