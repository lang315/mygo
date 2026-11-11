# Model guidance behind this skill

Source: Addy Osmani, "Which Claude model should you build with? Fable, Opus, Sonnet, or Haiku", claude.dev, published
2026-11-09: https://claude.dev/blog/which-claude-model-should-you-build-with/ (fetched 2026-11-10).

## Key points

1. **The default is Opus 5.5 at `medium`.**
   - Move down to Sonnet 5.5 for well-scoped work: a bug fix, or a feature built against a spec.
   - Move down to Haiku 5.5 for the simplest high-volume work.
   - Move up to Fable 5.2 only when Opus at `high` falls short.
   - A clear spec is not enough on its own. A migration or a performance optimization can be well specified and
     still need Opus.
2. **Judge by cost per finished task, not price per token.** The hardest tasks dominate spend.
3. **Effort is a finer, cheaper dial than switching models.** Opus 5.5 on SWE-bench Pro, against `high`:

   | Effort | Score | Cost |
   |---|---|---|
   | `low` | about −8 points | about 1/3 |
   | `medium` | about −2.5 points | about 70% |
   | `xhigh` | about +1.4 points | 2.5× |

4. **Self-verification beats raising effort.** When a model falls short, take these steps in order:
   1. make sure it can check its own work, with a test, a build or a script;
   2. raise effort one level;
   3. move to the next model up, *starting at a lower effort than you used on the smaller one*.

   Running everything at `low` and re-running only the failures at `high` reached about 97% at $0.17 a task.
   Everything at `high` reached 95.3% at $0.29.
5. **Every turn resends the context.** Keep the expensive model's context lean.
6. **Switch models at natural breaks.** The prompt cache and thinking blocks belong to the model that wrote them.
   - A move down a tier loses the earlier reasoning.
   - Before switching, start fresh with a short written plan, or run `/compact`.
7. **Put subagents on small models for cheap-to-verify work** (search, logs, summaries), at the model's default
   effort. Keep code edits and judgment calls on the main model.
   - The article's table: well-scoped coding goes to Sonnet 5.5 at `medium`, then `high` for harder tasks.
   - Repeated agent tasks (investigation, review) go to Sonnet 5.5 at `medium`.
8. **Orchestrator plus cheap parallel workers.** An Opus 5.5 lead over Sonnet 5.5 workers cost about a third less than
   Opus alone, with about the same score. Skip the pattern for one chain of dependent steps, or for work that fits in
   one context window: the planning and merging cost more than they save.
9. **When routing is simple, let code pick the model**, not a model.
10. **Measure on your own tasks.**
    - Take real tasks, each with an automatic check, and run each at 2–3 effort levels in separate sessions.
    - Record the cost of each from `usage`, and plot score against spend.
    - Keep a multi-model setup only if it beats every single-model point.
    - Then run the winner in shadow on a slice of real work.

## How this applies to mygo

| Point | Rule here |
|---|---|
| 1. Opus default, move down when scoped | Routing checklist items 3–6 (SKILL.md): sonnet for one package with a local test, opus for API, backends and hard-rule zones. The orchestrator runs on the model the owner opens it with. |
| 2. Cost per finished task | Tuning (SKILL.md): count escalations, review rounds and oversized roles in `.orch/log.md`, not model prices. |
| 3. Effort before model | Roles (routing.md) pin efforts: haiku high, sonnet medium, opus medium. **Open for tuning:** the article's order adds "raise effort one level" before moving up a model; our chain goes straight to the next model. |
| 4. Self-verification first | Done criteria (SKILL.md): every code task has a repro and commands that pass or fail. A worker must reproduce before fixing (task-brief.md), and the orchestrator reruns the decisive commands. |
| 5. Lean main context | SKILL.md intro: the orchestrator reads reports, never raw logs; workers write reports to `.orch/reports/`. |
| 6. Switch at natural breaks | Each worker is a fresh session with a written brief. Escalation starts a new worker (`<name>-2`) with the brief and the failed report, never a mid-session `/model`. The orchestrator's model and effort are the owner's; a change mid-session gets a log row. |
| 7. Small models for cheap-to-verify work | Haiku is read-only: CI triage, logs, search, PR diff checks (routing.md). Code edits go to sonnet or opus; judgment, review and merge stay with the orchestrator. **Open for tuning:** the article runs such subagents at the model's default effort; ours run haiku at `high`. |
| 8. Lead plus cheap workers, not for dependent chains | At most 3 code workers in parallel. A task that is one dependent chain stays with one worker; work that fits in one context is not split. |
| 9. Code picks the model | The pinned launch templates (hc-cheatsheet.md) fix each worker's model and effort; the banner check confirms them. `scripts/preflight` reports the orchestrator's and checks neither. |
| 10. Measure on your own tasks | The log row records model and effort from the banner, and a session row the orchestrator's. Tuning compares outcomes every ~10 tasks. Changing a pinned model ID means re-checking the routing (routing.md). |
