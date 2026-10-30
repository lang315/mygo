//go:build mygo_noinspector && !js

package mygo

// Production builds leave the TypeScript generator out (client_generate.go),
// some 400 KB of an app that binds a service, as they do the inspector:
// their developer tools are off.
func linkClient() {}

// clientBuiltOut is whether this build leaves the TypeScript generator out.
const clientBuiltOut = true
