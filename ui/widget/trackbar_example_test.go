package widget_test

import "github.com/egoist/mygo/ui/widget"

type player struct{}

func (player) SetVolume(float64) {}

// The snippets of docs/ui/widget/trackbar.md, which must compile.
func ExampleTrackBar() {
	var player player
	var volume *widget.TrackBarWidget
	volume = widget.TrackBar(0, 100).SetValue(60).OnChange(func() { player.SetVolume(volume.Value()) })
	quality := widget.TrackBar(0, 100).SetStep(25) // 0, 25, 50, 75, 100, with ticks
	_ = quality
}
