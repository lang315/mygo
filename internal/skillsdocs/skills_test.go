package skillsdocs

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	skillsDir = "../../cmd/mygo/template/shared/.agents/skills"
	repoRoot  = "../.."
	module    = "github.com/egoist/mygo"
)

// packages are the packages whose exported names the prose of a skill may
// mention in backticks, as widget.Button or ui.NewTester, by the name it uses.
var packages = map[string]string{
	"widget": module + "/ui/widget",
	"ui":     module + "/ui",
	"tw":     module + "/ui/widget/tw",
	"mygo":   module,
}

// skill is one bundled skill.
type skill struct {
	name  string
	dir   string
	files []string // the Markdown files, SKILL.md first
}

func skills(t *testing.T) []skill {
	t.Helper()
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s := skill{name: e.Name(), dir: filepath.Join(skillsDir, e.Name())}
		s.files = append(s.files, filepath.Join(s.dir, "SKILL.md"))
		filepath.WalkDir(s.dir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") && filepath.Base(p) != "SKILL.md" {
				s.files = append(s.files, p)
			}
			return nil
		})
		out = append(out, s)
	}
	if len(out) == 0 {
		t.Fatal("no skills found")
	}
	return out
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var (
	frontRE = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)
	fieldRE = regexp.MustCompile(`(?m)^(\w+):\s*(.*)$`)
)

func TestSkillMetadata(t *testing.T) {
	for _, s := range skills(t) {
		m := frontRE.FindStringSubmatch(read(t, s.files[0]))
		if m == nil {
			t.Errorf("%s: SKILL.md has no front matter", s.name)
			continue
		}
		fields := map[string]string{}
		for _, f := range fieldRE.FindAllStringSubmatch(m[1], -1) {
			fields[f[1]] = strings.TrimSpace(f[2])
		}
		if fields["name"] != s.name {
			t.Errorf("%s: front matter name is %q, want the directory's", s.name, fields["name"])
		}
		if d := fields["description"]; d == "" || len(d) > 1024 {
			t.Errorf("%s: description is %d bytes, want 1 to 1024", s.name, len(d))
		}
		yaml := filepath.Join(s.dir, "agents", "openai.yaml")
		y, err := os.ReadFile(yaml)
		if err != nil {
			t.Errorf("%s: %v", s.name, err)
			continue
		}
		for _, key := range []string{"display_name:", "short_description:"} {
			if !strings.Contains(string(y), key) {
				t.Errorf("%s: agents/openai.yaml lacks %s", s.name, key)
			}
		}
	}
}

var (
	fenceRE   = regexp.MustCompile("(?s)```(\\w*)\n(.*?)```")
	docPathRE = regexp.MustCompile("docs/[A-Za-z0-9_./-]+\\.md")
	nameRE    = regexp.MustCompile("`(widget|ui|tw|mygo)\\.([A-Za-z_][A-Za-z0-9_]*)")
)

// prose is the text of a Markdown file outside its code blocks.
func prose(text string) string { return fenceRE.ReplaceAllString(text, "") }

func TestSkillDocsExist(t *testing.T) {
	for _, s := range skills(t) {
		for _, f := range s.files {
			for _, p := range docPathRE.FindAllString(read(t, f), -1) {
				if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(p))); err != nil {
					t.Errorf("%s mentions %s, which is not in the repository", f, p)
				}
			}
		}
	}
}

// loader type-checks files against the repository's source.
type loader struct {
	fset *token.FileSet
	imp  types.Importer
	pkgs map[string]*types.Package
}

// fromRepo imports packages as the repository's own code does, through its
// module.
type fromRepo struct {
	from types.ImporterFrom
	dir  string
}

func (r fromRepo) Import(path string) (*types.Package, error) {
	return r.from.ImportFrom(path, r.dir, 0)
}

func newLoader() *loader {
	if build.Default.GOROOT == "" { // go test -trimpath leaves it empty
		out, err := exec.Command("go", "env", "GOROOT").Output()
		if err != nil {
			panic(err)
		}
		build.Default.GOROOT = strings.TrimSpace(string(out))
	}
	fset := token.NewFileSet()
	dir, err := filepath.Abs(repoRoot)
	if err != nil {
		panic(err)
	}
	from := importer.ForCompiler(fset, "source", nil).(types.ImporterFrom)
	return &loader{fset: fset, imp: fromRepo{from, dir}, pkgs: map[string]*types.Package{}}
}

func (l *loader) pkg(t *testing.T, path string) *types.Package {
	if p, ok := l.pkgs[path]; ok {
		return p
	}
	p, err := l.imp.Import(path)
	if err != nil {
		t.Fatalf("importing %s: %v", path, err)
	}
	l.pkgs[path] = p
	return p
}

// names of the prose that are not exported by their package but are fine to
// write: none now; add with a reason.
var allowedNames = map[string]string{}

func TestSkillNamesExist(t *testing.T) {
	l := newLoader()
	for _, s := range skills(t) {
		for _, f := range s.files {
			for _, m := range nameRE.FindAllStringSubmatch(prose(read(t, f)), -1) {
				qual, name := m[1], m[2]
				if allowedNames[qual+"."+name] != "" {
					continue
				}
				if obj := l.pkg(t, packages[qual]).Scope().Lookup(name); obj == nil || !obj.Exported() {
					t.Errorf("%s mentions %s.%s, which %s does not export", f, qual, name, packages[qual])
				}
			}
		}
	}
	if len(allowedNames) > 3 {
		t.Errorf("%d allowed names: fix the skills rather than allow more", len(allowedNames))
	}
}

func TestSkillExamplesCompile(t *testing.T) {
	l := newLoader()
	blocks := 0
	for _, s := range skills(t) {
		for _, f := range s.files {
			text := read(t, f)
			for _, m := range fenceRE.FindAllStringSubmatchIndex(text, -1) {
				if text[m[2]:m[3]] != "go" {
					continue
				}
				blocks++
				line := 1 + strings.Count(text[:m[4]], "\n")
				where := f + ":" + itoa(line)
				file, err := parser.ParseFile(l.fset, where, text[m[4]:m[5]], parser.AllErrors)
				if err != nil {
					t.Errorf("%s: a Go example must be a whole file (package, imports): %v", where, err)
					continue
				}
				conf := types.Config{
					Importer: l.imp,
					Error:    func(err error) { t.Errorf("%s", err) },
				}
				conf.Check(file.Name.Name, l.fset, []*ast.File{file}, nil)
			}
		}
	}
	t.Logf("%d Go examples checked", blocks)
}

func itoa(n int) string {
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	if s == "" {
		return "0"
	}
	return s
}

var _ = sort.Strings
