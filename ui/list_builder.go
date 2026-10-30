package ui

// ListRow gives a row its shared context, index, selection and list focus.
type ListRow struct {
	Context  *Context
	Index    int
	owner    Element
	selected bool
}

// Selected reports whether the row is chosen. It finds the list through lookup,
// not Valid: the list is outside the row, and Valid notes that for the row being
// built, which would then be built in every frame (the choice is a key of a kept
// row already, rowKeep.chosen).
func (r ListRow) Selected() bool { return r.owner.lookup() != nil && r.selected }

// ListFocused reports whether the list has the focus within it. The list is
// built in the pass of its rows, so the row records the answer it got and is kept
// only while the list's focus is the same (rowKeep.listFocus), not built in every
// frame for asking of an element outside it.
func (r ListRow) ListFocused() bool {
	n := r.owner.lookup()
	if n == nil {
		noteMissingElement()
		return false
	}
	v := n.focusWithin()
	if rb := n.c.row; rb != nil {
		rb.noteListFocus(v)
	}
	return v
}

type listSpec struct {
	state *ListState
	count int
	built bool
}

// List creates a virtual list. The original row callback remains supported.
// Omit it to configure ItemKey and Selection, then call Rows.
func List(c *Context, s *ListState, n int, row ...func(int)) Element {
	raw := c.build()
	if raw == nil {
		return Element{}
	}
	if len(row) > 1 {
		panic("ui: List accepts one row builder")
	}
	if len(row) == 1 {
		// List is a public constructor, as the generated ones: its input
		// reaches Changed as it returns, also in a container still
		// building, as a split's pane.
		raw.rt.beginBuild()
		e := coreList(raw, s, n, row[0])
		raw.rt.endBuild()
		return wrapElement(e)
	}
	e := coreScroll(raw)
	e.widget, e.role = "List", RoleList
	if s == nil {
		s = coreLocal(e, "list", func() ListState { return ListState{} })
	}
	*valueBinding[listSpec](e) = listSpec{state: s, count: n}
	return wrapElement(e)
}
func (e Element) ItemKey(key func(int) any) Element {
	if n := e.node(); n != nil {
		if s, ok := n.st.valueBinding.(*listSpec); ok {
			s.state.Key = key
		}
	}
	return e
}
func (e Element) Selection(selection Selector) Element {
	if n := e.node(); n != nil {
		if s, ok := n.st.valueBinding.(*listSpec); ok {
			s.state.Selection = selection
		}
	}
	return e
}
func (e Element) Rows(row func(ListRow)) Element {
	n := e.node()
	if n == nil {
		return e
	}
	s, ok := n.st.valueBinding.(*listSpec)
	if !ok {
		return e
	}
	if s.built {
		panic("ui: List.Rows called twice in one pass")
	}
	s.built = true
	// Rows builds the list: it is the constructor, as List with a row.
	n.c.rt.beginBuild()
	buildList(n.c, n, n, s.state, s.count, func(i int) {
		if row == nil {
			return
		}
		selected := false
		if n.c.row != nil {
			selected = n.c.row.chosen
		}
		row(ListRow{Context: makeContext(n.c), Index: i, owner: e, selected: selected})
	}, plainList)
	n.c.rt.endBuild()
	return e
}
