# Findings — opencode-ensemble research

## Research: What can skillgrid-teams (squad plugin) and the skillgrid UI learn from opencode-ensemble?

**Decision served:** scope of the planned `skillgrid-teams` plugin (the squad plane:
`plugins/opencode/skillgrid-squad.ts` + `/teams/tasks*` HTTP routes + `TeamsPage`)
and the embedded Dashboard.
**Type:** technical (integration design, grounded in a comparable shipped product).

### What opencode-ensemble is (verified this run)

An npm OpenCode plugin (v0.18.0, MIT, 233★ / 32 forks as of 2026-10-07) implementing
"agent teams": a lead agent creates a team, spawns teammates in their **own OpenCode
sessions** (via the public SDKs `@opencode-ai/sdk` / `@opencode-ai/plugin`), and they
coordinate through peer **messaging** and a shared **task board** with `depends_on`
edges [1][2]. State is SQLite (WAL) in the plugin process; a TUI companion
(SolidJS + OpenTUI, peer deps from the host) plus a **mission-control dashboard at
localhost:4747** render live state [1][2]. 14 tools: 8 lead-only lifecycle
(create/spawn/shutdown/merge/cleanup/status/view/purge) + 6 shared (message,
broadcast, results, tasks list/add/complete, atomic claim) [2]. 835 tests, bun
runtime, TypeScript strict [2].

Key design points worth noting for skillgrid:

- **Push, not poll.** Messages are delivered into the lead's session
  (`promptAsync`); the skill's hard rule is "wait for `team_message`, don't poll
  `team_status`" [2][3].
- **Worktree isolation by default** per teammate (`worktree: false` opt-out for
  read-only explore agents); `team_merge` blocks on local file overlap;
  squash-merge to unstaged changes on cleanup [2].
- **Liveness machinery**: stall detection (low output tokens / no communication →
  escalate to lead), hard timeout watchdog, spawn circuit breaker (3 consecutive
  failures), rate-limit token bucket, crash recovery (stale busy → errored, orphan
  sessions aborted, undelivered messages redelivered), spawn rollback [2].
- **Plan approval mode**: `plan_approval: true` makes the teammate send a plan and
  wait for approve/reject via `team_message` before editing [2][3].
- **Model selection per role**: `modelsByAgent`, `modelPool` + rotate/random,
  `promptForModels` (user picks); resolution order explicit-param → byAgent →
  pool → default → host default [2]. The skill says always pass an explicit model
  because the host default "may be paid or misconfigured" [3].
- **System-prompt injection of team state** (member statuses, task counts) on
  every LLM call of the lead, with explicit compaction-safety (state preserved
  across compaction) [2].
- **Tool asymmetry**: teammates get only the 6 shared tools; lead-only tools are
  unavailable to them (sub-agent isolation via parent-chain tracking, max depth 10)
  [2][3].
- **Dashboard features** [1]: health ring; agent cards (status, current task,
  activity sparkline, timing) → detail drawer with full prompt, model, chat-style
  message history (markdown); task board with progress bar, collapsible status
  groups, dependency arrows; activity feed (chat bubbles with avatars); horizontal
  timeline strip (spawns/messages/completions/shutdowns); `j/k/Enter/Esc/?`
  keyboard nav; per-project grouping; live clock/session duration.
- **Bundled agent skill** (`skills/opencode-ensemble` + installable via
  `npx skills`) teaches the *lead* when to form a team, role defaults (scout =
  explore/read-only/cheap model; builder = build/worktree/strong model; qa;
  reviewer = explore/read-only), coordination patterns, lead checklists, and
  anti-patterns (spawn-because-hard, vague delegation, one-agent-per-file,
  parallelizing coupled edits, polling, trusting without review) [3][4][5][6].

### Gap analysis vs skillgrid's current/planned squad plane

