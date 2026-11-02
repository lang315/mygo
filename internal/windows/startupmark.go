//go:build windows && (amd64 || arm64)

package windows

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// DRAFT (measurement): startupMark appends the milestone name and the time
// to the file MYGO_STARTUP_LOG, the first time it is told of each name, as
// the apps of internal/startupbench do. Nothing happens without the variable.
var startup struct {
	sync.Mutex
	path string
	seen map[string]bool
}

func init() { startup.path = os.Getenv("MYGO_STARTUP_LOG") }

func startupMark(name string) {
	if startup.path == "" {
		return
	}
	t := time.Now().UnixNano()
	startup.Lock()
	defer startup.Unlock()
	if startup.seen == nil {
		startup.seen = map[string]bool{}
	}
	if startup.seen[name] {
		return
	}
	startup.seen[name] = true
	if f, err := os.OpenFile(startup.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		fmt.Fprintf(f, "%s %d\n", name, t)
		f.Close()
	}
}
