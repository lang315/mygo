//go:build !js

package e2e

import (
	"sync"
	"testing"
	_ "unsafe" // for go:linkname

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// docWork is mygo's count of the work it does off the main thread for
// documents and app files (document.go): the writes of window states,
// recent documents and recovery copies, which read PathUserData when they
// run, and the watches of open documents.
//
//go:linkname docWork github.com/egoist/mygo.docWork
var docWork sync.WaitGroup

// tempUserData has PathUserData in a directory of the test's until the
// test ends. The reset is a cleanup registered before the test's windows
// and documents, so it runs after their cleanups (a defer would run before
// them), and it waits for the writes they queued: a keyed window's Destroy
// queues its state's, which reads the path when it runs. Without either,
// the state goes into the runner's real user-data directory.
func tempUserData(t *testing.T) {
	mygo.App.SetPath(mygo.PathUserData, t.TempDir())
	t.Cleanup(func() {
		defer mygo.App.SetPath(mygo.PathUserData, "") // even when the wait fails
		done := make(chan struct{})
		go func() { docWork.Wait(); close(done) }()
		wait.For(t, "mygo's writes of app files", done)
	})
}
