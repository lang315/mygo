package skin

import (
	"fmt"
	"image"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"sort"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// The sentinel test asks of each pixel whether it moves when the tokens
// move. The gallery is drawn twice, under two skins whose every color differs
// (A and B) and whose metrics are the same: a pixel the same in both, and
// neither white nor black (the neutral materials: knobs, shadows, scrims),
// is a color no token gave, a literal.
//
// Known hole: a translucent or sub-pixel literal over a token's ground moves
// with the ground and passes; thin text in a literal color may too.
// TestNoNewLiteralColors (PR #486) keeps literals out of ui/widget's code
// statically.

// sentinelTolerance is how far apart, in 0-255 RGB, two pixels may be and
// count as the same, and how near white or black a pixel is neutral.
const sentinelTolerance = 3

var black = ui.Color{A: 255} // white is the package's

// colorFields returns pointers to every ui.Color of v, an addressable
// struct, in its fields, the fields of its structs and of its arrays.
func colorFields(v reflect.Value) []*ui.Color {
	var out []*ui.Color
	switch v.Kind() {
	case reflect.Struct:
		if c, ok := v.Addr().Interface().(*ui.Color); ok {
			return []*ui.Color{c}
		}
		for i := range v.NumField() {
			out = append(out, colorFields(v.Field(i))...)
		}
	case reflect.Array:
		for i := range v.Len() {
			out = append(out, colorFields(v.Index(i))...)
		}
	}
	return out
}

// sentinelColors returns n opaque colors, each a CIE76 ΔE of 20 or more from
// the others, from white and from black: picked one by one from a grid of
// RGB, each the farthest from those picked so far.
func sentinelColors(n int) []ui.Color {
	var grid []ui.Color
	for r := 0; r <= 255; r += 51 {
		for g := 0; g <= 255; g += 51 {
			for b := 0; b <= 255; b += 51 {
				grid = append(grid, ui.Color{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
			}
		}
	}
	picked := []ui.Color{white, black}
	for len(picked) < n+2 {
		best, far := ui.Color{}, -1.0
		for _, c := range grid {
			near := math.Inf(1)
			for _, p := range picked {
				near = min(near, deltaE(c, p, nil))
			}
			if near > far {
				best, far = c, near
			}
		}
		picked = append(picked, best)
	}
	return picked[2:]
}

// sentinelTokens returns a copy of Calm's tokens in one appearance whose
// i-th color is sents[(i+shift) % len], keeping its alpha (opaque where the
// token is empty, as a shadow left out).
func sentinelTokens(dark bool, sents []ui.Color, shift int) *skinhook.Tokens {
	tk := Calm.light
	if dark {
		tk = Calm.dark
	}
	for i, c := range colorFields(reflect.ValueOf(&tk).Elem()) {
		s := sents[(i+shift)%len(sents)]
		s.A = c.A
		if c.A == 0 {
			s.A = 255
		}
		*c = s
	}
	return &tk
}

// sentinelSkins are two skins of sentinels, A and B: the same metrics, and
// each color of A another sentinel in B.
type sentinelSkins struct {
	sents []ui.Color
	a, b  [2]*skinhook.Tokens // light, dark
}

func newSentinelSkins(t *testing.T) *sentinelSkins {
	t.Helper()
	n := len(colorFields(reflect.ValueOf(&skinhook.Tokens{}).Elem()))
	sents := sentinelColors(n)
	for i, a := range sents {
		for _, b := range append(slices.Clone(sents[:i]), white, black) {
			if d := deltaE(a, b, nil); d < 20 {
				t.Fatalf("sentinels %v and %v are ΔE %.1f apart, want 20", a, b, d)
			}
		}
	}
	s := &sentinelSkins{sents: sents}
	for i, dark := range []bool{false, true} {
		s.a[i], s.b[i] = sentinelTokens(dark, sents, 0), sentinelTokens(dark, sents, 1)
		ca, cb := colorFields(reflect.ValueOf(s.a[i]).Elem()), colorFields(reflect.ValueOf(s.b[i]).Elem())
		for j := range ca {
			if *ca[j] == (ui.Color{}) || *cb[j] == (ui.Color{}) {
				t.Fatalf("dark=%v: color %d of the tokens is zero", dark, j)
			}
			if x, y := *ca[j], *cb[j]; deltaE(ui.Color{R: x.R, G: x.G, B: x.B, A: 255}, ui.Color{R: y.R, G: y.G, B: y.B, A: 255}, nil) < 20 {
				t.Fatalf("dark=%v: color %d is %v in A and %v in B, want them ΔE 20 apart", dark, j, x, y)
			}
		}
	}
	return s
}

// use installs skin A (b false) or B as Use does, until the test ends.
func (s *sentinelSkins) use(t *testing.T, b bool) {
	tk := s.a
	if b {
		tk = s.b
	}
	skinhook.Set(tk[0], tk[1])
	ui.SetDefaultThemes(themeOf(tk[0], false), themeOf(tk[1], true))
	t.Cleanup(func() { Use(nil) })
}

func colorAt(img *image.RGBA, x, y int) ui.Color {
	p := img.RGBAAt(x, y)
	return ui.Color{R: p.R, G: p.G, B: p.B, A: 255}
}

func dist(a, b ui.Color) float64 {
	dr, dg, db := float64(a.R)-float64(b.R), float64(a.G)-float64(b.G), float64(a.B)-float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// miss is a pixel that did not move: its color, where, and the deepest node
// over it.
type miss struct {
	c    ui.Color
	x, y int
	node ui.TestNode
}

func (m miss) String() string {
	return fmt.Sprintf("%v at (%d, %d) in %v %q", m.c, m.x, m.y, m.node.Role, m.node.Label)
}

// deepest returns the smallest node of nodes over the pixel (x, y), and
// whether one of them is skipped.
func deepest(nodes []ui.TestNode, x, y int, skip func(n ui.TestNode) bool) (ui.TestNode, bool) {
	fx, fy := float32(x)+0.5, float32(y)+0.5
	var best ui.TestNode
	least := float32(math.MaxFloat32)
	for _, n := range nodes {
		r := n.Bounds
		if fx < r.X || fy < r.Y || fx >= r.X+r.W || fy >= r.Y+r.H {
			continue
		}
		if skip != nil && skip(n) {
			return n, true
		}
		if a := r.W * r.H; a < least {
			least, best = a, n
		}
	}
	return best, false
}

// bitmapNode tells the nodes whose bitmap content the test skips: images.
func bitmapNode(n ui.TestNode) bool { return n.Role == ui.RoleImage }

// knobNode tells the nodes that draw a knob, white over its black shadow
// (the neutral materials): sliders, track bars and switches. Only in them
// may a pixel be a gray, a blend of white and black.
func knobNode(n ui.TestNode) bool { return n.Role == ui.RoleSlider || n.Role == ui.RoleSwitch }

// gray reports whether c lies within sentinelTolerance of the segment from
// black to white.
func gray(c ui.Color) bool {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	k := (r + g + b) / 3 // the nearest point of the segment is (k, k, k), inside it
	return math.Sqrt((r-k)*(r-k)+(g-k)*(g-k)+(b-k)*(b-k)) <= sentinelTolerance
}

// still draws what tt shows under skin A and under skin B, and returns the
// pixels inside area (all of the window when empty) that are the same in
// both and neither white nor black, one per color and node. It fails the
// test when the two frames do not lay out alike.
func (s *sentinelSkins) still(t *testing.T, tt *widget.Tester, area ui.Rect) []miss {
	t.Helper()
	s.use(t, false)
	tt.Frame()
	imgA, nodes := tt.Image(), tt.Nodes()
	s.use(t, true)
	tt.Frame()
	imgB, nodesB := tt.Image(), tt.Nodes()
	if len(nodes) != len(nodesB) {
		t.Fatalf("%d nodes under A, %d under B", len(nodes), len(nodesB))
	}
	for i := range nodes {
		if nodes[i].Bounds != nodesB[i].Bounds || nodes[i].Label != nodesB[i].Label {
			t.Fatalf("node %d is %q %v under A, %q %v under B", i, nodes[i].Label, nodes[i].Bounds, nodesB[i].Label, nodesB[i].Bounds)
		}
	}
	b := imgA.Bounds()
	if area.W == 0 {
		area = ui.Rect{W: float32(b.Dx()), H: float32(b.Dy())}
	}
	seen := map[string]bool{}
	var out []miss
	for y := max(int(area.Y), 0); y < int(area.Y+area.H) && y < b.Dy(); y++ {
		for x := max(int(area.X), 0); x < int(area.X+area.W) && x < b.Dx(); x++ {
			a := colorAt(imgA, x, y)
			if dist(a, colorAt(imgB, x, y)) > sentinelTolerance || dist(a, white) <= sentinelTolerance || dist(a, black) <= sentinelTolerance {
				continue
			}
			n, skipped := deepest(nodes, x, y, bitmapNode)
			if skipped || gray(a) && knobNode(n) {
				continue
			}
			if key := fmt.Sprint(a, n.ID); !seen[key] {
				seen[key] = true
				out = append(out, miss{a, x, y, n})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].node.Label < out[j].node.Label })
	return out
}

// report fails the test with the misses, at most 40 lines.
func report(t *testing.T, dark bool, ms []miss) {
	t.Helper()
	for i, m := range ms {
		if i == 40 {
			t.Errorf("dark=%v: %d misses more", dark, len(ms)-40)
			break
		}
		t.Errorf("dark=%v: %v does not move with the tokens", dark, m)
	}
}

// What the theme alone paints moves with the tokens: a bordered plain
// button, and the selected row of a light and a dark DataGridView.
func TestSentinelRule(t *testing.T) {
	s := newSentinelSkins(t)
	type row struct{ ID, Name string }
	for _, dark := range []bool{false, true} {
		grid := widget.DataGridView(func(r *row) string { return r.ID },
			widget.Column[row]{Title: "Name", Text: func(r *row) string { return r.Name }},
		).SetRows([]row{{"1", "Ada"}, {"2", "Grace"}}).SetSize(0, 120)
		s.use(t, false)
		tt := widget.NewTester(t, widget.VBox(8).Add(widget.Button("Plain"), grid), 300, 200)
		tt.SetDark(dark)
		tt.Find(widget.ByRole(ui.RoleRow).Label("Grace")).Click()
		for _, q := range []widget.Query{widget.ByLabel("Plain"), widget.ByRole(ui.RoleRow).Label("Grace")} {
			report(t, dark, s.still(t, tt, tt.Find(q).Bounds()))
		}
		// It is chosen: its end is B's RowSelected over B's Content.
		tk, r := s.b[b2i(dark)], tt.Find(widget.ByRole(ui.RoleRow).Label("Grace")).Bounds()
		if got, want := colorAt(tt.Image(), int(r.X+r.W-4), int(r.Y+r.H/2)), tk.RowSelected.Over(tk.Content); dist(got, want) > 2 {
			t.Errorf("dark=%v: the clicked row fills %v, want RowSelected over Content %v", dark, got, want)
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Literals planted in the gallery's form are each reported, naming their
// node: a panel in the plain accent, a 1 px literal gray border around a box
// of a sentinel, square and rounded, 20 seeded random fills, and a label in
// a literal red (thin text: logged when it slips through, a known hole).
func TestSentinelRuleCatchesPlantedLiterals(t *testing.T) {
	s := newSentinelSkins(t)
	planted := widget.FlowPanel(8).Add(
		widget.Button("").SetAccessibleName("planted panel").Background(ui.Hex("#2563eb")).SetSize(60, 24),
		widget.Label("Planted text").TextColor(ui.Hex("#dc2626")),
		widget.Button("").SetAccessibleName("planted border").Background(s.sents[0]).Border(1, ui.Hex("#71717a")).SetSize(40, 24),
		widget.Button("").SetAccessibleName("planted corner").Background(s.sents[0]).Border(1, ui.Hex("#71717a")).Radius(6).SetSize(40, 24),
		// A gray, which only a knob may paint.
		widget.Button("").SetAccessibleName("planted gray").Background(ui.Hex("#808080")).SetSize(40, 24),
	)
	want := []string{"planted panel", "planted border", "planted corner", "planted gray"}
	rng := rand.New(rand.NewPCG(3, 4))
	for i := 0; i < 20; {
		c := ui.Color{R: uint8(rng.IntN(256)), G: uint8(rng.IntN(256)), B: uint8(rng.IntN(256)), A: 255}
		if dist(c, white) < 40 || dist(c, black) < 40 {
			continue
		}
		name := fmt.Sprintf("random %d %v", i, c)
		planted.Add(widget.Button("").SetAccessibleName(name).Background(c).SetSize(24, 24))
		want = append(want, name)
		i++
	}
	for _, dark := range []bool{false, true} {
		s.use(t, false)
		tt := widget.TestForm(t, gallery(planted))
		tt.SetDark(dark)
		found := map[string]bool{}
		for _, m := range s.still(t, tt, ui.Rect{}) {
			found[m.node.Label] = true
		}
		for _, name := range want {
			if !found[name] {
				t.Errorf("dark=%v: no miss is reported in %q", dark, name)
			}
		}
		if !found["Planted text"] {
			t.Logf("dark=%v: the planted red text is not reported: thin text, a known hole", dark)
		}
	}
}

// knownMisses are the nodes, by label, whose misses only log: parts a later
// task of the plan brings onto the skin, each named with it. PR 2 and PR 3
// remove theirs; the list ends empty.
var knownMisses = map[string]string{
	"chart": "Task 5: the chart's series colors",
}

// Every pixel of every control of the gallery, light and dark, moves when
// the tokens move, but white and black: no control paints a color the skin
// does not give it.
func TestSentinelSkinReachesEveryPixel(t *testing.T) {
	s := newSentinelSkins(t)
	for _, dark := range []bool{false, true} {
		s.use(t, false)
		tt := widget.TestForm(t, gallery())
		tt.SetDark(dark)
		galleryStates(tt)
		var ms []miss
		for _, m := range s.still(t, tt, ui.Rect{}) {
			if task, ok := knownMisses[m.node.Label]; ok {
				t.Logf("dark=%v: known (%s): %v", dark, task, m)
				continue
			}
			ms = append(ms, m)
		}
		report(t, dark, ms)
	}
}
