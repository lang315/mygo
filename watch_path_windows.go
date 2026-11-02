package mygo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// OpenRoot uses an ordinary Windows handle that prevents deletion and rename.
// Root.OpenRoot shares deletion, so retain that anchored handle instead and
// release the temporary parent before returning. A drive's root has no
// parent, and can't be renamed anyway.
func fileWatchOpenRoot(path string) (*os.Root, error) {
	if filepath.Dir(path) == path {
		return os.OpenRoot(path)
	}
	parent, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	return parent.OpenRoot(filepath.Base(path))
}

// fileWatchInaccessible reports whether err says a file can't be looked
// at: without permission, or held by the system with a sharing mode that
// leaves out even reading its attributes, as pagefile.sys and
// hiberfil.sys.
func fileWatchInaccessible(err error) bool {
	const errorSharingViolation, errorLockViolation = syscall.Errno(32), syscall.Errno(33)
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, errorSharingViolation) || errors.Is(err, errorLockViolation)
}

// fileWatchEntryInfo returns what entry is, as reading its directory said.
// On NTFS and ReFS that reading (FILE_ID_BOTH_DIR_INFO, which Go reads since
// 1.23) carries the volume and file index os.SameFile compares, so the entry
// needs no stat of its own: opening a handle for each one cost 30 µs an entry,
// three quarters of a scan. It says what the file system keeps in the entry,
// which is what it notifies of: the size and times of a file open for writing
// show once it is closed, and a write through a hard link in another
// directory shows in this one's entry when something opens it. A file system
// that has no file index (FAT), and a link or other reparse point, are
// looked at in root by name (Lstat), which says what they are.
func fileWatchEntryInfo(root *os.Root, entry fs.DirEntry) (os.FileInfo, error) {
	if info, err := entry.Info(); err == nil && info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
		if _, ok := fileWatchIdentity(info); ok {
			return info, nil
		}
	}
	return root.Lstat(entry.Name())
}
