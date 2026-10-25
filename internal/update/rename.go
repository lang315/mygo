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

// retryGOOS and sleep are what RetryInUse asks the system and waits with, for
// its test on any system.
var (
	retryGOOS = runtime.GOOS
	sleep     = time.Sleep
)

// retryTime is how long RetryInUse waits in all.
const retryTime = 5 * time.Second

// RetryInUse runs f, and again for a few seconds while it fails because
// Windows reports a file in use. The waits start at a millisecond and double
// up to a second: what holds the file for an instant, a reader or a scanner,
// lets go soon, and many tries within the first moments find it closed.
func RetryInUse(f func() error) error {
	wait, waited := time.Millisecond, time.Duration(0)
	for {
		err := f()
		if err == nil || waited >= retryTime || retryGOOS != "windows" ||
			// ERROR_SHARING_VIOLATION, or ERROR_ACCESS_DENIED for a file
			// being deleted or replaced, or a directory holding an open file.
			!errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(5)) {
			return err
		}
		wait = min(wait, retryTime-waited)
		sleep(wait)
		waited += wait
		wait = min(wait*2, time.Second)
	}
}
