package skillsdocs

import (
	"go/types"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/widget/skin/skinfile"
)

const (
	skinDoc   = "../../docs/ui/skin.md"
	skinSkill = "mygo-widget-skin"
)

// skinPackages are the packages docs/ui/skin.md covers, by the name its prose
// gives them.
var skinPackages = []string{"skin", "skinfile"}

// exported lists the names the page must have for pkg: its exported objects,
// as Name, and the exported methods of its exported named types, as
// Type.Method.
func exported(t *testing.T, pkgName string) []string {
	t.Helper()
	pkg := theLoader().pkg(t, packages[pkgName])
	var names []string
	for _, n := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(n)
		if !obj.Exported() {
			continue
		}
		names = append(names, n)
		tn, ok := obj.(*types.TypeName)
		if !ok || tn.IsAlias() {
			continue // an alias is another package's type: its methods are its own
		}
		ms := types.NewMethodSet(types.NewPointer(tn.Type()))
		for i := 0; i < ms.Len(); i++ {
			if m := ms.At(i).Obj(); m.Exported() {
				names = append(names, n+"."+m.Name())
			}
		}
	}
	return names
}

// chainRE finds a dotted chain of names, as skin.Skin.WithAccent.
var chainRE = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*`)

// spanNames returns the names the backtick spans of text hold, with the
// package qualifier of pkgs taken off: skin.Use is Use, skin.Skin.ID is
// Skin.ID.
func spanNames(text string, pkgs []string) map[string]bool {
	out := map[string]bool{}
	for _, span := range spanRE.FindAllString(prose(text), -1) {
		for _, chain := range chainRE.FindAllString(span, -1) {
			parts := strings.Split(chain, ".")
			for _, p := range pkgs {
				if parts[0] == p && len(parts) > 1 {
					parts = parts[1:]
					break
				}
			}
			out[strings.Join(parts, ".")] = true
		}
	}
	return out
}

func TestSkinDocsComplete(t *testing.T) {
	have := spanNames(read(t, skinDoc), skinPackages)
	var missing []string
	for _, pkg := range skinPackages {
		for _, name := range exported(t, pkg) {
			if !have[name] {
				missing = append(missing, pkg+"."+name)
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s lacks, in a backtick span (as Name, pkg.Name, Type.Method or pkg.Type.Method): %s",
			skinDoc, strings.Join(missing, ", "))
	}
}

// TestSkillSkinNamesChecked keeps the skill's skin names under TestSkillNamesExist:
// mentionRE reads every package of packages whole, and the skill mentions skin.
func TestSkillSkinNamesChecked(t *testing.T) {
	for qual := range packages {
		if m := mentionRE.FindStringSubmatch("`" + qual + ".Name`"); m == nil || m[2] != qual {
			t.Errorf("mentionRE does not read %s.Name as a name of package %s", qual, qual)
		}
	}
	seen := map[string]bool{}
	for _, span := range spanRE.FindAllString(prose(read(t, filepath.Join(skillsDir, skinSkill, "SKILL.md"))), -1) {
		for _, m := range mentionRE.FindAllStringSubmatch(span, -1) {
			seen[m[2]] = true
		}
	}
	for _, qual := range skinPackages {
		if !seen[qual] {
			t.Errorf("the %s skill mentions no %s.Name in backticks", skinSkill, qual)
		}
	}
}

var jsonFenceRE = regexp.MustCompile("(?s)```json\n(.*?)```")

func TestSkinFileExample(t *testing.T) {
	file := filepath.Join(skillsDir, skinSkill, "skinfile.md")
	text := read(t, file)
	blocks := jsonFenceRE.FindAllStringSubmatch(text, -1)
	if len(blocks) == 0 {
		t.Fatalf("%s has no json example", file)
	}
	for i, b := range blocks {
		if _, err := skinfile.Parse([]byte(b[1])); err != nil {
			t.Errorf("%s: json block %d does not parse: %v", file, i+1, err)
		}
	}
}
