// Package skillsdocs checks the agent skills bundled with the CLI
// (cmd/mygo/template/shared/.agents/skills, which `mygo install-skills`
// puts in an app): their metadata, the repository docs they point to, the
// names of the ui and widget API they mention, and that every Go example in
// them is a whole file that type-checks against this repository's source. It
// also checks that the files docs/roadmap.md names as the evidence of work
// done are in the repository.
//
// A method is checked on its very type, so a skill writes
// widget.Form.SetAcceptButton, not AcceptButton; a bare method name is not
// checked.
package skillsdocs
