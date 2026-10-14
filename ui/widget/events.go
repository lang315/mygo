package widget

// Post is a stand-in until the event goroutine lands.
func Post(fn func()) { go fn() }
