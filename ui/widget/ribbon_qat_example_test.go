package widget_test

import "github.com/egoist/mygo/ui/widget"

// The Quick Access Toolbar and the contextual tab of docs/ui/widget/ribbon.md:
// Save and Undo above the tabs, the user's choice saved by name, and a
// Format tab while a picture is selected.
func ExampleRibbonWidget_AddQuickAccess() {
	save := widget.ToolButton("Save").SetTooltip("Save")
	undo := widget.ToolButton("Undo").SetTooltip("Undo")
	paste := widget.ToolButton("Paste")
	var ribbon *widget.RibbonWidget
	ribbon = widget.Ribbon().
		AddTab("Home").AddGroup("Clipboard").AddLarge(paste).
		AddContextualTab("Picture Tools", "Format").AddGroup("Arrange").AddSmall(widget.ToolButton("Rotate")).
		AddQuickAccess(save, undo).
		OnQuickAccessChange(func() {
			println(len(ribbon.QuickAccess())) // save ribbon.QuickAccess() in the app's settings
		})
	if saved := []string{"Save", "Paste"}; len(saved) > 0 { // restored from the settings, when the user changed them
		ribbon.SetQuickAccess(saved...)
	}
	picture := widget.CheckBox("Picture selected")
	picture.OnChange(func() { ribbon.SetCategoryVisible("Picture Tools", picture.Checked()) })
}
