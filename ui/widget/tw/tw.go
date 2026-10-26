// Package tw makes widget Styles of Tailwind classes:
//
//	widget.Button("Save").Apply(tw.S("px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover transition"))
//
// Values are Tailwind v3's; hover:, active:, focus:, disabled: and dark:
// apply a class in a state, and dark: may stack on one of the others. On
// the same property the later class wins. gota's halign-*, valign-*,
// hexpand, vexpand, suggested-action and destructive-action work too. A
// class tw does not know is an error for Parse, and mygo vet reports it
// in a literal string.
package tw

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/internal/twclass"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/twsource"
)

// Bad is a class Parse could not use, and why.
type Bad struct{ Class, Reason string }

// Error lists the bad classes of a string.
type Error struct{ Bad []Bad }

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("tw: ")
	for i, x := range e.Bad {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(x.Reason + " " + strconv.Quote(x.Class))
	}
	return b.String()
}

// maxEntries bounds the cache and the classes logged, so generated
// strings cannot grow them without end.
const maxEntries = 4096

var (
	cacheMu sync.RWMutex
	cache   = map[string]widget.Style{}
	logged  = map[Bad]bool{}
)

// S returns the Style the classes make, parsed once and cached. A bad
// class is logged once and ignored; the others still apply. Safe from
// any goroutine.
func S(classes string) widget.Style {
	cacheMu.RLock()
	s, ok := cache[classes]
	cacheMu.RUnlock()
	if ok {
		return s
	}
	rules, bad := twclass.Parse(classes)
	s = withClasses(build(rules), classes) // once: the cache keeps it, for the inspector
	// With the cache full and nothing new to log, take no write lock: a miss
	// would hold up the hits.
	cacheMu.RLock()
	store, say := len(cache) < maxEntries, unlogged(bad)
	cacheMu.RUnlock()
	if !store && !say {
		return s
	}
	var lines []Bad
	cacheMu.Lock()
	if len(cache) < maxEntries {
		cache[classes] = s
	}
	for _, b := range bad {
		k := Bad(b)
		if !logged[k] && len(logged) < maxEntries {
			logged[k] = true
			lines = append(lines, k)
		}
	}
	cacheMu.Unlock()
	for _, k := range lines { // not under the lock: a slow log must not stall the hits
		log.Printf("tw: %s %q", k.Reason, k.Class)
	}
	return s
}

// unlogged reports whether bad has a class to log, as S would. Under cacheMu.
func unlogged(bad []twclass.Bad) bool {
	for _, b := range bad {
		if !logged[Bad(b)] && len(logged) < maxEntries {
			return true
		}
	}
	return false
}

// withClasses notes on s the classes that made it, for the inspector.
func withClasses(s widget.Style, classes string) widget.Style {
	twsource.Note(&s, classes)
	return s
}

// Parse is S without the cache, returning an *Error naming every bad
// class.
func Parse(classes string) (widget.Style, error) {
	rules, bad := twclass.Parse(classes)
	s := withClasses(build(rules), classes)
	if len(bad) == 0 {
		return s, nil
	}
	e := &Error{Bad: make([]Bad, len(bad))}
	for i, b := range bad {
		e.Bad[i] = Bad(b)
	}
	return s, e
}

// draft is the Style of one variant, light or dark, being built.
type draft struct {
	s            widget.Style
	italic, used bool
}

func (d *draft) style() widget.Style {
	if d.italic {
		return d.s.Italic()
	}
	return d.s
}

var variantOf = [...]func(widget.Style, widget.Style) widget.Style{
	twclass.Hover:    widget.Style.Hover,
	twclass.Active:   widget.Style.Pressed,
	twclass.Focus:    widget.Style.Focused,
	twclass.Disabled: widget.Style.Disabled,
}

// build makes the Style of rules: each variant's declarations in order,
// a later one winning.
func build(rules []twclass.Rule) widget.Style {
	var d [twclass.Disabled + 1][2]draft
	for _, r := range rules {
		dark := 0
		if r.Key.Dark {
			dark = 1
		}
		x := &d[r.Key.Variant][dark]
		x.used = true
		x.add(r.Decl)
	}
	s := d[twclass.Base][0].style()
	if d[twclass.Base][1].used {
		s = s.Dark(d[twclass.Base][1].style())
	}
	for v := twclass.Hover; v <= twclass.Disabled; v++ {
		light, dark := &d[v][0], &d[v][1]
		if !light.used && !dark.used {
			continue
		}
		vs := light.style()
		if dark.used {
			vs = vs.Dark(dark.style())
		}
		s = variantOf[v](s, vs)
	}
	return s
}

