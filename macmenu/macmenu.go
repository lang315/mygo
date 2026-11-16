// Package macmenu links the parts of menus that macOS draws only for apps
// that ask for them: the column a menu's badges share (MenuItem.Badge and
// BadgeAlign) and items that keep their menu open (MenuItem.KeepOpen).
// Import it for its effect:
//
//	import _ "github.com/egoist/mygo/macmenu"
//
// Without it, macOS shows a badge after its item's label, as the other
// platforms do, and a KeepOpen item closes its menu like any other. On other
// platforms the package does nothing.
package macmenu
