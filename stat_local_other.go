//go:build !linux && !darwin && !windows

package mygo

// localVolume reports whether the file at path is on a volume whose file
// identities can be trusted to name a file for as long as it exists: never
// where the system has no way to say, so stamps compare modification times
// and sizes alone.
func localVolume(string) bool { return false }
