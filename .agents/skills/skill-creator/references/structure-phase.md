# Phase 2: Structure — Steps and References

## 3. Phase 2: Structure — Steps and References

### 3a. Classify the Steps (deterministic boundary)

List every step the skill must handle. For each, apply the classification test:

> "If I ran this step twice with identical input, would the correct output be
> identical both times?"

| Step | Deterministic? | Action |
|------|---------------|--------|
| Parse Trivy JSON, filter by severity | Yes | Extract to `scripts/trivy-parse.mjs` |
| Classify an "unconfirmed" OWASP pattern | No | Keep as judgment prose |
| Check if state file matches spec zone | Yes | Extract to `scripts/state-drift-check.mjs` |
| Decide which test layer fits | No | Keep as judgment prose |

**Stop point:** If a step is *mixed* (deterministic parse + judgment
classification), split it: the deterministic part goes to a script, the
judgment part stays in prose. Name both.

### 3b. Map the Branches

A skill may have **branches** — distinct ways it can be used. Branches drive
where reference material lives:

- Reference used on **every** branch → keep in `SKILL.md`.
- Reference used on **one** branch only → move it behind a **context pointer**
  (a `references/` file pointed to by a one-line "if you're doing X, load
  `references/X.md`" line).

This is how you keep `SKILL.md` small: single-branch material never pays the
token cost on branches that don't use it.

### 3c. Write the Scripts First

Before drafting SKILL.md, write each script. For each:
1. Create `scripts/<name>.mjs` (ESM, Node 18+, no deps) or bash for git
   plumbing with heavy `awk`/`jq` usage
2. `chmod +x`
3. Run against a real input, confirm the output is correct and deterministic
4. Document the exit-code contract (0 = success/no-issue, 1 = issue found,
   2 = usage/parse error)

**Script conventions:**
- ESM (`.mjs`), Node 18+, no dependencies (use `node:fs`, `node:path`,
  `node:child_process`)
- Print a one-line verdict to stdout (e.g. `DRIFT: none`, `PASS: 36/36`)
- Print details to stdout when the verdict is non-zero
- Accept the project root as the first arg (default `.`)
- Data files (thresholds, budgets) live alongside the script or in `_shared/`

**Skip** if the skill is judgment-only. Not every skill needs a script.
**Scripts don't consume context** — they're executed, not read into the model's
window. A 200-line script costs zero tokens at load time.

**Repo lints ship with this skill.** Two validators run against the whole
`.agents/skills` tree; both are deterministic and exit 0/1/2:

- `scripts/skill-size-budget.mjs check .` — byte-budget ratchet (the P2
  check). Already covered below.
- `scripts/check-frontmatter.mjs check .` — the frontmatter contract: field
  set, field order, `name` = directory (kebab-case), `description` ≤ 1024
  chars, `metadata` block with quoted semver `version` and
  `part-of: skillgrid`, `disable-model-invocation` boolean when present.
  This is the F1 audit rule and a CI lint: it catches frontmatter drift the
  size budget cannot. Run it after any frontmatter edit.

### 3d. Progressive Disclosure

Keep `SKILL.md` under **500 lines**. The main file is a routing table, not a
manual. Claude loads only what it needs, when it needs it.

- **One level deep only.** Every reference links directly from SKILL.md.
  skill.md → advance.md → details.md means Claude may only preview the first
  100 lines of details.md. Flatten the chain.
- **Split by area** when a skill covers several domains (finance, sales,
  marketing). A question about revenue never loads marketing.md.
- Any file over **100 lines** gets a TOC at the top so Claude can jump to the
  section it needs from a 100-line preview.
- **Two size axes — lines AND bytes.** The ~500-line ceiling is the
  routing-table rule; `skill-size-budget.mjs` enforces the per-tier byte
  ceiling (`_shared/skill-size-budget.json`). Tables are dense: a 480-line
  skill with long decision tables can exceed its byte tier while under the
  line ceiling — run P2.

```
skill-name/
├── SKILL.md          ← routing table, steps, pointers (< 500 lines)
├── references/       ← loaded on demand, one file per concern
│   ├── pricing.md
│   └── edge-cases.md
└── scripts/          ← executed, not loaded (no token cost)
```

### 3e. The Spine Sections

An operational skill carries four standard sections in a fixed position:
**Input Contract** (the exact inputs the skill expects: file paths, config
keys, user parameters — one line per input; skip and cite when the input is
already a named section), **Decision Gates** (condition→action table before
the step walk, every row sourced from an existing rule), **Output Contract**
(the exact final output shape after the steps; skip and cite when the output
is already a named section), and **Anti-Patterns**
(`| Anti-pattern | Symptom | Fix |` at the tail, replacing 2-column
"Common Rationalizations"; Red Flags de-duplicated against it).
Input Contract + Output Contract together make the skill's behavior
reproducible — every input that changes the output is either pinned in the
Input Contract, scripted, or explicitly variable. Judgment-only and reference
skills skip all four deliberately.
Full contract: [references/spine-sections.md](references/spine-sections.md).

### 3f. Degrees of Freedom

Match the rigidity of each step to the consequence of getting it wrong.

| Level | When | Format |
|-------|------|--------|
| **High** | Brainstorming, code review, creative work | Plain instructions, no MUST/NEVER |
| **Medium** | Structured output with variation allowed | Template with optional fields |
| **Low** | Consequential, fragile steps (billing, migrations, deletes) | Exact script or checklist |

**Decision test:** "What happens if Claude does this differently?" Nothing
changes → high freedom. Something consequential → low freedom.

One skill can mix all three. An invoicing skill: the email is high freedom,
the invoice creation is low freedom. **Low freedom usually means a script, not
more prose** — scripts run the same on every model.

Claude 5.5 is more creative than older models — over-prescriptive skills
hamstring output. If you catch yourself writing ALWAYS or NEVER in all caps,
reframe: explain *why* the constraint exists.

### 3g. Thin Router, Thick Workflow

When `SKILL.md` crosses ~500 lines and most of the weight is a *procedure*
(the step-by-step walk) rather than *routing* (which branch, which gate, which
reference), split it:

```
skill-name/
├── SKILL.md              ← thin: frontmatter, branch table, Decision Gates,
│                           pointers. The router. (~200 lines)
├── workflow.md           ← thick: the step-by-step procedure, gates,
│                           error handling. The manual.
├── references/           ← on-demand detail
└── scripts/              ← deterministic steps
```

**When to apply:** the skill has a single dominant procedure (one main path
with minor branches) AND SKILL.md exceeds the 500-line ceiling AND the
procedure's prose is not already in `references/`. The test: can you point to
one contiguous block of steps that is 60%+ of the file? If yes, it is a
workflow, not a routing table.

**The pointer:** SKILL.md references the workflow with one line —
`Execute per [workflow.md](workflow.md).` The agent loads SKILL.md (the
router), sees the pointer, loads workflow.md only when it actually walks the
steps. This is progressive disclosure applied to the procedure itself, not
just to reference material.

**What stays in SKILL.md:** frontmatter, the branch/mode table, Decision
Gates, Input/Output Contracts, Anti-Patterns, the References index. These are
the *shape* — they determine which branch the agent takes. The *walk* (each
step, its error handling, its sub-gates) moves to workflow.md.

**What moves to workflow.md:** the ordered step sequence, per-step error
handling, the feedback loops, the detailed gate payloads. If the skill has
multiple distinct procedures (one per branch), each gets its own workflow
file: `workflow-plan.md`, `workflow-execute.md`.

**Anti-pattern:** a 50-line SKILL.md that says "read workflow.md" with no
branch table, no gates, no contracts. That is a pointer, not a router. The
skill still must decide *which* workflow to load — the routing intelligence
stays in SKILL.md.

**Source:** GSD-core's 70+ skills all follow this split (SKILL.md 0.5–10KB
as router, `workflows/*.md` 5–49KB as procedure). The pattern works at scale
because the router is cheap to load and the procedure is only loaded when
needed.

---

