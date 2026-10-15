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
	for i := 1; ; i++ {
		err := os.Rename(from, to)
		if err == nil || i == 10 || runtime.GOOS != "windows" ||
			// ERROR_SHARING_VIOLATION, or ERROR_ACCESS_DENIED for a directory
			// holding an open file.
			!errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(5)) {
			return err
		}
		time.Sleep(time.Duration(i) * 100 * time.Millisecond)
	}
}
