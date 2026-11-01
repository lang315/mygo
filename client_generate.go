//go:build !mygo_noinspector && !js

package mygo

import "os"

// linkClient makes App.Run write the TypeScript client under
// MYGO_GENERATE, for Bind, BindAs and NewEvent: only apps that bind something link
// the generator (go/parser and all), which plugins' services, bound by
// MyGo, and apps of native UI have no use for. Production builds
// (mygo_noinspector) leave it out too: it is a developer tool, which
// mygo generate and mygo dev build without that tag.
func linkClient() {
	ipc.Lock()
	ipc.writeClient = writeGenerated
	ipc.Unlock()
}

// writeGenerated is what generate mode writes: the TypeScript client to path,
// and, when MYGO_GENERATE_BINDINGS names a file, the list of the bound
// types, for mygo build to name their methods (see bind_methods.go).
func writeGenerated(path string) error {
	if err := WriteTypeScript(path); err != nil {
		return err
	}
	if list := os.Getenv("MYGO_GENERATE_BINDINGS"); list != "" {
		return writeBoundTypes(list)
	}
	return nil
}

// clientBuiltOut is whether this build leaves the TypeScript generator out.
const clientBuiltOut = false
