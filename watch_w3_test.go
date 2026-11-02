package mygo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkFileWatchWalkParts times the parts of looking at a tree the way a
// recursive watch's scan does (a generated tree nine directories deep), per
// entry: reading a directory, and the ways to learn what an entry is.
func BenchmarkFileWatchWalkParts(b *testing.B) {
	dir := b.TempDir()
	var fill func(p string, depth int)
	fill = func(p string, depth int) {
		for i := range 8 {
			if err := os.WriteFile(filepath.Join(p, fmt.Sprint("f", i)), nil, 0600); err != nil {
				b.Fatal(err)
			}
		}
		if depth == 9 {
			return
		}
		for i := range 2 {
			sub := filepath.Join(p, fmt.Sprint("d", i))
			if err := os.Mkdir(sub, 0700); err != nil {
				b.Fatal(err)
			}
			fill(sub, depth+1)
		}
	}
	fill(dir, 1)
	var dirs []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	entries := 0
	run := func(name string, each func(root *os.Root, dirPath string, e os.DirEntry)) {
		b.Run(name, func(b *testing.B) {
			n := 0
			for b.Loop() {
				n = 0
				for _, d := range dirs {
					root, err := os.OpenRoot(d)
					if err != nil {
						b.Fatal(err)
					}
					f, err := root.Open(".")
					if err != nil {
						b.Fatal(err)
					}
					es, err := f.ReadDir(-1)
					if err != nil {
						b.Fatal(err)
					}
					f.Close()
					for _, e := range es {
						n++
						if each != nil {
							each(root, d, e)
						}
					}
					root.Close()
				}
			}
			entries = n
			b.ReportMetric(float64(n), "entries")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(n)/1000, "µs/entry")
		})
	}
	run("readdir-only", nil)
	run("dirent-info", func(_ *os.Root, _ string, e os.DirEntry) {
		if _, err := e.Info(); err != nil {
			b.Fatal(err)
		}
	})
	run("dirent-info+identity", func(_ *os.Root, _ string, e os.DirEntry) {
		info, err := e.Info()
		if err != nil {
			b.Fatal(err)
		}
		fileWatchEntryOf(info)
	})
	run("root-lstat", func(root *os.Root, _ string, e os.DirEntry) {
		if _, err := root.Lstat(e.Name()); err != nil {
			b.Fatal(err)
		}
	})
	run("root-lstat+identity", func(root *os.Root, _ string, e os.DirEntry) {
		info, err := fileWatchEntryInfo(root, e)
		if err != nil {
			b.Fatal(err)
		}
		fileWatchEntryOf(info)
	})
	run("os-lstat", func(_ *os.Root, d string, e os.DirEntry) {
		if _, err := os.Lstat(filepath.Join(d, e.Name())); err != nil {
			b.Fatal(err)
		}
	})
	_ = entries
}
