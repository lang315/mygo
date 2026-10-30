//go:build !mygo_noinspector && !js

package mygo

// linkClient makes App.Run write the TypeScript client under
// MYGO_GENERATE, for Bind, BindAs and NewEvent: only apps that bind something link
// the generator (go/parser and all), which plugins' services, bound by
// MyGo, and apps of native UI have no use for. Production builds
// (mygo_noinspector) leave it out too: it is a developer tool, which
// mygo generate and mygo dev build without that tag.
func linkClient() {
	ipc.Lock()
	ipc.writeClient = WriteTypeScript
	ipc.Unlock()
}

// clientBuiltOut is whether this build leaves the TypeScript generator out.
const clientBuiltOut = false
