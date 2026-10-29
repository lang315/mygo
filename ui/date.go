package ui

import (
	"time"

	"github.com/egoist/mygo/i18n"
)

// DateInput creates a field showing *date in the user's regional formats,
// as 1/5/2026 in the United States, which a calendar below it changes. A click, Enter or Space opens the calendar, where a
// click chooses a day, as do the arrows and Enter; Page Up and Page Down,
// or its buttons, move by months, and Escape closes it. Changed reports a
// new date, which keeps the time of day and location of *date.
func coreDateInput(c *context, date *time.Time) *node {
	t := c.theme
	_, locale := c.formats()
	text := c.formatted(fmtKey{locale, fmtShortDate, dayNumber(*date)}, func() string {
		return i18n.FormatDate(locale, *date, i18n.DateOptions{Skeleton: "yMd"})
	})
	b := coreButton(c, "")
	b.widget, b.role, b.accValue = "DateInput", RolePopUpButton, text
	b.MinWidth(t.Space(32.5)).Justify(SpaceBetween)
	open := coreLocal(b, popupOpen{}, func() bool { return false })
	// cursor is the day the keys move in the calendar.
	cursor := coreLocal(b, "cursor", func() time.Time { return *date })
	b.afterInput(func() {
		if b.Clicked() {
			*open = !*open
			*cursor = *date
		}
		b.expanded = *open
	})
	b.Children(func() {
		coreText(c, text).SingleLine().FontFeatures("tnum")
		coreBox(c).Size(t.Space(3.5), t.Space(3.5)).Shrink(0).Draw(func(p *Painter, r Rect) {
			// A calendar page.
			p.Stroke(Rect{r.X + 1, r.Y + 2, r.W - 2, r.H - 3}, t.TextMuted, 2, 1.2)
			p.Fill(Rect{r.X + 1, r.Y + 5, r.W - 2, 1.2}, t.TextMuted, 0)
		})
	})
	choose := func(d time.Time) {
		if setDay(date, d) {
			b.st.markChanged()
			c.rt.consumed = true
		}
		*open = false
		b.Focus()
	}
	// The calendar keeps its own width under a wide field: not Popover,
	// whose panel is as wide as the field at least.
	corePopoverBase(c, b, open, func(panel *node) {
		stylePanel(c, panel)
		calendarGrid(c, calendarBox(c), date, cursor, false, choose).AutoFocus()
	})
	return b
}

// Calendar creates a month's calendar choosing *date, as SwiftUI's
// graphical date picker, in the window's language, its weeks starting on
// the first day of the user's regional formats (Context.Region): a click chooses a day, as the arrows do while the
// calendar has the focus, Page Up and Page Down or its buttons move by
// months, and Home and End go to the first and the last day of the month.
// Changed reports a new date, which keeps the time of day and location of
// *date. Assistive technology reads the day chosen as the arrows move.
// Browse makes the keys and the buttons move without choosing.
func coreCalendar(c *context, date *time.Time) *node {
	g := calendarBox(c)
	g.widget = "Calendar"
	// Input reaches the calendar as its constructor returns, before this
	// build's Browse: it follows the build before, which the user saw.
	set := coreLocal(g, calendarKey{}, func() calendarSettings { return calendarSettings{} })
	set.shown, set.now = set.now, false
	// The cursor follows the date the app sets.
	if !set.shown || !sameDay(set.date, *date) {
		set.cursor = *date
	}
	set.date = *date
	cursor := date
	if set.shown {
		cursor = &set.cursor
	}
	calendarGrid(c, g, date, cursor, !set.shown, func(d time.Time) {
		if setDay(date, d) {
			g.st.markChanged()
			c.rt.consumed = true
		}
	})
	return g
}

type calendarKey struct{}

// calendarSettings are a Calendar's: whether it browses, in this build
// and in the one before, its cursor while it does, and the date the
// cursor followed last.
type calendarSettings struct {
	now, shown   bool
	cursor, date time.Time
}

// Browse lets the user move about a Calendar without choosing a day: its
// buttons and Page Up and Page Down only show other months, and the
// arrows, Home and End only move among the days, until a click, Enter or
// Space chooses one, as in a date input's calendar. Changed then reports
// only a day chosen, as a calendar in a popover that closes when the
// user picks a day needs. The calendar goes back to *date as the app
// sets another.
func (e *node) Browse() *node {
	if e.widget == "Calendar" {
		coreLocal(e, calendarKey{}, func() calendarSettings { return calendarSettings{} }).now = true
	}
	return e
}

