//go:build mygo_noinspector && !js

package mygo

import (
	"errors"
	"os"
)

// Production builds leave the TypeScript generator out (client_generate.go),
// some 400 KB of an app that binds a service or declares an event, as they
// do the inspector: their developer tools are off. What Bind, BindAs and
// NewEvent link instead says so, as the automation plugin does, rather than
// leave a client unwritten for mygo generate to take for an app that binds
// nothing.
func linkClient() {
	ipc.Lock()
	ipc.writeClient = noClient
	ipc.Unlock()
}

func noClient(string) error {
	const msg = "mygo: MYGO_GENERATE cannot write the TypeScript client: this build leaves the generator out (mygo_noinspector); build without that tag, as mygo generate and mygo dev do"
	os.Stderr.WriteString(msg + "\n")
	return errors.New("mygo: the TypeScript client was not written")
}

// clientBuiltOut is whether this build leaves the TypeScript generator out.
const clientBuiltOut = true
