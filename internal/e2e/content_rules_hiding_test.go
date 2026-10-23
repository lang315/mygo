//go:build !js

package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

// TestContentRulesHiding applies lists with what the script that hides
// elements on Windows must live with: a pattern JavaScript rejects (^*, which
// Go takes), a selector that is not a rule, and a list of 200,000 selectors.
// Each leaves the banner hidden and the rest of the page be.
func TestContentRulesHiding(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("macOS and Linux hide elements with WebKit's content blockers")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>hiding</title>
<div class="banner" id="banner">ad</div><div class="bad" id="bad">kept</div>`)
	}))
	defer server.Close()
	hide := func(trigger, selector string) string {
		return `{"trigger":{"url-filter":"` + trigger + `"},"action":{"type":"css-display-none","selector":"` + selector + `"}}`
	}
	many := make([]string, 0, 200001)
	for i := range 200000 {
		many = append(many, hide(".*", fmt.Sprintf(".n%d", i)))
	}
	for _, c := range []struct{ name, rules string }{
		{"pattern", "[" + hide("^*", ".bad") + "," + hide(".*", ".banner") + "]"},
		{"selector", "[" + hide(".*", ".bad{") + "," + hide(".*", ".banner") + "]"},
		{"many", "[" + strings.Join(append(many, hide(".*", ".banner")), ",") + "]"},
	} {
		t.Run(c.name, func(t *testing.T) {
			rules, err := mygo.CompileContentRules("e2e-hiding-"+c.name, c.rules)
			if err != nil {
				t.Fatal(err)
			}
			w := newWindow(t, mygo.WindowOptions{Title: "Hiding", Width: 300, Height: 200, Page: mygo.PageOptions{ContentRules: []*mygo.ContentRules{rules}}})
			if err := w.Page().LoadURL(server.URL + "/"); err != nil {
				t.Fatal(err)
			}
			waitFor(t, w, "document.title === 'hiding'")
			got, err := mygo.EvalAs[string](w.Page(), `["banner", "bad"].map(id => getComputedStyle(document.getElementById(id)).display).join(" ")`)
			if err != nil {
				t.Fatal(err)
			}
			if got != "none block" {
				t.Errorf("banner and bad are %q, want the banner hidden and the other element shown", got)
			}
		})
	}
}
