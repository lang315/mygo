package mygo

import "syscall"

// localVolume reports whether the file at path is on a volume whose file
// identities (device and inode) can be trusted to name a file for as long
// as it exists, which network volumes and FUSE do not promise: false when
// it is on one, and when the system doesn't say.
func localVolume(path string) bool {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return false
	}
	switch uint32(st.Type) { // int32 on some architectures
	case 0xFF534D42, // CIFS
		0xFE534D42, // SMB2
		0x517B,     // SMB
		0x6969,     // NFS
		0x65735546, // FUSE
		0x01021997: // 9p, as WSL mounts Windows drives
		return false
	}
	return true
}
