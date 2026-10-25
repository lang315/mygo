package skillsdocs

import (
	"path"
	"regexp"
	"strings"
	"testing"
)

const roadmapPath = "docs/roadmap.md"

var (
	codeSpanRE = regexp.MustCompile("`([^`]+)`")
	linkRE     = regexp.MustCompile(`\]\(([^)#\s]+)(?:#[^)]*)?\)`)
	pathRE     = regexp.MustCompile(`^\.?[a-z][a-z0-9._-]*/[A-Za-z0-9_./-]+$`)
)

// The roadmap marks work done by naming its evidence, in backticks and in
// links: each file or directory it names must be in the repository.
func TestRoadmapPathsExist(t *testing.T) {
	text := read(t, repoRoot+"/"+roadmapPath)
	checked := 0
	for _, m := range codeSpanRE.FindAllStringSubmatch(prose(text), -1) {
		// A span can run over two lines, so the pairs of backticks are
		// matched across them; its first word is the path, if any.
		words := strings.Fields(m[1])
		if len(words) == 0 {
			continue
		}
		span := words[0]
		if !pathRE.MatchString(span) {
			continue // a name, a command, a class
		}
		checked++
		if !inRepo(strings.TrimSuffix(span, "/")) {
			t.Errorf("%s names %s, which is not in the repository", roadmapPath, span)
		}
	}
	for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
		if strings.Contains(m[1], "://") {
			continue
		}
		checked++
		if !inRepo(path.Join(path.Dir(roadmapPath), m[1])) {
			t.Errorf("%s links %s, which is not in the repository", roadmapPath, m[1])
		}
	}
	if checked < 40 {
		t.Errorf("%d paths checked in the roadmap: did the evidence go?", checked)
	}
	t.Logf("%d paths checked", checked)
}
