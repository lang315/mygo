package psl

import (
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/net/publicsuffix"
)

var domains = []string{
	"", "com", "example.com", "co.uk", "example.co.uk", "amazon.co.uk", "books.amazon.co.uk",
	"www.books.amazon.co.uk", "uk", "foo.bar.uk", "jp", "kobe.jp", "city.kobe.jp", "www.city.kobe.jp",
	"foo.kobe.jp", "a.b.foo.kobe.jp", "ck", "www.ck", "foo.ck", "bar.foo.ck", "blogspot.com", "foo.blogspot.com",
	"github.io", "user.github.io", "a.b.user.github.io", "appspot.com", "foo.appspot.com", "cromulent",
	"foo.cromulent", "a.b.cromulent", "xn--fiqs8s", "foo.xn--fiqs8s", "中国", "foo.中国", "com.cn", "foo.com.cn",
	"org", "foo.org", "net", "foo.net", "io", "foo.io", "dev", "app.dev", "localhost", "foo.localhost",
	"example.local", "127.0.0.1", "::1", "192.168.1.2", "[::1]", "a.b.c.d.e.f.g.example.com",
	"EXAMPLE.COM", "Example.Co.Uk", ".example.com", "example.com.", "..", "a..b", "-", "a-b.example.com",
	"s3.amazonaws.com", "foo.s3.amazonaws.com", "execute-api.us-east-1.amazonaws.com", "x.execute-api.us-east-1.amazonaws.com",
	"herokuapp.com", "foo.herokuapp.com", "vercel.app", "foo.vercel.app", "pages.dev", "foo.pages.dev",
	"workers.dev", "foo.workers.dev", "netlify.app", "foo.netlify.app", "web.app", "foo.web.app",
	"co.za", "foo.co.za", "com.au", "foo.com.au", "co.nz", "foo.co.nz", "com.br", "foo.com.br",
	"ac.jp", "foo.ac.jp", "gov.uk", "nhs.uk", "foo.nhs.uk", "bd", "foo.bd", "bar.foo.bd", "mm", "foo.mm",
	"tokyo.jp", "foo.tokyo.jp", "city.nagoya.jp", "foo.city.nagoya.jp", "dyndns.org", "foo.dyndns.org",
	"cloudfront.net", "foo.cloudfront.net", "us-east-1.elasticbeanstalk.com", "foo.us-east-1.elasticbeanstalk.com",
}

// TestPSLIsPublicsuffix: for every domain, with the prefixes pages have, the
// public suffix and the registrable domain are those of
// golang.org/x/net/publicsuffix, which this package was copied from, while
// both hold the same snapshot of the list (go generate here brings this
// package ahead of x/net, until x/net is updated).
func TestPSLIsPublicsuffix(t *testing.T) {
	same := publicsuffix.List.String() == version
	if !same {
		t.Logf("x/net holds %q, this package %q: the comparison is skipped", publicsuffix.List.String(), version)
	}
	for _, d := range domains {
		for _, prefix := range []string{"", "www.", "a.b.c."} {
			domain := prefix + d
			gotS, gotI := PublicSuffix(domain)
			gotE, gotErr := EffectiveTLDPlusOne(domain)
			if !same {
				continue
			}
			wantS, wantI := publicsuffix.PublicSuffix(domain)
			wantE, wantErr := publicsuffix.EffectiveTLDPlusOne(domain)
			if gotS != wantS || gotI != wantI || gotE != wantE || (gotErr == nil) != (wantErr == nil) {
				t.Errorf("%q: PublicSuffix (%q, %v) EffectiveTLDPlusOne (%q, %v); publicsuffix has (%q, %v) (%q, %v)",
					domain, gotS, gotI, gotE, gotErr, wantS, wantI, wantE, wantErr)
			}
		}
	}
}

// TestEffectiveTLDPlusOne: what the list says of some domains, whichever
// snapshot the data is.
func TestEffectiveTLDPlusOne(t *testing.T) {
	for domain, want := range map[string]string{
		"example.com":                 "example.com",
		"www.example.com":             "example.com",
		"www.books.amazon.co.uk":      "amazon.co.uk",
		"a.b.c.example.org":           "example.org",
		"foo.bar.github.io":           "bar.github.io", // a private suffix
		"www.ck":                      "www.ck",        // an exception to *.ck
		"city.kobe.jp":                "city.kobe.jp",  // an exception to *.kobe.jp
		"a.foo.kobe.jp":               "a.foo.kobe.jp", // *.kobe.jp
		"foo.cromulent":               "foo.cromulent", // a TLD the list does not name
		"com":                         "",
		"co.uk":                       "",
		"127.0.0.1":                   "",
		"":                            "",
		"a.b.us-east-1.amazonaws.com": "b.us-east-1.amazonaws.com",
	} {
		got, err := EffectiveTLDPlusOne(domain)
		if err != nil {
			got = ""
		}
		if got != want {
			t.Errorf("EffectiveTLDPlusOne(%q) = %q, %v; want %q", domain, got, err, want)
		}
	}
}

// TestPSLDoesNotImportNetHTTP: the point of the copy: x/net/publicsuffix
// imports net/http/cookiejar, so net/http, which costs 2.2 MB of an app.
func TestPSLDoesNotImportNetHTTP(t *testing.T) {
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
			t.Errorf("package psl depends on %s", pkg)
		}
	}
}
