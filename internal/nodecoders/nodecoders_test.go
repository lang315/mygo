// Package nodecoders holds the tests of programs that import no decoder of an
// optional image format (JPEG, WebP, BMP): this test binary imports none, so
// what the packages that decode images say of an image they cannot decode is what
// such a program sees.
package nodecoders

import (
	"encoding/base64"
	"errors"
	"image"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/egoist/mygo/ui/widget"
)

// a 4 by 4 JPEG
const jpgBase64 = "/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AABEIAAQABAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APn+iiigD//Z"

func jpeg(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(jpgBase64)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func namesTheImport(err error) bool {
	return err != nil && errors.Is(err, image.ErrFormat) && strings.Contains(err.Error(), `import _ "image/jpeg"`)
}

func TestDecodeBitmapNamesTheImport(t *testing.T) {
	if _, err := ui.DecodeBitmap(jpeg(t)); !namesTheImport(err) {
		t.Errorf("DecodeBitmap of a JPEG: %v", err)
	}
	if _, err := ui.DecodeBitmapConfig(jpeg(t)); !namesTheImport(err) {
		t.Errorf("DecodeBitmapConfig of a JPEG: %v", err)
	}
}

// ImageList.Add reads the size of an image first: its error names the import too.
func TestImageListAddNamesTheImport(t *testing.T) {
	if err := widget.ImageList(16).Add("photo", jpeg(t)); !namesTheImport(err) {
		t.Errorf("ImageList.Add of a JPEG: %v", err)
	}
}

// HTML pasted from a browser with a JPEG in a data: URL has no image in this
// program, and says why in the log, once. The log is once for each format in a
// process, so a child process makes the one that this test reads.
func TestPastedJPEGSaysWhyItHasNoImage(t *testing.T) {
	if os.Getenv("MYGO_NODECODERS_CHILD") != "" {
		richtext.ParseHTML(`<p><img alt="photo" src="data:image/jpeg;base64,` + jpgBase64 + `"></p>`)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPastedJPEGSaysWhyItHasNoImage$")
	cmd.Env = append(os.Environ(), "MYGO_NODECODERS_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `import _ "image/jpeg"`) {
		t.Errorf("pasting a JPEG in HTML logged nothing that names the import:\n%s", out)
	}
}
