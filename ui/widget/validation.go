package widget

import (
	"regexp"
	"slices"
	"strings"
	"weak"
)

// ruleList is what Rule keeps on a control: its rules, in the order given,
// and what runs them when the focus leaves it.
type ruleList struct {
	list  []rule
	leave func() // posted by render when the focus leaves the control
}

// rule is one check of a control's value, for the provider p.
type rule struct {
	p     *ErrorProviderWidget
	check func() string // the error text of the control's value, "" for valid; takes mu
}

// formRules runs the rules of the controls cs for ValidateChildren, set by
// Rule: an app that makes no rule does not link them. Under mu.
var formRules func(cs []*core) string

// Rule gives c the check check, for p: Validate, Form.ValidateChildren, and
// with SetValidateOnLeave the focus leaving c, run it on c's value and mark
// c with its text, or clear p's mark for "". It returns c, so calls chain.
// A control may be given several rules, by several providers too: for each
// provider, its first failing rule's text shows. Rule is safe from any
// goroutine; check runs on the goroutine that validates, with no lock held.
//
//	name := widget.Rule(errors, widget.TextBox(), widget.Required("Name is required"))
func Rule[C Bindable[T], T comparable](p *ErrorProviderWidget, c C, check func(v T) string) C {
	addRule(p, c.base(), func() string { return check(c.bindValue()) })
	return c
}

// addRule is Rule's work, apart from the value's type.
func addRule(p *ErrorProviderWidget, b *core, check func() string) {
	b.change(func() {
		formRules = runFormRules
		if b.rules == nil {
			b.rules = &ruleList{leave: func() { leaveRules(b) }}
		}
		if !slices.ContainsFunc(b.rules.list, func(r rule) bool { return r.p == p }) {
			p.ruled = append(p.ruled, weak.Make(b))
		}
		b.rules.list = append(b.rules.list, rule{p, check})
	})
}

// Required is a rule for Rule that fails with msg on text that is empty
// once spaces are trimmed.
func Required(msg string) func(string) string {
	return func(v string) string {
		if strings.TrimSpace(v) == "" {
			return msg
		}
		return ""
	}
}

// MatchRegexp is a rule for Rule that fails with msg on text re does not
// match.
func MatchRegexp(re *regexp.Regexp, msg string) func(string) string {
	return func(v string) string {
		if !re.MatchString(v) {
			return msg
		}
		return ""
	}
}

// InRange is a rule for Rule that fails with msg on a number below min or
// above max; both bounds are in the range.
func InRange(min, max float64, msg string) func(float64) string {
	return func(v float64) string {
		if v < min || v > max {
			return msg
		}
		return ""
	}
}

// Validate runs every rule given with Rule for p, sets or clears the mark of
// each control they check, and reports whether none failed. It does not move
// the focus; FocusFirst does. Call it from the event goroutine (a button's
// handler) or any other, not from a view.
func (p *ErrorProviderWidget) Validate() bool {
	mu.Lock()
	cs := make([]*core, 0, len(p.ruled))
	live := p.ruled[:0]
	for _, w := range p.ruled {
		if b := w.Value(); b != nil {
			cs, live = append(cs, b), append(live, w)
		}
	}
	clear(p.ruled[len(live):])
	p.ruled = live
	mu.Unlock()
	return checkRules(cs, func(q *ErrorProviderWidget) bool { return q == p }) == ""
}

// SetValidateOnLeave makes the focus leaving a control run its rules for p,
// after its OnFocusOut, as Validate runs them; off, the default, only
// Validate and Form.ValidateChildren run them.
func (p *ErrorProviderWidget) SetValidateOnLeave(on bool) *ErrorProviderWidget {
	mu.Lock()
	p.onLeave = on
	mu.Unlock()
	return p
}

// runFormRules is formRules.
func runFormRules(cs []*core) string { return checkRules(cs, nil) }

// leaveRules runs b's rules of the providers that validate on leave.
func leaveRules(b *core) {
	checkRules([]*core{b}, func(p *ErrorProviderWidget) bool { return p.onLeave })
}

// checkRules runs the rules of cs whose provider keep accepts (every one
// for nil), and marks each control for each provider with its first failing
// rule's text, or clears the mark. It returns the first failing text, "" when
// none failed. keep runs under mu; the checks and the marks off it.
func checkRules(cs []*core, keep func(*ErrorProviderWidget) bool) string {
	type key struct {
		b *core
		p *ErrorProviderWidget
	}
	type job struct {
		key
		check func() string
	}
	mu.Lock()
	var jobs []job
	for _, b := range cs { // each has rules
		for _, r := range b.rules.list {
			if keep == nil || keep(r.p) {
				jobs = append(jobs, job{key{b, r.p}, r.check})
			}
		}
	}
	mu.Unlock()
	var order []key
	msgs := map[key]string{}
	for _, j := range jobs {
		msg, seen := msgs[j.key]
		if !seen {
			order = append(order, j.key)
		}
		if msg == "" {
			msgs[j.key] = j.check()
		}
	}
	first := ""
	for _, k := range order {
		msg := msgs[k]
		k.p.setError(k.b, msg)
		if first == "" {
			first = msg
		}
	}
	return first
}
