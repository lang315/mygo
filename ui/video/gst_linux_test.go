package video

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// TestMissingDecoder checks that a video GStreamer has no decoder for
// fails with an error that says so, whichever of the pipeline's two
// reports, the missing plugin's and the demuxer's "not linked", comes
// first. rgb.mp4's H.264 is made a codec nothing decodes.
func TestMissingDecoder(t *testing.T) {
	data := bytes.ReplaceAll(rgbMP4, []byte("avc1"), []byte("zzzz"))
	if bytes.Equal(data, rgbMP4) {
		t.Fatal("rgb.mp4 has no avc1")
	}
	p, err := OpenData(data)
	if errors.Is(err, ErrUnsupported) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	if gstIsMissingPlugin == nil {
		t.Skip("no libgstpbutils")
	}
	wait.Until(t, "an error", func() bool { return p.Err() != nil })
	if err := p.Err(); !strings.Contains(err.Error(), "decoder") {
		t.Fatalf("error %q does not name the missing decoder", err)
	}
}
