package widget

import "github.com/egoist/mygo"

// Filter limits a file dialog to files with one of Extensions (without
// the dot, as "png"), listed as Name.
type Filter = mygo.FileFilter

// waitFor runs fn, a call that blocks until the user answers, on a
// goroutine of its own, and waits for it as Dialog.Run waits: pumping
// the handler queue on the event goroutine, panicking on the main thread.
func waitFor[T any](name string, fn func() T) T {
	notMain(name)
	var v T
	done := make(chan struct{})
	go func() {
		defer close(done)
		v = fn()
	}()
	block(name, done)
	return v
}

// OpenFile asks for a file to open, over parent (nil for none), and
// returns its path, "" if the user cancels.
func OpenFile(parent *Window, title string, filters ...Filter) string {
	return openPath("OpenFile", mygo.OpenDialogOptions{Parent: parent.native(), Title: title, Filters: filters})
}

// OpenFiles asks for files to open and returns their paths, nil if the
// user cancels.
func OpenFiles(parent *Window, title string, filters ...Filter) []string {
	return openPaths("OpenFiles", mygo.OpenDialogOptions{Parent: parent.native(), Title: title, Filters: filters, Multiple: true})
}

// SelectFolder asks for a folder, starting in dir, and returns its path,
// "" if the user cancels.
func SelectFolder(parent *Window, title, dir string) string {
	return openPath("SelectFolder", mygo.OpenDialogOptions{Parent: parent.native(), Title: title, DefaultPath: dir, Directory: true})
}

// SaveFile asks where to save a file, suggesting name, and returns the
// path, "" if the user cancels.
func SaveFile(parent *Window, title, name string, filters ...Filter) string {
	return waitFor("SaveFile", func() string {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: parent.native(), Title: title, DefaultPath: name, Filters: filters})
		if err != nil {
			return ""
		}
		return path
	})
}

// openPaths runs an open dialog; an error counts as a cancel.
func openPaths(name string, opts mygo.OpenDialogOptions) []string {
	return waitFor(name, func() []string {
		paths, err := mygo.Dialog.Open(opts)
		if err != nil {
			return nil
		}
		return paths
	})
}

// openPath runs an open dialog for one path, "" for a cancel.
func openPath(name string, opts mygo.OpenDialogOptions) string {
	if paths := openPaths(name, opts); len(paths) > 0 {
		return paths[0]
	}
	return ""
}
