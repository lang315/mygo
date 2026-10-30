package ui

// boolRead is the *bool of a control (Checkbox, Toggle, Switch) a row built and
// the value it held then. The control shows its value in the row, and an app may
// change it without input (a "select all" outside the list) and without saying so
// in RowVersion: a row is kept only while each *bool holds what the row was built
// with, so the control versions itself.
type boolRead struct {
	p *bool
	v bool
}

// maxBoolReads is how many different controls a row records; more, and the row
// is built in every frame.
const maxBoolReads = 8

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
	for _, b := range rb.bools {
		if b.p == p {
			if b.v != v {
				rb.neverKept()
			}
			return
		}
	}
	if len(rb.bools) >= maxBoolReads {
		rb.neverKept()
		return
	}
	if rb.bools == nil && rb.f != nil {
		rb.bools = rb.f.boolBuf[:0]
	}
	rb.bools = append(rb.bools, boolRead{p, v})
}

// neverKept marks the row, and the rows around it, built in every frame.
func (rb *rowBuild) neverKept() {
	for r := rb; r != nil; r = r.outer {
		r.outside = true
	}
}

// sameBools reports whether every *bool holds what the row was built with.
func sameBools(bs []boolRead) bool {
	for _, b := range bs {
		if *b.p != b.v {
			return false
		}
	}
	return true
}