Current state (verified in-repo): `plugins/opencode/skillgrid-squad.ts` (149 lines)
exposes 6 pull-queue tools → `/teams/tasks*` HTTP routes; `TeamsPage.tsx` exists in
the Dashboard; ADR-0020 kept the SDD ledger as the execution record and demoted
the teams tables to a "hybrid teams plane" the execution skills never call.

| Dimension | opencode-ensemble | skillgrid squad (today) | Learn / action |
|---|---|---|---|
| Coordination model | Peer messaging + shared board; push delivery | Pull queue only (`squad_pull_next_task`), status via output/review/done | **Add a message plane** (per-member inbox + broadcast + results) with push delivery into the lead's session; otherwise the "team" is two queues that don't talk |
| Task graph | `depends_on` with generated IDs, unblocks dependents | None (priority only) | **Add dependency edges** to the teams tasks schema — this is what makes sequencing visible instead of implicit |
| Lifecycle | create/spawn/shutdown/merge/cleanup/status/view, per-member sessions + branches | Task-level only (spawn/output/review/done) | The *member* concept (own session, own worktree/branch, status, model, role) is the missing half; pair it with the member session id so the UI can link through |
| Isolation | Git worktree per teammate by default, overlap-blocking merge | None | **Worktree-by-default** is the enabler of true parallelism. Locked constraint "no parallel branches" is a *development-process* rule for skillgrid's own repo — a team runtime that gives each member a worktree is consistent with it. This is the biggest fork: decide whether skillgrid-teams is a runtime (then worktrees) or a queue (then not) |
| Liveness | Stall detection, timeout watchdog, circuit breaker, crash recovery, redelivery, spawn rollback | Not present in squad plugin | These are the parts that make parallel agents *trustworthy* (core PRD theme: "done is a claim with evidence"). Even a minimal stall/timeout + crash-recovery set is worth stealing |
| Liveness in the lead | Team state injected into lead system prompt every call, compaction-safe | `skillgrid-compaction.ts` already owns start-injection and compaction | **Reuse the existing injection seam** (ADR-0027 CLM context hook) to inject team status — this is exactly the kind of state that survives compaction, and skillgrid has the seam ensemble lacks in a principled way |
| Plan approval | `plan_approval: true` → plan-then-edit gate via messaging | Review plane exists but is post-hoc (spec_compliance/code_quality after output) | A **pre-edit plan gate** on risky tasks is cheap to add (one status transition) and matches skillgrid's evidence culture |
| Model per role | modelsByAgent / pool / promptForModels | Not in squad | Optional; skillgrid's LLM provider (ADR-0023) already has a catalog — a `model` field on spawn + a by-role map is the minimal steal |
| Read-only roles | `explore` + `worktree: false` for scout/reviewer | No role concept | Role = (agent, worktree, model, allowed tools); makes prompt authoring and the UI both simpler |
| Dashboard | Mission control: health ring, agent cards + sparklines + drawer w/ message history, task board w/ dependency arrows, activity feed, timeline, keyboard nav | `TeamsPage` (status list) | See UI section below |
| Skill | Bundled skill with when/when-not, role defaults, patterns, checklists, anti-patterns | `using-skillgrid` already routes fan-out vs teams; `parallel-code-review` is a fan-out | **Write a `skillgrid-teams` skill** in the same shape: "Use when investigation needs debate / work is divisible into independent slices; do not when coupled" + role defaults + lead checklists. It's the difference between a tool set and a discipline |

### What the UI (Dashboard) can learn

The ensemble dashboard is a strong reference for a **TeamsPage upgrade**:

1. **Agent cards, not just a task table** — one card per member: status, current
   task, activity sparkline, timing; click → drawer with full prompt, model, and
   chat-style message history with markdown. Skillgrid's store already has the
   ingredients (member status, task rows, output markdown, session_events for the
   sparkline data) — the rendering pattern is the steal [1].
2. **Task board with dependency arrows + collapsible status groups + progress
   bar** — only possible once `depends_on` exists (action above).
