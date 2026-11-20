// Package waitcheck holds the test that tests do not wait on the wall
// clock ("Waiting in tests" in docs/architecture.md): no sleeps, timers,
// deadlines or timeouts in a _test.go file outside a benchmark. A test
// waits for the condition or the event with internal/wait, or advances a
// ui.Tester's clock. The test reads the source whatever its build
// constraints, so it runs on any host.
//
// The waits on the clock the tests had when the check came are counted in
// baseline, by "file function call": each key must have exactly as many
// as it says, so that a new wait, a wait of another kind or in another
// function fails, and a removed one lowers the count. A wait that is right
// goes in allowed, with why, and an entry there that matches no call
// fails.
package waitcheck
