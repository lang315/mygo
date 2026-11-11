# Routing reference

Use this with the checklist in SKILL.md. It comes from the 2026-11-10 trial: 34 past sessions analysed, then 3 real tasks.

## Roles

| Role | Launch | Does | Never does |
|---|---|---|---|
| haiku | `--model claude-haiku-5-5 --effort high`, `--setting-sources user`, `dontAsk` + `reader-settings.json` | Read-only work: CI triage (failed jobs and steps, `--- FAIL` lines, the first decisive error line); summarizing a finished CI run's failures (it cannot watch a running one); `gh pr diff --name-only` against intent; code and caller search; history and doc lookups | Edit tracked files, commit, push, rerun/cancel/dispatch CI, delete |
| sonnet | `--model claude-sonnet-5-5 --effort medium`, `dontAsk` + `worker-settings.json` | One package with a local automated test; the same mechanical fix applied across several packages; prose-only docs or skills fixes; a locale string with its own `go generate`; app sizes via `scripts/measure-size.sh` | Exported API, backends, `loop.go`, purego, shared size paths |
| opus | `--model claude-opus-5-5 --effort medium`, `dontAsk` + `worker-settings.json` | Real logic across two or more packages; Windows-only failures with no local repro; work in the hard-rule zones below that the owner has approved for a worker | Push, merge |
| orchestrator | this session, on the model, effort and permission mode the owner opened it with (`claude --dangerously-skip-permissions --settings …/orchestrator-settings.json`) | Specs, routing, the duplicate check, review of every diff, rerunning the tests, push, PR, CI, merge after the owner says yes, cleanup, log | Write the code itself |

Haiku is read-only. App sizes need builds, so they go to sonnet.

The model IDs are pinned (`claude-haiku-5-5`, `claude-sonnet-5-5`, `claude-opus-5-5`), never aliases. This routing
was calibrated on the 5.5 models. Changing an ID means checking the routing again: rerun a few logged tasks of
that role, and compare the outcomes in `.orch/log.md`. The reasoning behind the roles and efforts, from Anthropic's
model guidance, is in [model-guidance.md](model-guidance.md).

## Permissions

`dontAsk` refuses anything outside the allowlist instead of prompting, so no worker waits on a dialog. Read-only
Bash commands (`ls`, `cat`, `grep`, `git status`) pass without a rule.

- **Haiku:** `--setting-sources user`, `--permission-mode dontAsk` and `reader-settings.json`.
  - `--setting-sources user` keeps out the checkout's `.claude/settings.json` and `settings.local.json`. Both allow
    commands a reader must not run: `gh api` writes on the upstream, `gh pr *`, `git branch *`, `hc kill *`.
  - Allowed: read-only `gh` and `git` subcommands, `grep`/`rg`/`ls`/`cat`/`head`/`tail`/`wc`, `hc send`,
    Read/Grep/Glob, and `scripts/write-report`.
  - Denied: Write, Edit, `git push`/`commit`/`checkout`/`reset`, `gh pr merge`/`create`/`comment`, `rm`.
  - `~/.claude/settings.json` allows Edit and Write everywhere, and settings files merge. That is why Write and Edit
    are denied outright, and why reports go through `write-report <name>` with a quoted heredoc.
  - Tested 2026-11-10: Write and `git push --dry-run` were denied; `gh run list` and write-report worked.
    Without `--setting-sources user`, `git branch <new>` and `hc rm` ran; with it, both were refused.
  - Pipes work (`gh run view <id> --log-failed | grep -c FAIL`, `2>&1 | grep`, `| head`, `| wc -l`).
  - Refused: `for` loops, `$(...)`, redirects (`>`), and `cd … &&` (even with a `cd` rule). So the brief asks for one
    command per call, with paths instead of `cd`.
  - `git grep`, `sort`, `uniq`, `cut` and `tr` are allowed; `sort -o` is denied, because it writes a file.
- **Code workers:** `--permission-mode dontAsk` and `worker-settings.json`. A worker reports a refused command
  (task-brief.md).
  - Allowed: Read/Edit/Write/Grep/Glob, go, gotip, gofmt, `git add/commit/diff/status/log/show/mv/rm`,
    `mkdir`/`rmdir`, `hc send`, heavy-lock, measure-size.
  - Denied: `git push`, `git stash`, `git commit --amend`, `git worktree remove`, `gh pr merge`, `rm -rf`.
  - `go *` runs any Go program, so this is not a sandbox.
  - Tested 2026-11-10 with `-p`: Edit, Write, `go version`, `git commit` and `git mv` ran; `rm`, `git push --dry-run`
    and `gh pr list` were refused with no dialog.
  - The pre-push hook refuses pushes run inside a `mygo-wt-*` worktree, `git -C . push` included. It does not stop
    `--no-verify`.