3. **Horizontal timeline strip** (spawns / messages / completions / shutdowns) — a
   single cheap component that answers "what happened and when"; session_events
   already gives the events.
4. **Health ring** in the header (derived from stall/timeout/recovery state) —
   pairs with the liveness machinery.
5. **Keyboard nav** (`j/k`, `Enter`, `Esc`, `?`) — the skillgrid Dashboard already
   targets WCAG 2.1 AA; a keyboard-nav pass on TeamsPage is consistent.
6. **Per-project grouping** — relevant once central-PG (ADR-0024) makes multi-project
   views real; leave a seam now, don't build it.
7. **Do not copy:** the separate dashboard server on its own port (4747). Skillgrid
   already serves the SPA at `GET /` on 7438 with the existing SSE/tracker streams
   (SSE pattern shipped in kanban-milestones, W018). Teams live state should reuse
   the same SSE hub, not a second server [1].

### Tensions to resolve (ADR territory, not for the plugin to decide)

- **ADR-0020 vs a real runtime.** ADR-0020 demoted the teams plane because SDD
  execution never used it and two queues looked like one. If skillgrid-teams grows
  members/worktrees/messaging, it becomes the execution record for *team* runs —
  the ledger stays the record for *SDD* runs, and the ADR's "two queues" problem
  returns. A new ADR should name the boundary explicitly (e.g. "SDD runs: ledger;
  team runs: teams tables; the Dashboard shows both, each read from its owner").
- **Push delivery needs a prompt seam.** Ensemble uses `promptAsync` into the lead
  session. Skillgrid's OpenCode plugin already does `client.session.prompt`
  (start-injection, compaction plugin) — the seam exists, but the v1 mode-switch
  bug ensemble hit (teammate message flips lead into build mode) is a real gotcha
  to test for [2].
- **Worktrees vs local-first.** `~/.local/share/opencode/worktree/**` permission
  allowlisting is a known install friction for ensemble; skillgrid's installer is
  the right place to write it if worktrees are adopted.

### Biggest caveat

Ensemble is young (v0.18, 106 commits, 3 open issues) and its coordination
guarantees are best-effort (stall *detection* escalates; it does not make work
converge). The parts that generalize to skillgrid are the *mechanisms*
(messaging, dependencies, liveness, worktree isolation, the dashboard patterns,
the skill) — not any claim that teams are better than fan-out. skillgrid's own
`using-skillgrid` already encodes the verdict-vs-investigation distinction
(fan-out → verdict on a known artifact; teams → consensus among hypotheses);
the teams plugin should be scoped to the latter.

## Sources

1. GitHub repo home + README, hueyexe/opencode-ensemble, main, accessed 2026-10-07.
   https://github.com/hueyexe/opencode-ensemble (badges show v0.18.0; 233★, 32 forks,
   106 commits, MIT).
2. README "Tools / Architecture / Configuration / Model Selection / Known
   limitations" sections (same URL as [1]).
3. `skills/opencode-ensemble/SKILL.md`, v1.2.0, main, accessed 2026-10-07.
   https://raw.githubusercontent.com/hueyexe/opencode-ensemble/main/skills/opencode-ensemble/SKILL.md
4. `references/coordination-patterns.md` (same tree), accessed 2026-10-07.
5. `references/lead-checklists.md` (same tree), accessed 2026-10-07.
6. `references/anti-patterns.md` (same tree), accessed 2026-10-07.
7. In-repo (project context — shapes the question, not the evidence for ensemble
   claims): `plugins/opencode/skillgrid-squad.ts`, `mnemonic/internal/http/teams.go`,
   `skillgrid-ui/src/features/observe/TeamsPage.tsx`,
   `.skillgrid/artifacts/04-adr-0020-sdd-ledger-owns-execution.md`,
   `.skillgrid/specs/2026-10-07-replace-opencode-hooks-with-plugins/blueprint.md`,
   `.skillgrid/artifacts/00-prd.md`.
