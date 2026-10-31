package fsretry

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// A file in use for a short time, as one a reader or a scanner opens and
// closes over and over, is open at most of the instants a retry looks: the
// first retries come soon, and many, so that one finds it closed. Ten tries
// with the first wait at 100 ms failed a write in 200 of a test that read the
// file without a pause (TestAppFileInUse, "Access is denied").
func TestRetryInUseTriesSoonAndOften(t *testing.T) {
	retryGOOS = "windows"
	var slept []time.Duration
	sleep = func(d time.Duration) { slept = append(slept, d) }
	t.Cleanup(func() { retryGOOS, sleep = runtime.GOOS, time.Sleep })
	inUse := &os.PathError{Op: "rename", Err: syscall.Errno(5)}
	tries := 0
	err := RetryInUse(func() error {
		if tries++; tries <= 12 {
			return inUse
		}
		return nil
	})
	if err != nil {
		t.Fatalf("12 tries in use, then free: %v after %d tries", err, tries)
	}
	if len(slept) == 0 || slept[0] > 10*time.Millisecond {
		t.Errorf("the first wait is %v, want a few milliseconds", slept)
	}
	var total time.Duration
	for _, d := range slept {
		total += d
	}
	if total > 6*time.Second {
		t.Errorf("waited %v in all, want a few seconds", total)
	}
	// A file that stays in use fails once the time is spent.
	tries, slept = 0, nil
	if err := RetryInUse(func() error { tries++; return inUse }); !errors.Is(err, syscall.Errno(5)) {
		t.Errorf("a file always in use: %v", err)
	}
}