- **Orchestrator:** `.claude/settings.json` has `ask` rules. They hold in every mode, bypass and `auto` included;
  tested 2026-11-10 and 2026-11-11. They cover:
  - `gh pr merge`;
  - merges through `gh api` on `repos/*/mygo` (REST `/merge`, `/merges`, `ccr/auto_merge`, GraphQL
    `mergePullRequest`);
  - `git push` to `main`, to `HEAD`, a bare `git push`, and a push with only a remote;
  - the same forms written `git <opts> push`, as in `git -C "$R" push origin`;
  - `bash`/`sh`/`zsh -c` commands that mention push.

  Notes:
  - `orchestrator-settings.json`, given with `--settings`, tells auto mode's classifier this repository's rules:
    what the orchestrator does every day (`go build ./...`, pushes of `orch/*`, PRs on the fork, worktrees), and
    what it must not (a `v*` tag, anything on the upstream). The classifier reads `autoMode` from the user's
    settings and from `--settings` only, never from a checkout's `.claude/`. Tested 2026-11-10:
    `claude --settings <file> auto-mode config` lists the entries.
  - `orchestrator-settings.json` also has `deny` rules, for what the orchestrator never does: a push of a tag
    (`refs/tags/`, `--tags`, a ref that starts with `v`), a force push (`--force`, `-f`, `+ref`, `--mirror`,
    `--all`), a push to `upstream`, any `gh` command that names egoist/mygo, `gh release`, and an edit of
    `.claude/`. A deny refuses with no dialog, so it never blocks a worker's `hc send`. Tested 2026-11-11 with
    `-p`, in bypass and in `auto`: 12 such commands were refused, and a push and a delete of an `orch/` branch, a
    PR list on the fork and `git fetch upstream` ran.
  - The limits: the rules match a command's text, so a script or another spelling passes them, and the edit rule
    stops the Edit and Write tools, not a shell command. In bypass nothing else checks a command, and the docs
    say that mode "offers no protection against prompt injection or unintended actions". In `auto` the
    classifier does; a dry run of a tag push passed it, so its hold on the real one is not shown.
  - `bypassPermissions` honors `ask` rules (docs: "Actions no mode auto-approves").
  - Tested 2026-11-10 under `--dangerously-skip-permissions`: all those forms were held, and a push of an `orch/`
    branch ran.
  - Tested 2026-11-10 with `-p` and `Bash(git *)` allowed: `git -C "$R" push --dry-run origin` was held, and
    `git -C "$R" push --dry-run -u origin orch/<name>` ran.
  - Not covered: a push made from inside a script or another program, e.g. Python's `subprocess`.
  - This skill's commit removes the two egoist/mygo merge allows that came from upstream. This fork never merges
    there.

## Hard-rule zones: opus at least, and the owner sees the plan first

- An exported identifier of `mygo`, `ui`, `ui/widget`, `i18n`, `ffi` or a plugin. Also `TestUIAPI`, `go generate ./ui`, and the skills that teach that API.
- A platform method: all of `darwin`, `linux`, `windows`, `web`, `fake` and `unsupported`.
- Threading in `loop.go` (`onMain`/`postMain`/`await`), and purego callbacks.
- Calls from `App.Run`, windows, the default menus, `init` or a backend method into opt-in code. Measure both apps.
- Generated files that must stay in sync: `internal/bridge/bridge.js`, `examples/*/src/mygo.ts`, `examples/native-module`, `internal/nativegen` glue, shaders, `packages/cli/index.d.ts`.

These are out of scope for workers unless the owner asks: release config (`release.yml`, `scripts/version.ts`), upstream sync of egoist/mygo, and anything published.

## Examples

| Task | Role | Why | Outcome |
|---|---|---|---|
| Linux ARM64 red on 9 main pushes plus Go next red (runs 42741374869, 42626173357, …) | haiku | Read-only CI triage | Pass. Found the cause: the H.264 decoder package `gstreamer1.0-libav` missing on ARM64 and in go-next, fixed by #459. Also found open PR #461 duplicating another task. |
| `TestListBuildInputs` fails depending on test order (cmd/mygo, run 42757887133) | sonnet | One package, local repro with `-shuffle` | Pass, PR #462. Reproduced before fixing. The real cause (`t.Setenv("HOME")` in `TestNatives` poisoning the `goEnv` cache) differed from the brief's guess. |
| Go tip vet: `bufio.Scanner` without `Err()` check in 4 packages (run 42626173357) | opus (was) → sonnet now | Same mechanical pattern across packages | Pass after one review round: a `t.Error` from an untracked goroutine could panic once the test had ended. Dropped as a duplicate of #461. |
| A race in one `ui/widget` test | sonnet | One package; `heavy-lock go test -race -count=20 ./ui/widget` (add `-run X` once the test is known) | from history |
| A test red only in the Windows CI shard, with no local repro | haiku triage, opus fix, you watch CI, haiku summarizes the finished run | Windows and CI-only flow in SKILL.md | from history |
| A new platform method (e.g. FileWatch on Windows) | opus, plan to the owner first | 6 backends plus `mygo` | from history |
| DataGridView in-cell editing (new widget API) | opus, spec to the owner first | Public `ui/widget` API | from history |

## Escalation

- The chain is exactly: haiku → sonnet → opus (medium) → ask the owner. Move a task one step up when any of these happens:
  - the worker reports stuck;
  - its done criterion still fails after its second attempt;
  - its diff reaches outside the brief's scope.
- Kill the old worker. Start the next one as `<name>-2` (then `-3`), with report `<name>-2.md`. The branch stays
  `orch/<name>`.
  - If the branch is usable, use the same worktree as `--cwd`.
  - Otherwise remove that worktree, then recreate it:
    `git -C "$R" worktree add "$R/../mygo-wt-<name>" -B orch/<name> origin/main`. For a PR fix, use
    `origin/<head>` instead.
  - Give it the brief and the failed report. Moving from haiku to a code role needs a code brief.
- A review finding is not an escalation. Send it back to the same worker with `hc send` (scannererr took one round).
- When a haiku triage ends in "needs a code change", run the checklist on it as a new task.
