package skin

import (
	"strings"

	"github.com/egoist/mygo/ui/widget/internal/skinhook"
	"github.com/egoist/mygo/ui/widget/tw"
)

// Looks are the looks a skin gives each kind of control, and two parts of
// controls, as Tailwind classes (package tw): a look sets the shape, not
// only the color, so two skins can differ as much as their controls do. ""
// leaves a kind its plain look.
//
// A control's own methods and its Apply'd styles win over its look, and the
// look wins over the rest of the skin. A look does not place a control:
// its halign-*, hexpand and the like are ignored. A look may name the
// skin's colors by role (bg-chrome, border-control, …) and its states with
// hover:, focus:, active:, disabled:, dark: and selected: (a CheckBox or a
// RadioButton checked, a toggle ToolButton on, the open tab).
type Looks struct {
	// Button is a Button's face, ToolButton a ToolStrip's buttons'.
	Button, ToolButton string
	// Field is the box of the controls typed into: TextBox,
	// MultilineTextBox, MaskedTextBox, NumericUpDown, DomainUpDown,
	// DateTimePicker and an editable ComboBox. ComboBox is a drop-down
	// ComboBox's.
	Field, ComboBox string
	// CheckBox and RadioButton are the whole control, with its text.
	CheckBox, RadioButton string
	// GroupBox and Card are the frames of the controls so named.
	GroupBox, Card string
	// TabStrip is the strip of a TabControl's tabs, under its StripStyle;
	// TabHeader each tab, selected: the open one. With a TabHeader look the
	// open tab has no line under it, unless the look's border draws one.
	TabStrip, TabHeader string
	// GridHeader is each column header cell of a DataGridView.
	GridHeader string
	// ToolStrip, StatusStrip and MenuBar are the strips so named, HeaderBar
	// a Form's header bar.
	ToolStrip, StatusStrip, MenuBar, HeaderBar string
}

// WithLooks returns a copy of the skin whose looks are l: each kind of l
// replaces the skin's look of that kind whole, and its looks of kinds l
// leaves "" go. A class tw does not know is logged once and skipped, as
// tw.S does; the rest of the look applies.
//
//	boxy := skin.Calm.WithLooks(skin.Looks{
//		Button:    "rounded-none border border-control",
//		TabHeader: "rounded-t border border-b-0 selected:bg-background",
//	})
//	skin.Use(boxy)
func (s *Skin) WithLooks(l Looks) *Skin {
	c := *s
	c.looks, c.parsed = l, parseLooks(l)
	return &c
}

// Looks returns the skin's looks, as WithLooks set them.
func (s *Skin) Looks() Looks { return s.looks }

// parseLooks returns the Styles of l's classes by kind, each boxed once, as
// skinhook.SetLooks takes them; nil when l sets none.
func parseLooks(l Looks) *[skinhook.NumKinds]any {
	classes := [skinhook.NumKinds]string{
		skinhook.KindButton: l.Button, skinhook.KindToolButton: l.ToolButton,
		skinhook.KindField: l.Field, skinhook.KindComboBox: l.ComboBox,
		skinhook.KindCheckBox: l.CheckBox, skinhook.KindRadioButton: l.RadioButton,
		skinhook.KindGroupBox: l.GroupBox, skinhook.KindCard: l.Card,
		skinhook.KindTabStrip: l.TabStrip, skinhook.KindTabHeader: l.TabHeader,
		skinhook.KindGridHeader: l.GridHeader, skinhook.KindToolStrip: l.ToolStrip,
		skinhook.KindStatusStrip: l.StatusStrip, skinhook.KindMenuBar: l.MenuBar,
		skinhook.KindHeaderBar: l.HeaderBar,
	}
	var out [skinhook.NumKinds]any
	set := false
	for k, cl := range classes {
		if strings.TrimSpace(cl) == "" {
			continue
		}
		st := tw.S(cl)
		out[k], set = &st, true
	}
	if !set {
		return nil
	}
	return &out
}
