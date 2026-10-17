package update

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"time"
)

// Rename renames a file, retrying for a few seconds while Windows reports
// it in use: antivirus software opens new executables, such as an app that
// just started, the one of an update or a library just downloaded, to
// scan them.
func Rename(from, to string) error {
	return RetryInUse(func() error { return os.Rename(from, to) })
}

// RetryInUse runs f, and again for a few seconds while it fails because
// Windows reports a file in use.
func RetryInUse(f func() error) error {
	for i := 1; ; i++ {
		err := f()
		if err == nil || i == 10 || runtime.GOOS != "windows" ||
			// ERROR_SHARING_VIOLATION, or ERROR_ACCESS_DENIED for a file
			// being deleted or replaced, or a directory holding an open file.
			!errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(5)) {
			return err
		}
		time.Sleep(time.Duration(i) * 100 * time.Millisecond)
	}
}
