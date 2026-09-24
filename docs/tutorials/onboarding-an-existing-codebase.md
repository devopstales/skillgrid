# Onboarding an existing codebase

In this tutorial you will bring Skillgrid into a repository that **already has
code in it** — the realistic case, not a greenfield demo. You will onboard the
project, let the agent learn the stack, conventions, and test runner from what
is actually there, and run your first **interview → blueprint → slice** cycle
for a small focused change. By the end, the pipeline will know your
conventions, and it will use that knowledge every time it plans.

The key idea: **onboarding's real job is inherited code.** The typical target
is a codebase somebody else wrote, years old, with conventions to
reverse-engineer and documentation that is missing or stale. Skillgrid onboards
by reading what is actually there — not what a project "like this" would look
like — and plans the next slice **on top of reality**, against the constraints
the existing system already imposes. Reuse instead of regenerate.

---

## What you'll build

We will add a single `GET /health` endpoint to an existing web application.
The change is small enough that it never distracts from the real lesson: how
Skillgrid learns your codebase before it plans anything, and how that knowledge
shapes every later decision.

---

## Before you begin

- An agent host that loads skills from `.agents/skills/` (OpenCode, Kilo, or
  Cursor).
- **A real repo with code already in it.** It does not have to be the same
  stack as the examples; the steps apply to any project.
- The repo must be a git repository (if it is not, `git init` — Skillgrid's
  artifacts land in a tracked repo).

You do **not** need to run `skillgrid install` again if you have already done it
on this machine — the hooks and `~/.agents/` are machine-global.

---

## Step 1 — Start the agent in the repo root

From the repo root, start your agent and tell it to route:

```text
Use using-skillgrid for the work I want.
```

The router runs its config check: `.skillgrid/config.yaml` does not exist yet,
so it routes to `onboarding`.

---

## Step 2 — Onboard: detect, then confirm

Onboarding **detects** facts from the codebase using a strict source
precedence — **AGENTS.md/CLAUDE.md → `.skillgrid/config.yaml` → Mnemonic → git
remote → project files** — and **confirms** each one with you. It does not
guess.

Watch it read the project to detect:

- **Project name** — from the git remote or the repo name.
- **Stack** — from `go.mod` / `package.json` / `requirements.txt` / whatever is
  present.
- **Test runner** — from the test config or the build tooling.
- **Tracker** — default to `backlogmd` unless the repo clearly uses GitHub /
  GitLab / Jira.
- **Security tooling** — e.g. a `trivy` config.

> [!IMPORTANT]
> **The confirm step is blocking.** Onboarding will not write
> `config.yaml` until you confirm the detected facts. If it detects the wrong
> test runner, correct it *now* — `qa` reads `testing.runner` from this file
> every time it runs the gate.

When you confirm, it writes:

```text
.skillgrid/config.yaml     # project facts
AGENTS.md                  # the skillgrid block (merged, not overwritten,
                           # if AGENTS.md already exists)
```

Open `config.yaml`. Note `testing.runner` — that is the exact command `qa`
will use to run your suite. If your repo has multiple test layers
(unit/integration/e2e), confirm `testing.layers` reflects them, because `qa`
selects the **highest layer that fits** each behavior.

---

## Step 3 — Build the domain model

Onboarding (and the skills that follow) maintain the **domain model** — the
project's vocabulary and its architectural decisions:

```text
.skillgrid/artifacts/
├── 01-business-terms.md     # business glossary
├── 02-technical-terms.md    # technical glossary
├── 03-adr-index.md          # in-force ADR index
└── 04-adr-*.md              # individual ADRs
```

In a mature repo you may already have some of this. If not, the files are
stubs the agent fills in as terms resolve. This matters for a brownfield repo:
the agent must use *your* vocabulary — the name you call the core object, the
pattern your routing follows — not invented synonyms. When a term resolves or
a hard-to-reverse decision is made, the agent records it via
`architectural-decision-records`.

> [!TIP]
> In a large inherited codebase, point the agent at the parts you care about
> first. "We are adding X, focus on the area around <path>." The router will
> scope the domain-model work to what the change touches rather than trying to
> gloss the whole repo in one pass.

---

## Step 4 — Interview the change on top of reality

Ask for the change:

```text
Add a GET /health endpoint that returns 200 with {"status":"ok"}.
```

`interviewing` now grills the ambiguity — but the questions are **grounded in
your codebase**. Because onboarding detected your stack and the agent has read
your routing pattern, the questions come out like:

- "Your routes are registered in `src/routes/index.js` — should the health
  route follow that pattern?"
- "Your handlers return via the `respondJson` helper — should health use it,
  or a raw write?"

This is the difference from greenfield: the interview is **constrained by what
the code already does**. As the terms resolve ("route", "handler", "health
check"), they are recorded into the glossary.

When the clarity gate passes, you approve.

---

## Step 5 — Blueprint against the existing conventions

`writing-blueprints` writes `blueprint.md` to the change folder. Because the
agent read your codebase map and glossary, the plan references **your actual
paths and patterns**:

```text
.skillgrid/specs/YYYY-MM-DD-health-endpoint/blueprint.md
```

Open it. The **Must-Haves** are derived goal-backward from the requirement, and
the file-structure section maps to *your* existing layout — the same router,
the same helper, the same test convention. The agent is not restructuring your
repo to match a template; it is adding one slice to the system you have.

If the agent hits a decision nobody made yet (e.g. "does health go in the
public or authed route group?"), the **Owed-Decision Gate** forces it to either
resolve it with you or record it as an `ASSUMED` blueprint. Either way, the
decision is in a file — not buried in code.

---

## Step 6 — Slice and approve

`slicing` produces `tasks.md` with vertical tracer-bullet tickets, dependency
edges, and waves. For a single endpoint this classifies as `trivial` and
produces a light `tasks.md` with the waiver recorded in `briefing.md`.

The **approval gate** fires: **Go** or **Revise**. Say **Go**, and the
execution skill runs the ticket(s) in a fresh context, TDD red→green, committing
with the `[skillgrid-context]` block.

From here the loop is identical to a greenfield project:

```text
apply → qa → review → ship → reflect
```

---

## What you've learned

- How `onboarding` **detects then confirms** — it reads the real codebase,
  never guesses, and blocks until you confirm the facts.
- How the source precedence works: **AGENTS.md/CLAUDE.md → config → Mnemonic →
  git remote → project files**, first answer wins.
- How `testing.runner` in `config.yaml` is the exact command `qa` uses, so a
  wrong detection now is a wrong gate later.
- How the **domain model** (glossary + ADRs) makes the agent use *your*
  vocabulary on inherited code.
- How the interview, blueprint, and slice are all **constrained by the existing
  code** — file paths, patterns, and conventions come from your repo, not a
  template.
- How the **Owed-Decision Gate** catches a load-bearing choice on a brownfield
  change and writes it to a file instead of letting it live in code.

---

## Related

- [Your first change](your-first-change.md) — the full greenfield loop from
  install to reflect.
- [User guide — layout](../user-guide/01-layout.md) — what lives where in
  `.skillgrid/` and `~/.skillgrid/`.
- [User guide — concepts](../user-guide/08-concepts.md) — the domain model,
  ADRs, and the rest of the recurring ideas.