// monthChevron draws the arrow of a calendar's button to the month before,
// back, or after, mirrored right to left: drawn, it sits in the middle of
// the button, where the text's ‹ and › sat low.
func monthChevron(c *context, back bool) {
	t := c.theme
	e := coreBox(c).Size(t.Space(1.5), t.Space(2.5)).Shrink(0)
	tip, ends := float32(0.15), float32(0.85) // ‹
	if back == e.rtl() {
		tip, ends = ends, tip // ›
	}
	e.Draw(func(p *Painter, r Rect) {
		var path Path
		path.MoveTo(r.X+r.W*ends, r.Y+r.H*0.1).LineTo(r.X+r.W*tip, r.Y+r.H*0.5).LineTo(r.X+r.W*ends, r.Y+r.H*0.9)
		p.StrokePath(&path, 1.5, t.Text)
	})
}

// setDay sets the day of *date to d's, keeping its time of day and its
// location, and reports whether it changed.
func setDay(date *time.Time, d time.Time) bool {
	y, m, day := d.Date()
	h, mi, s := date.Clock()
	next := time.Date(y, m, day, h, mi, s, date.Nanosecond(), date.Location())
	if next.Equal(*date) {
		return false
	}
	*date = next
	return true
}

// sameDay reports whether a and b fall on the same day.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// addMonths returns t n months on, at the same day of the month or the
// last one of a month that is shorter, so that paging from the 31st
// neither skips a month nor stays in one.
func addMonths(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, t.Location())
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
	return time.Date(first.Year(), first.Month(), min(d, last), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// calendarBox creates the element of a month's grid, which calendarGrid
// fills.
func calendarBox(c *context) *node {
	grid := coreColumn(c).Gap(c.theme.Space(0.5)).Focusable().Shrink(0).Role(RoleTable)
	grid.flags |= flagOwnRing
	return grid
}

// calendarGrid fills grid, from calendarBox, with the month around
// *cursor, the day the keys move, with *date chosen; choose chooses a
// day, as a click and Enter do, and every move does with moveChooses.
// Where moves choose, paging by months moves what the grid shows alone, and
// the day chosen stays.
func calendarGrid(c *context, grid *node, date, cursor *time.Time, moveChooses bool, choose func(time.Time)) *node {
	t := c.theme
	region, locale := c.formats()
	// view is the month shown. Where moves choose, it is its own: it
	// follows *date when that changes, and paging moves it.
	view := coreLocal(grid, "view", func() time.Time { return *cursor })
	if moveChooses {
		seen := coreLocal(grid, "seen", func() time.Time { return *date })
		if !sameDay(*seen, *date) {
			*seen, *view = *date, *date
		}
	} else {
		*view = *cursor
	}
	cur := *cursor
	// kc is the day the keys act on: the cursor, or where another month is
	// shown, the same day of the month there, clamped.
	kc := cur
	if moveChooses && (view.Year() != cur.Year() || view.Month() != cur.Month()) {
		kc = addMonths(cur, (view.Year()-cur.Year())*12+int(view.Month())-int(cur.Month()))
	}
	month := time.Date(view.Year(), view.Month(), 1, 0, 0, 0, 0, time.UTC)
	title := c.formatted(fmtKey{locale, fmtMonthYear, int64(month.Year())*12 + int64(month.Month())}, func() string {
		return i18n.FormatDate(locale, month, i18n.DateOptions{Skeleton: "yMMMM"})
	})
	grid.Label(title)
	move := func(d time.Time) {
		// Choosing it moves the cursor where it is the date chosen.
		if moveChooses {
			choose(d)
			*view = d
		} else {
			*cursor = d
		}
		c.rt.consumed = true
	}
	page := func(n int) {
		if moveChooses {
			*view = addMonths(month, n)
		} else {
			*cursor = addMonths(cur, n)
		}
		c.rt.consumed = true
	}
	grid.afterInput(func() {
		// The days of a week go from the right right to left.
		back, forth := grid.arrows()
		switch {
		case grid.Shortcut(0, back):
			move(kc.AddDate(0, 0, -1))
		case grid.Shortcut(0, forth):
			move(kc.AddDate(0, 0, 1))
		case grid.Shortcut(0, KeyUp):
			move(kc.AddDate(0, 0, -7))
		case grid.Shortcut(0, KeyDown):
			move(kc.AddDate(0, 0, 7))
		case grid.Shortcut(0, KeyPageUp):
			move(addMonths(kc, -1))
		case grid.Shortcut(0, KeyPageDown):
			move(addMonths(kc, 1))
		case grid.Shortcut(0, KeyHome):
			move(time.Date(kc.Year(), kc.Month(), 1, 0, 0, 0, 0, kc.Location()))
		case grid.Shortcut(0, KeyEnd):
			move(time.Date(kc.Year(), kc.Month()+1, 0, 0, 0, 0, 0, kc.Location()))
		case grid.Shortcut(0, KeyEnter), grid.Shortcut(0, KeySpace):
			choose(kc)
		}
	})
	grid.Children(func() {
		coreRow(c).AlignItems(Center).Gap(t.Space(1)).Children(func() {
			prev := coreButton(c, "").Padding(t.Space(1), t.Space(2.5)).Label(c.texts().PreviousMonth())
			prev.Children(func() { monthChevron(c, true) })
			prev.flags |= flagClickable
			prev.afterInput(func() {
				if prev.Clicked() {
					page(-1)
				}
			})
			coreText(c, title).Bold().Grow(1).TextAlign(Center)
			next := coreButton(c, "").Padding(t.Space(1), t.Space(2.5)).Label(c.texts().NextMonth())
			next.Children(func() { monthChevron(c, false) })
			next.flags |= flagClickable
			next.afterInput(func() {
				if next.Clicked() {
					page(1)
				}
			})
		})
		names := i18n.Weekdays(locale, i18n.Short, true)
		first := region.FirstWeekday
		coreRow(c).Children(func() {
			for i := range 7 {
				d := names[(int(first)+i)%7]
				coreText(c, d).Width(t.Space(8)).TextAlign(Center).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Role(RoleColumnHeader)
			}
		})
		// Six weeks from the first day of the week on or before the first.
		start := month.AddDate(0, 0, -((int(month.Weekday()) - int(first) + 7) % 7))
		today := c.now
		for w := range 6 {
			coreRow(c).Role(RoleRow).Children(func() {
				for d := range 7 {
					day := start.AddDate(0, 0, w*7+d)
					label := c.formatted(fmtKey{locale, fmtDay, dayNumber(day)}, func() string {
						return i18n.FormatDate(locale, day, i18n.DateOptions{Skeleton: "yMMMMd"})
					})
					cell := coreBox(c).Size(t.Space(8), t.Space(7)).Center().Radius(t.Radius).Role(RoleButton).Label(label)
					cell.flags |= flagClickable | flagHover
					cell.afterInput(func() {
						if cell.Clicked() {
							choose(day)
							if cursor != date {
								*cursor = day
							}
						}
					})
					chosen, atCursor := sameDay(day, *date), sameDay(day, kc)
					fg := t.Text
					if day.Month() != month.Month() {
						fg = t.TextMuted
					}
					switch {
					case chosen:
						cell.Background(t.Accent)
						fg = t.AccentText
					case atCursor:
						cell.Background(t.SurfaceHover)
					}
					if sameDay(day, today) && !chosen {
						cell.Border(1, t.Accent)
					}
					if !chosen {
						cell.builtWith(t)
						setStyle(cell, hoverSurface, staticHoverSurface)
					}
					cell.checked = 1 + int8(b2f(chosen))
					if atCursor {
						// Assistive technology reads the day the keys are on.
						grid.activeDescendant = cell
					}
					cell.Children(func() {
						coreText(c, c.formatNumber(locale, day.Day(), 1)).TextColor(fg).FontFeatures("tnum")
					})
				}
			})
		}
	})
	grid.DrawOver(func(p *Painter, r Rect) {
		if grid.FocusVisible() {
			p.FocusRing(r, [4]float32{t.Radius, t.Radius, t.Radius, t.Radius})
		}
	})
	return grid
}
