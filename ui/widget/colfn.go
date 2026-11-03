package widget

import "runtime/debug"

// runColumnFn runs f, one of an app's column functions (Validate, Set,
// CanEdit, EditText), and returns what it panicked with and the stack at the
// panic. The recovery is this function's own defer, so a caller that holds the
// package's lock gets control back and can unlock before it reports the panic
// (report): the app's OnPanic handler may call the widgets.
func runColumnFn(f func()) (v any, stack []byte, panicked bool) {
	defer func() {
		if v = recover(); v != nil {
			stack, panicked = debug.Stack(), true
		}
	}()
	f()
	return nil, nil, false
}
