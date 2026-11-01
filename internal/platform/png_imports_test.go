package platform_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestBackendsDoNotImportPNG keeps image/png's encoder, and with it
// compress/flate's compressor, out of native apps: the darwin and linux
// backends hand pixels to AppKit and GDK as they are.
func TestBackendsDoNotImportPNG(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go command")
	}
	for _, c := range []struct{ goos, goarch, pkg string }{
		{"darwin", "arm64", "github.com/egoist/mygo/internal/darwin"},
		{"linux", "amd64", "github.com/egoist/mygo/internal/linux"},
	} {
		cmd := exec.Command("go", "list", "-f", `{{join .Imports " "}}`, c.pkg)
		cmd.Env = append(os.Environ(), "GOOS="+c.goos, "GOARCH="+c.goarch, "CGO_ENABLED=0")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list %s for %s/%s: %v", c.pkg, c.goos, c.goarch, err)
		}
		for _, imp := range strings.Fields(string(out)) {
			if imp == "image/png" {
				t.Errorf("%s imports image/png for %s/%s", c.pkg, c.goos, c.goarch)
			}
		}
	}
}
