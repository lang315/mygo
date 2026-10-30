package httplite

import (
	"net/http"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// TestNamesAreNetHTTPs: every name is net/http's.
func TestNamesAreNetHTTPs(t *testing.T) {
	if MethodGet != http.MethodGet || TimeFormat != http.TimeFormat {
		t.Errorf("MethodGet %q TimeFormat %q", MethodGet, TimeFormat)
	}
	for name, pair := range map[string][2]int{
		"StatusOK":                 {StatusOK, http.StatusOK},
		"StatusNoContent":          {StatusNoContent, http.StatusNoContent},
		"StatusTemporaryRedirect":  {StatusTemporaryRedirect, http.StatusTemporaryRedirect},
		"StatusForbidden":          {StatusForbidden, http.StatusForbidden},
		"StatusBadGateway":         {StatusBadGateway, http.StatusBadGateway},
		"StatusServiceUnavailable": {StatusServiceUnavailable, http.StatusServiceUnavailable},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %d, net/http has %d", name, pair[0], pair[1])
		}
	}
	for code := -5; code < 700; code++ {
		if got, want := StatusText(code), http.StatusText(code); got != want {
			t.Errorf("StatusText(%d) = %q, net/http has %q", code, got, want)
		}
	}
}

// TestHeaderIsNetHTTPsHeader: Get, Set, Add and Clone do what http.Header's
// do, for keys in any case and for values already there, and the types
// convert to each other.
func TestHeaderIsNetHTTPsHeader(t *testing.T) {
	keys := []string{"Content-Type", "content-type", "CONTENT-TYPE", "x-custom-header", "X-A", "", "bad key", "Accept-Ranges"}
	start := []map[string][]string{nil, {}, {"Content-Type": {"a", "b"}, "x-raw": {"c"}}}
	for _, init := range start {
		for _, key := range keys {
			mine, theirs := Header(cloneMap(init)), http.Header(cloneMap(init))
			if init == nil {
				mine, theirs = Header{}, http.Header{}
			}
			if mine.Get(key) != theirs.Get(key) {
				t.Errorf("Get(%q) on %v: %q, net/http has %q", key, init, mine.Get(key), theirs.Get(key))
			}
			mine.Set(key, "v1")
			theirs.Set(key, "v1")
			mine.Add(key, "v2")
			theirs.Add(key, "v2")
			mine.Add("Other", "x")
			theirs.Add("Other", "x")
			if !reflect.DeepEqual(map[string][]string(mine), map[string][]string(theirs)) {
				t.Errorf("after Set, Add on %v with %q: %v, net/http has %v", init, key, mine, theirs)
			}
			if !reflect.DeepEqual(map[string][]string(mine.Clone()), map[string][]string(theirs.Clone())) {
				t.Errorf("Clone: %v, net/http has %v", mine.Clone(), theirs.Clone())
			}
			c := mine.Clone()
			c.Add("Other", "y")
			if len(mine["Other"]) != 1 {
				t.Errorf("Clone shares the values: %v", mine)
			}
		}
	}
	if Header(nil).Clone() != nil || http.Header(nil).Clone() != nil {
		t.Error("Clone of a nil header is not nil")
	}
	var plain map[string][]string = Header{"A": {"b"}} // as the platform's header fields take it
	back := Header(http.Header{"A": {"b"}})
	if plain["A"][0] != "b" || back["A"][0] != "b" || http.Header(back)["A"][0] != "b" {
		t.Error("the header types do not convert to each other")
	}
}

func cloneMap(m map[string][]string) map[string][]string {
	if m == nil {
		return nil
	}
	out := make(map[string][]string, len(m))
	for k, v := range m {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// TestHttpliteDoesNotImportNetHTTP: the point of the package.
func TestHttpliteDoesNotImportNetHTTP(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool")
	}
	out, err := exec.Command(goTool, "list", "-deps", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == "net/http" || strings.HasPrefix(pkg, "net/http/") {
			t.Errorf("package httplite depends on %s", pkg)
		}
	}
}