var (
	padSides    = [4]func(widget.Style, float32) widget.Style{widget.Style.PaddingTop, widget.Style.PaddingRight, widget.Style.PaddingBottom, widget.Style.PaddingLeft}
	marginSides = [4]func(widget.Style, float32) widget.Style{widget.Style.MarginTop, widget.Style.MarginRight, widget.Style.MarginBottom, widget.Style.MarginLeft}
	corners     = [4]func(widget.Style, float32) widget.Style{widget.Style.RadiusTopLeft, widget.Style.RadiusTopRight, widget.Style.RadiusBottomRight, widget.Style.RadiusBottomLeft}
	borderSides = [4]func(widget.Style, float32) widget.Style{widget.Style.BorderTop, widget.Style.BorderRight, widget.Style.BorderBottom, widget.Style.BorderLeft}
)

// four sets v on the sides of mask: all four side by side equal the typed
// Padding(v) and the others.
func four(s widget.Style, mask uint8, v float32, side [4]func(widget.Style, float32) widget.Style) widget.Style {
	for i := range 4 {
		if mask&(1<<i) != 0 {
			s = side[i](s, v)
		}
	}
	return s
}

func (d *draft) add(c twclass.Decl) {
	s := d.s
	switch c.Kind {
	case twclass.Padding:
		s = four(s, c.Sides, c.Value, padSides)
	case twclass.Margin:
		s = four(s, c.Sides, c.Value, marginSides)
	case twclass.Radius:
		s = four(s, c.Sides, c.Value, corners)
	case twclass.BorderWidth:
		s = four(s, c.Sides, c.Value, borderSides)
	case twclass.BorderStyle:
		b := ui.BorderSolid
		if c.Value == 1 {
			b = ui.BorderDashed
		}
		s = s.BorderStyle(b)
	case twclass.Width:
		s = s.Width(c.Value)
	case twclass.Height:
		s = s.Height(c.Value)
	case twclass.MinWidth:
		s = s.MinWidth(c.Value)
	case twclass.MaxWidth:
		s = s.MaxWidth(c.Value)
	case twclass.MinHeight:
		s = s.MinHeight(c.Value)
	case twclass.MaxHeight:
		s = s.MaxHeight(c.Value)
	case twclass.Background:
		s = s.Background(colorOf(c.Color))
	case twclass.TextColor:
		s = s.TextColor(colorOf(c.Color))
	case twclass.BorderColor:
		s = s.BorderColor(colorOf(c.Color))
	case twclass.TextSize:
		s = s.TextSize(widget.Size(c.Value))
	case twclass.FontSize:
		s = s.FontSize(c.Value)
	case twclass.FontWeight:
		s = s.FontWeight(int(c.Value))
	case twclass.Italic:
		d.italic = c.Value == 1
	case twclass.Mono:
		s = s.Mono()
	case twclass.Opacity:
		s = s.Opacity(c.Value)
	case twclass.Cursor:
		s = s.Cursor(cursors[c.Name])
	case twclass.Shadows:
		s = s.NoShadow()
		for _, x := range c.Shadows {
			col := widget.Fixed(ui.RGBA(0, 0, 0, x.Alpha))
			if x.Inset {
				s = s.InsetShadow(x.X, x.Y, x.Blur, x.Spread, col)
			} else {
				s = s.Shadow(x.X, x.Y, x.Blur, x.Spread, col)
			}
		}
	case twclass.Align:
		s = s.Align(widget.Align(c.Value))
	case twclass.VAlign:
		s = s.VAlign(widget.Align(c.Value))
	case twclass.HExpand:
		s = s.HExpand()
	case twclass.VExpand:
		s = s.VExpand()
	case twclass.Transition:
		s = s.Transition(time.Duration(c.Value) * time.Millisecond)
	}
	d.s = s
}

func colorOf(c twclass.Color) widget.Color {
	if c.Role != "" {
		return roles[c.Role]
	}
	return widget.Fixed(ui.Color{R: uint8(c.RGBA >> 24), G: uint8(c.RGBA >> 16), B: uint8(c.RGBA >> 8), A: uint8(c.RGBA)})
}

var roles = map[string]widget.Color{
	"background": widget.RoleBackground, "surface": widget.RoleSurface,
	"surface-hover": widget.RoleSurfaceHover, "surface-pressed": widget.RoleSurfacePressed,
	"border": widget.RoleBorder, "text": widget.RoleText, "text-muted": widget.RoleTextMuted,
	"accent": widget.RoleAccent, "accent-hover": widget.RoleAccentHover,
	"accent-pressed": widget.RoleAccentPressed, "accent-text": widget.RoleAccentText,
	"danger": widget.RoleDanger, "warning": widget.RoleWarning, "success": widget.RoleSuccess,
	"selection": widget.RoleSelection, "focus": widget.RoleFocus, "inverse": widget.RoleInverse,
	"inverse-text": widget.RoleInverseText,
}

var cursors = map[string]ui.Cursor{
	"default": ui.CursorDefault, "pointer": ui.CursorPointer, "text": ui.CursorText,
	"move": ui.CursorMove, "not-allowed": ui.CursorNotAllowed, "grab": ui.CursorGrab,
	"grabbing": ui.CursorGrabbing, "crosshair": ui.CursorCrosshair,
}
