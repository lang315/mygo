package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The settings dialog of the pilot: a GroupBox in a padded box, holding a
// stacked field whose combo has a width of its own, wider than the room.
func TestAFixedWidthControlDoesNotOverflowItsGroupBox(t *testing.T) {
	combo := ComboBox(Option{"true", "Có"}, Option{"false", "Không"}).SetValue("false")
	combo.SetSize(400, 0)
	form := Fields().Stacked().Row("Mở chrome trước khi chạy:", combo)
	d := NewDialog(nil, "Cấu hình", "Cài đặt", 500, 0,
		VBox(8).Add(GroupBox("Trình duyệt").Add(form)).Padding(12))
	dt := TestDialog(t, d)
	title := dt.Find(ByLabel("Trình duyệt")).Bounds() // as wide as the card's content
	field := dt.Find(ByRole(ui.RolePopUpButton)).Bounds()
	if right, room := field.X+field.W, title.X+title.W; right > room+0.5 {
		t.Fatalf("the combo ends at %v, past the card's content at %v", right, room)
	}
}

func TestAFixedWidthKeepsItsSizeWhereThereIsRoom(t *testing.T) {
	combo := ComboBox(Option{"a", "A"}).SetValue("a")
	combo.SetSize(400, 0)
	tt := NewTester(t, VBox(8).Add(combo).Padding(1), 900, 300)
	if w := tt.Find(ByRole(ui.RolePopUpButton)).Bounds().W; w != 400 {
		t.Fatalf("width %v, want its own 400", w)
	}
}
