// Command ciaffected prints the packages of the module that the files
// named on its standard input (one a line, as `git diff --name-only` prints
// them) affect, one a line, or FULL when they all are. CI runs it on a pull
// request's changes to test only those packages: the ones the changed files
// belong to and, through their imports (their tests' too), every package
// that depends on them. A change that code cannot be traced through, such as
// go.mod or a workflow, affects all of them.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	var changed []string
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		if f := strings.TrimSpace(in.Text()); f != "" {
			changed = append(changed, filepath.ToSlash(f))
		}
	}
	pkgs, err := list()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ciaffected:", err)
		os.Exit(1)
	}
	affected, full := Affected(changed, pkgs)
	if full {
		fmt.Println("FULL")
		return
	}
	for _, p := range affected {
		fmt.Println(p)
	}
}

// list returns the packages of the module, as go list describes them.
func list() ([]Package, error) {
	cmd := exec.Command("go", "list", "-json=ImportPath,Dir,Deps,TestImports,XTestImports,EmbedFiles,TestEmbedFiles,XTestEmbedFiles,Module", "./...")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p struct {
			Package
			Module struct{ Dir string }
		}
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(p.Module.Dir, p.Dir)
		if err != nil {
			return nil, err
		}
		p.Package.Dir = filepath.ToSlash(rel)
		pkgs = append(pkgs, p.Package)
	}
	return pkgs, nil
}
