# Worker brief template

Write the filled brief to `<R>/.orch/tmp/<name>-brief.md`. Start the worker with "Read <that path> and follow it."
Replace every `<...>` with an absolute value before you write the file:
- `<R>`: the main checkout, `/Users/lang/GolandProjects/github.com/lang315/mygo`;
- `<S>`: `<R>/.claude/skills/orchestrate`;
- `<branch>`: the task's branch name without `orch/`. It stays the same after an escalation, while `<name>` becomes
  `<name>-2`.
- `<orch>`: your own address, from `echo $HC_TERMINAL_ID` (preflight prints it). Never copy an example.

Heavy-lock and measure-size are allowed only by their absolute paths.

## Code worker (sonnet, opus)

```text
Task: <one sentence: what to change and why>.
Source: <run ID, PR #, or file:line>. Decisive lines (you have no gh; these are all you get):
  <the error lines from the CI log or the haiku report, quoted exactly; or "read <R>/.orch/reports/<haiku>.md">
Scope: change only <package(s)/files>. <"No exported identifiers." unless the owner approved an API plan; then: the approved API, verbatim>. <other limits>.
Hypothesis to confirm, not assume: <what the orchestrator suspects, with file:line>.
Steps:
<For a feature or API task, replace step 1 with: "Write the failing test for the done criterion first; show it
   failing.">
1. Reproduce on the unchanged code first. Put the exact command and its output in the report.
   If you cannot reproduce after 3 tries, stop and report stuck; do not fix blind.
   <For a CI-only bug, replace step 1 with: "No local repro exists; CI is the repro. Fix from the log lines above;
   the orchestrator pushes your branch and runs CI.">
2. Fix the root cause with the smallest change. Prefer the shared cause over special-casing a test.
3. Done criterion, all must pass:
   - <the reproducing command>
   - <the package tests, e.g. go test -count=1 ./<pkg>>
   - for ui or ui/widget: <S>/scripts/heavy-lock go test -race -count=<1, or 20 for a race> ./<pkg>
   - gofmt -l <touched dirs> prints nothing
   - <S>/scripts/heavy-lock GOOS=<os> go vet ./<pkgs> for darwin, linux and windows

Rules:
- Work only in your working directory, a git worktree on branch orch/<branch>. Commit your finished change there,
  in the style of git log.
- Never push, open PRs, git stash, rebase, reset, amend or remove worktrees. No agents or subagents.
- Never set MYGO_E2E and never go run an example or an app: they open windows on this Mac.
- Temp files go in <R>/.orch/tmp/<name>/, never /tmp.
- Run heavy commands (-race, go test ./cmd/mygo, go build ./..., gotip vet ./..., any GOOS=... vet or build)
  through <S>/scripts/heavy-lock. It takes only GO*/CGO_ENABLED/MYGO_* variables and go, gotip or gofmt.
- Keep commands simple: one command per Bash call, no cd, no $(...), no loops. If a command is refused,
  do not work around it: report it.
- Report: write <R>/.orch/reports/<name>.md with:
  - root cause;
  - files changed;
  - every verification command with its exit status and last output lines;
  - what you did not check.
  Then run exactly: hc send <orch> "<name> done: <R>/.orch/reports/<name>.md"
  If stuck, or the criterion still fails after 2 attempts, write what you found, then:
  hc send <orch> "<name> stuck: <R>/.orch/reports/<name>.md"
- Stay open after reporting: review findings may come back to you with hc send.
```

## Read-only worker (haiku)

```text
Read-only <CI triage of a finished run | search | summary>. Edit no tracked file. Never rerun, cancel or dispatch CI.
No agents or subagents. Repository lang315/mygo; pass --repo lang315/mygo to gh.
Question: <what to find out>. Known so far: <facts, run IDs, what another worker owns>.
Do:
1. <per run: failed jobs and steps (gh run view <id> --json jobs), --- FAIL lines of --log-failed,
   the first decisive error line of each, quoted exactly>
2. <root cause: read the code and workflow files involved>
3. Verdict per item: fixed / still failing / flaky / unknown, with evidence. Say what you did not verify.
You cannot use Write or Edit. Write the report (under 80 lines) with one Bash call:
<S>/scripts/write-report <name> <<'EOF'
...report...
EOF
Commands outside your allowlist are refused, not asked: do not work around a refusal; say so in the report.
Filter with pipes, one command per call, e.g. gh run view <id> --repo lang315/mygo --log-failed | grep -- '--- FAIL'.
for loops, $(...), redirects (>) and `cd ... &&` are refused: run one command per run, and pass paths instead of cd.
Never copy tokens or secrets.
Then run exactly: hc send <orch> "<name> done: <R>/.orch/reports/<name>.md"
If stuck: hc send <orch> "<name> stuck: <one line>"
```
