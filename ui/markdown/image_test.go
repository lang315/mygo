package markdown

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileImagesStaysInDir(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	dir := filepath.Join(parent, "doc")
	for _, p := range []string{filepath.Join(parent, "x.png"), filepath.Join(dir, "img", "a.png")} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A PNG with more than 64 MB after it.
	big := filepath.Join(dir, "big.png")
	if err := os.WriteFile(big, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(big, 64<<20+1); err != nil {
		t.Fatal(err)
	}
	load := FileImages(dir)

	refused := map[string]string{
		"parent":   "../x.png",
		"absolute": filepath.ToSlash(filepath.Join(parent, "x.png")),
		"file":     "file:../x.png",
		"device":   "/dev/zero",
		"large":    "big.png",
		"folder":   "img",
	}
	if runtime.GOOS == "windows" {
		refused["share"] = "//host/share/a.png"
	} else {
		if err := os.Symlink("/dev/zero", filepath.Join(dir, "zero.png")); err != nil {
			t.Fatal(err)
		}
		refused["link"] = "zero.png"
		if err := os.Symlink(filepath.Join(parent, "x.png"), filepath.Join(dir, "out.png")); err != nil {
			t.Fatal(err)
		}
		refused["link out"] = "out.png"
	}
	for name, url := range refused {
		t.Run(name, func(t *testing.T) {
			if src, err := load(url); err == nil {
				t.Errorf("%s loaded %v from %s, want refused", url, src, dir)
			}
		})
	}
	t.Run("in dir", func(t *testing.T) {
		for _, url := range []string{"img/a.png", "file:img/a.png", "img/../img/a.png"} {
			src, err := load(url)
			if err != nil {
				t.Fatalf("%s: %v", url, err)
			}
			if w, h := src.(interface{ Size() (int, int) }).Size(); w != 3 || h != 2 {
				t.Errorf("%s is %d×%d, want 3×2", url, w, h)
			}
		}
	})
}
