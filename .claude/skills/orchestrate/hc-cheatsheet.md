# Commands that worked

`hc --skill` is the full manual for hc. Start each Bash call with the variables, because the Bash tool keeps none:

```sh
R=/Users/lang/GolandProjects/github.com/lang315/mygo; S=$R/.claude/skills/orchestrate
```

## Step 0 (is someone already on it?)

```sh
gh pr list --repo lang315/mygo --state open
gh pr list --repo lang315/mygo --state merged --limit 15
git -C "$R" branch -r --sort=-committerdate | head -20    # other sessions' branches (fix/, sync/, feat/, dbg/)
hc ls                                                   # other sessions' live workers and their folders
```

## Address and state

```sh
echo $HC_TERMINAL_ID          # your address (e.g. t24); workers report to it
hc ls                         # all live terminals, including other sessions' work; touch only yours
hc get <t>                    # one terminal: state, status, exit_code
```

## Start a worker

```sh
# code worker (sonnet / opus), in its own worktree
git -C "$R" worktree add "$R/../mygo-wt-<name>" -b "orch/<name>" origin/main
hc run --bg --name <name> --cwd "$R/../mygo-wt-<name>" claude --add-dir "$R/.orch" \
  --settings "$S/worker-settings.json" --permission-mode dontAsk \
  --model claude-sonnet-5-5 --effort medium "Read $R/.orch/tmp/<name>-brief.md and follow it."

# opus code worker, in its own worktree
hc run --bg --name <name> --cwd "$R/../mygo-wt-<name>" claude --add-dir "$R/.orch" \
  --settings "$S/worker-settings.json" --permission-mode dontAsk \
  --model claude-opus-5-5 --effort medium "Read $R/.orch/tmp/<name>-brief.md and follow it."

# read-only worker (haiku)
hc run --bg --name <name> --cwd "$R" claude --setting-sources user --settings "$S/reader-settings.json" \
  --permission-mode dontAsk --model claude-haiku-5-5 --effort high "Read $R/.orch/tmp/<name>-brief.md and follow it."
```

- Flags after `claude` reach claude unchanged. The model was confirmed through `modelUsage`, and the effort
  through the banner.
- Right after spawning, check the banner: `hc read <name> --lines 0 | grep -E ' with [a-z]+ effort'`.
  - It must show `Haiku 5.5 with high effort`, `Sonnet 5.5 with medium effort` or `Opus 5.5 with medium effort`.
  - If there is no banner yet, run `hc wait <name> --timeout 5` once, then read again.
  - Still no banner, and the screen shows the worker at work: the banner scrolled off. Check the flags it runs
    with instead: `pgrep -fl -- '<name>-brief.md'` must show its template's `--model` and `--effort`.
  - Still no banner, and no work: run `hc get <name>`. If `first_message.sent` is false, or the screen shows a
    trust dialog, tell the owner. Never press keys yourself.
- `--add-dir` and `--allowedTools` take several values: one of them right before the prompt eats the prompt.
  Follow them with another flag, and keep the prompt last.
- The brief lives in a file, so that shell quoting cannot change it.
- `--add-dir "$R/.orch"` lets a worktree worker read its brief and write its report into the main checkout.

## Wait and talk

```sh
hc wait --children all --timeout 570     # run with a Bash timeout of 600000
hc wait --children any --timeout 570
```

- The wait returns `reason: message` when a report arrives. It returns `timeout` when the time runs out, which is
  a check-in, not a failure. Wait again until no child is working.
- Reports arrive as input: `<hc from="t2k listinputs">listinputs done: <path></hc>`.
- `hc send <t> "<review finding>"` gives a worker another round in the same context.
- `hc send` wraps the text in `<hc from=...>`, so a slash command sent this way does not run.
- `hc read --lines 0 <t>` shows the visible screen: a permission dialog, or the banner with the effort
  (`Haiku 5.5 with high effort`, `● high · /effort`).

## Stale heavy.lock

A long-silent worker may be waiting on `$R/.orch/heavy.lock`; heavy-lock prints "waiting" once. The holder's pid is
in `heavy.lock/pid`. If `kill -0 "$(cat "$R/.orch/heavy.lock/pid")"` fails, the lock is stale. Then run
`rm "$R/.orch/heavy.lock/pid"; rmdir "$R/.orch/heavy.lock"` and log it.

## Slow commands (keep yourself reachable)

```sh
hc run --bg --name cmdtest --cwd "$R/../mygo-wt-<name>" -- "$S/scripts/heavy-lock" GOFLAGS=-trimpath go test -count=1 -timeout 30m ./cmd/mygo
hc wait cmdtest --timeout 570; hc read cmdtest --lines 15; hc rm cmdtest
```

## PR, CI and merge (always `--repo lang315/mygo`; never the upstream)

- **The `pull_request` run** is the gate: the affected packages on macOS, Windows and Linux, every package on
  Windows ARM64. It skips Linux ARM64 and, on Linux, the cmd and ui shards.
- **A dispatched `ci.yml` on the branch** is the full run: every package, all three OSes, Linux ARM64. Only for
  the cases step 5 of SKILL.md names.

```sh
git -C "$R" diff --stat origin/main...orch/<name>
git -C "$R" push -u origin orch/<name>          # from the main checkout; the hook refuses worktrees
gh pr create --repo lang315/mygo --base main --head orch/<name> --title "<title>" --body-file "$R/.orch/tmp/<name>-pr.md"
gh workflow run ci.yml --repo lang315/mygo --ref orch/<name>      # the full run: only for the cases of step 5
gh run list --repo lang315/mygo --branch orch/<name> --limit 4 --json databaseId,event,status,conclusion,headSha
hc run --bg --name ci-<id> -- gh run watch <id> --repo lang315/mygo --exit-status --interval 60
hc wait ci-<id> --timeout 570; hc get ci-<id>; hc rm ci-<id>    # exit_code 0 = green
gh run view <id> --repo lang315/mygo --json status,conclusion   # check before believing a watcher's exit
gh run view <id> --repo lang315/mygo --json jobs --jq '.jobs[]|select(.conclusion=="cancelled")|"\(.name) \(.steps|length)"'
gh run rerun <id> --repo lang315/mygo --failed                  # a cancelled job with 0 steps: it got no runner
gh run view <id> --repo lang315/mygo --json attempt,status      # after a rerun, before the watcher
gh pr diff <n> --repo lang315/mygo --name-only
# only after the owner's yes:
gh pr merge <n> --repo lang315/mygo --merge
git -C "$R" worktree remove "$R/../mygo-wt-<name>"; git -C "$R" branch -D orch/<name>
git -C "$R" push origin --delete orch/<name>; git -C "$R" pull --ff-only
```

- A `gh run watch` once exited 1 while its run was still `queued`. Confirm with `gh run view` before acting on a
  watcher's exit code.
- A watcher started right after `gh run rerun` exits at once with the old attempt's result. Start it when
  `attempt` has gone up and `status` is `in_progress`.

## Blocked worker

- Workers run in `dontAsk`: a refused command comes back in the report, not as a dialog.
- Status `blocked` is then a dialog of another kind (trust, login). Read it: `hc read --lines 0 <t>`.
- Do not answer it with `hc keys`, and do not widen the settings files. Log it and tell the owner; the owner
  answers it in the pane (`hc attach <t>`) or has you kill the worker.

## Cleanup

```sh
hc kill <t>; hc rm <t>        # only terminals you started
```
