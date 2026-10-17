package mygo

import (
	"os"

	"github.com/egoist/mygo/internal/update"
)

// Windows refuses to replace or delete a file that another program has
// open without sharing its deletion, as Go opens files, and antivirus
// software and indexers do while they scan new files; and it refuses to
// open a file while it is being replaced. The app's own files (window
// states, recovery copies, recent documents) are replaced in place, so
// their renames and reads retry for a few seconds while Windows reports
// them in use, rather than lose what they hold.

// rename renames a file of the app, retrying while Windows reports it in
// use.
func rename(from, to string) error {
	return update.Rename(from, to)
}

// readAppFile reads a file of the app, retrying while Windows reports it
// in use.
func readAppFile(path string) (data []byte, err error) {
	err = update.RetryInUse(func() error {
		data, err = os.ReadFile(path)
		return err
	})
	return data, err
}
