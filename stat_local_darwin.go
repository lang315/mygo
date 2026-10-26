package mygo

import "syscall"

// mntLocal is MNT_LOCAL of <sys/mount.h>, which the syscall package lacks:
// the volume is on a disk of this computer.
const mntLocal = 0x1000

// localVolume reports whether the file at path is on a volume whose file
// identities (device and inode) can be trusted to name a file for as long
// as it exists, which network volumes don't promise: false when it is on
// one, and when the system doesn't say.
func localVolume(path string) bool {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return false
	}
	return st.Flags&mntLocal != 0
}
