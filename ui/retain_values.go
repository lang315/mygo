package ui

// valueRead is a value a control showed in a row as it was built, and where it
// came from: the *bool of a Checkbox or Toggle, the *string of a Select, the
// *float64 of a Slider. An app
// may change it without input (a "select all" outside the list) and without
// saying so in RowVersion: a row is kept only while each holds what the row was
// built with, so the control versions itself. (By address: see the doc of
// RowVersion for data that moves.)
type valueRead struct {
	p  *bool
	v  bool
	s  *string
	sv string
	f  *float64
	fv float64
}

// maxValueReads is how many different controls a row records; more, and the row
// is built in every frame.
const maxValueReads = 8

// readsBool notes the *bool of a control the row being built made.
func (c *context) readsBool(p *bool, v bool) {
	if c != nil && c.row != nil {
		c.row.noteBool(p, v)
	}
}

// noteBool records p once. A control read twice with two values in one build, or
// more controls than the limit, make the row (and the rows around it) built in
// every frame.
func (rb *rowBuild) noteBool(p *bool, v bool) {
	for _, b := range rb.values {
		if b.p == p {
			if b.v != v {
				rb.neverKept()
			}
			return
		}
	}
	if len(rb.values) >= maxValueReads {
		rb.neverKept()
		return
	}
	if rb.values == nil && rb.f != nil {
		rb.values = rb.f.valueBuf[:0]
	}
	rb.values = append(rb.values, valueRead{p: p, v: v})
}

// neverKept marks the row, and the rows around it, built in every frame.
func (rb *rowBuild) neverKept() {
	for r := rb; r != nil; r = r.outer {
		r.outside = true
	}
}

// sameValues reports whether every *bool holds what the row was built with.
func sameValues(bs []valueRead) bool {
	for _, b := range bs {
		if b.p != nil && *b.p != b.v || b.s != nil && *b.s != b.sv || b.f != nil && *b.f != b.fv {
			return false
		}
	}
	return true
}

// readsString notes the *string a control (Select) made the row show.
func (c *context) readsString(s *string, v string) {
	if c != nil && c.row != nil {
		c.row.noteString(s, v)
	}
}

// noteString is noteBool for a string.
func (rb *rowBuild) noteString(s *string, v string) {
	for _, b := range rb.values {
		if b.s == s && s != nil {
			if b.sv != v {
				rb.neverKept()
			}
			return
		}
	}
	if len(rb.values) >= maxValueReads {
		rb.neverKept()
		return
	}
	if rb.values == nil && rb.f != nil {
		rb.values = rb.f.valueBuf[:0]
	}
	rb.values = append(rb.values, valueRead{s: s, sv: v})
}

// readsFloat notes the *float64 a control (Slider) made the row show.
func (c *context) readsFloat(f *float64, v float64) {
	if c != nil && c.row != nil {
		c.row.noteFloat(f, v)
	}
}

// noteFloat is noteBool for a number. A NaN is never equal to itself, so a row
// that shows one is built in every frame.
func (rb *rowBuild) noteFloat(f *float64, v float64) {
	for _, b := range rb.values {
		if b.f == f && f != nil {
			if b.fv != v {
				rb.neverKept()
			}
			return
		}
	}
	if len(rb.values) >= maxValueReads {
		rb.neverKept()
		return
	}
	if rb.values == nil && rb.f != nil {
		rb.values = rb.f.valueBuf[:0]
	}
	rb.values = append(rb.values, valueRead{f: f, fv: v})
}
