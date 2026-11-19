package skin

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Every token has a reader: a file of package widget's tree that reads it,
// or themeOf, which gives it to ui. The fields of the families are left to
// P2's components, as their doc says; the family itself must be read.
//
// shortcut: a text match, so a common name (Text) passes on any reader; upgrade to go/types if a field slips through.
func TestEveryTokenIsRead(t *testing.T) {
	var src []string
	root := ".." // ui/widget
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(path)
		if d.IsDir() {
			if slash == "../internal/skinhook" || slash == "../skin" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			src = append(src, string(b))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	theme, err := os.ReadFile("theme.go")
	if err != nil {
		t.Fatal(err)
	}
	src = append(src, string(theme))
	tk := reflect.TypeFor[skinhook.Tokens]()
	for i := range tk.NumField() {
		name := tk.Field(i).Name
		if !readIn(src, name) {
			t.Errorf("no reader of Tokens.%s: no file of ui/widget nor themeOf reads .%s", name, name)
		}
	}
}

// readIn says whether a source of src has "."+name followed by a byte that
// cannot go on an identifier.
func readIn(src []string, name string) bool {
	for _, s := range src {
		for rest := s; ; {
			i := strings.Index(rest, "."+name)
			if i < 0 {
				break
			}
			rest = rest[i+1+len(name):]
			if rest == "" || !isIdent(rest[0]) {
				return true
			}
		}
	}
	return false
}

// readIn takes a name only where it ends: .Label in .LabelWidget is not a
// read of Label.
func TestReadIn(t *testing.T) {
	for _, c := range []struct {
		src  string
		want bool
	}{
		{"tk.Label)", true}, {"tk.Label", true}, {"tk.Label.Size", true},
		{"x.LabelWidget", false}, {"Label(", false}, {"x.Label_", false}, {"x.Label2 x.Labels", false},
	} {
		if got := readIn([]string{c.src}, "Label"); got != c.want {
			t.Errorf("readIn(%q) is %v, want %v", c.src, got, c.want)
		}
	}
}

func isIdent(b byte) bool {
	return b == '_' || '0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || b >= 0x80
}
