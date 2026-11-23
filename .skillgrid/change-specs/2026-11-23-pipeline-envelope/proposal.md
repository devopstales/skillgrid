# Proposal: Envelopes Only — Typed, Schema-Validated Pipeline Envelopes

> **STATUS:** `proposed` (2026-11-23, scoped down from the full pipeline-envelope proposal)
> Source of study: `../gentle-ai/` — `contracts/review-integration/v1|v2/schemas/`, `bench/journeys.go:29-97`, `bench/README.md:137-158`
> Scope: **envelopes only** — schemas, fixtures, the two existing emitters, one new gate renderer, one validator, one craft doc. No state machine, no `advance`/CAS/journal, no hooks, no drift-guard change in this pass.
> Status values per `_shared/rules/sdd-structure.md` § STATUS Banner.

## 1. What gentle-ai does with envelopes (findings)

**Output envelope** — one JSON object per machine call
(`bench/journeys.go:29-97`; schema at `contracts/review-integration/v2/schemas/status.schema.json`):

- `schema` / `contract` are **consts** (`gentle-ai.review-integration.status/v3` over
  `gentle-ai.review-integration/v2`); `additionalProperties: false` on every envelope.
- Readers **ignore unknown fields**, so older and newer envelopes both parse (forward
  compatibility) — stated verbatim in `bench/journeys.go:19` and `axis_damaged_store.go:318`.
- The envelope is the *only* machine shape: the agent "runs printed commands and carries
  printed tokens" (`docs/review-integration.md`, Quick path) and never reconstructs state from prose.

**Gate-result envelope** (`gentle-ai.review-gate-result/v1`,
`contracts/review-integration/v2/schemas/gate-result.schema.json`) — emitted by every delivery
gate, one shape for all of them:

```jsonc
{ "schema": "…gate-result/v1",
  "result": "allow | scope-changed | invalidated | escalated",
  "allowed": true,                    // schema-constrained to match `result` (if/then)
  "action": "continue | explicit-maintainer-action | review.status | repository-policy",
  "reason": "one line",
  "context": { },                     // free-form gate detail
  "next": { "reason_code": "…" } }    // where to go next
```

**Consent envelope** (`gentle-ai.review-integration.consent/v2`, `schemas/consent.schema.json`)
— a user gate made machine-shaped: `blocking: true`, exactly two choices
(`prefixItems` + `items: false`), each choice carrying its **exact invocation**, plus
`off_path: { note, command }` — the "how to stop instead" path.

**Blocks are measured** (`bench/README.md:137-158`, `bench/classify.go:287-306`): a denial is
*in-band* only if the emitted bytes contain a **runnable** continuation — an exact command or
exact edit; a template with `<placeholders>` does not count. The `next_action` /
`recovery_operation` / `collect.capture_operation` keys are looked up recursively
(`classify.go:216-217`) so the rule survives envelope reshapes.

## 2. Gap analysis (envelope-relevant only)

| # | Current state | Gap |
|---|---|---|
| G1 | Verdicts are machine JSON already, but untyped: `test/scripts/verdict-render.mjs` emits `{verdict, floor_area, floor_reasons, non_answers, needs_human, notes}`; `ship/scripts/ship-gates.mjs` emits `{qa, qa_blocks, qa_needs_override, review, review_needs_override, blocked, block_reason, findings}` | No `schema`/`contract`, no closed `result` vocabulary, no `allowed`, no `next` continuation — the two emitters speak two ad-hoc dialects that nothing validates |
| G2 | User gates are prose screens (`sdd-structure.md` § User gate contract: "one screen, readable in thirty seconds") | No machine shape for a gate: which choices exist, what each does, how to stop — is re-derived from prose each time |
| G3 | `verdict-render` is not bound to anything: its input is area *signals*, not files | A verdict is not bound to the digests of the artifacts it verified (gentle-ai binds every review to `subject_hash`); "the verdict for that acceptance spec" cannot be checked |
| G4 | No in-band rule anywhere in the library: a block may name a reason with no runnable continuation | The `out_of_band` failure mode gentle-ai measures is unmeasured here |
| G5 | No schema zone, no fixtures, no validator | Nothing stops an emitter from drifting; the 10-rule audit is prose-level |

Out of scope for this pass: `state-drift-check.mjs` resurrection, phase authority script,
journal/CAS, hooks, the 4 untested scripts (G6 in the full proposal), router rewiring.

## 3. The edit list

### 3.1 New files

**Schemas + fixtures** — new zone `.agents/skills/_shared/schemas/pipeline/` (in the skill
library, travels with it; mirrors gentle-ai's `contracts/` layout: frozen per version, fixtures
beside schemas):

| File | Contents |
|---|---|
| `gate-result.schema.json` | `skillgrid.gate-result/v1` (shape in §3.2) |
| `gate-consent.schema.json` | `skillgrid.gate-consent/v1` (shape in §3.3) |
| `phase-status.schema.json` | `skillgrid.phase-status/v1` — shape frozen now, **no emitter in this pass**; the authority script (later change) emits it, so the shape is already settled when it lands |
| `fixtures/gate-result-allow.json` | PASS verdict rendered by verdict-render's field mapping |
| `fixtures/gate-result-blocked.json` | FAIL floor + `next` with a runnable step |
| `fixtures/gate-result-scope-changed.json` | digest mismatch case |
| `fixtures/gate-consent-user-gate-slice.json` | two choices + off_path |
| `fixtures/phase-status-execute.json` | `kind: execute` sample |
| `fixtures/phase-status-gate-pending.json` | `kind: collect` with the banner-line input |

**Validator** — `_shared/scripts/validate-envelope.mjs` + `test-validate-envelope.mjs`:

- Usage: `node validate-envelope.mjs <schema.json> <envelope.json|->` → exit 0 valid, 1 invalid
  (findings on stdout as JSON), 2 usage. No dependencies — a subset validator covering exactly
  what these schemas use: required, `const`, `enum`, `additionalProperties` whitelist,
  types, `minLength`/`minItems`/`maxItems`, `prefixItems`+`items:false` (the two-choice
  pattern), the `^sha256:[0-9a-f]{64}$` / `maybe_sha256` patterns, one level of nested objects.
  (Full draft-2020-12 is not needed; the schemas stay within the subset on purpose.)
- **In-band rule check** (ported from `bench/classify.go`): when a validated envelope denies
  (`gate-result.allowed == false`; `phase-status.next_transition.kind == "stop"`), the
  continuation (`next.step` / `continuation.step`) must be non-empty, contain no
  `<placeholder>` template segments, and name a runnable step (a `node …mjs`/`git` command or
  an exact `file:line` edit). Templates do not count — that sub-rule is the load-bearing one.
- Test suite: every fixture validates; a deliberately broken copy of each fixture (missing
  `next`, template placeholder, unknown top-level field, `allowed`/`result` mismatch) is
  rejected with the right finding — the negative half first, per the library's own
  self-validation backstop.

**Gate renderer** — `_shared/scripts/gate-consent.mjs` + `test-gate-consent.mjs`:

- Same pattern as `verdict-render.mjs`: one JSON object on stdin → envelope on stdout, exit
  0/2. It **renders** the gate the calling skill has already reached; it reads no files and
  writes no state (envelope-only: no authority, no mutation).
- Input: `{ change, gate, headline, reason, file, line, required_value, off_path_reason? }`
  → output per §3.3. The two `choices` are generated from the gate name against the named
  gate vocabulary in `sdd-structure.md` § User gate contract
  (`blueprint-validation`, `owed-decision`, `user-gate-slice`, `qa-verdict`, `ship-decision`);
  an unknown gate is a usage error (exit 2) — free-form gates are a drift finding.

**Craft doc** — `_shared/craft/envelope-contract.md` (cited, not restated, per house style):

1. `schema`/`contract` consts; `additionalProperties: false`; readers ignore unknown fields.
2. **Frozen subject binding:** every gate envelope carries `subject` — `sha256` digests of the
   exact artifacts validated; a consumer echoing a different digest is `scope-changed`, not a
   re-verification.
3. **In-band continuation:** every denial carries a runnable step; templates don't count
   (the validator enforces this — prose is the spec, the script is the check).
4. One fixture per schema; emitters' output is schema-checked in their test suites.
5. The envelope is the verdict's machine shape; the human-readable screen/line is its
   rendering. An agent that reports a verdict the envelope doesn't carry has self-approved.

### 3.2 `skillgrid.gate-result/v1` (the verdict shape)

```jsonc
{
  "schema": "skillgrid.gate-result/v1",          // const
  "contract": "skillgrid.pipeline/v1",           // const
  "gate": "qa-verdict",                          // "qa-verdict" | "ship" | pre-commit gate names later
  "result": "allow | blocked | scope-changed | invalidated",
  "allowed": true,                               // constrained to match `result` (if/then, as in v2)
  "action": "continue | user-decision | fix-first | resume-pipeline",
  "reason": "one line",
  "subject": { "acceptance.feature": "sha256:…" },   // maybe_sha256 values; {} when unbound
  "next": { "reason_code": "qa-concerns", "step": "runnable continuation" },
  "context": { }                                  // free-form: the emitter's existing detail
}
```

### 3.3 `skillgrid.gate-consent/v1` (the user-gate shape)

Port of `consent/v2` — `blocking: true`, exactly two choices via `prefixItems` +
`items: false`, each choice `{ answer, label, effect, next_step }` with `next_step` an exact
action (the banner line to flip, or the `blocked(<phase>): <reason>` entry to record), plus
`off_path: { note, step }`.

## 4. Per-file edits

| # | File | Edit |
|---|---|---|
| E1 | `test/scripts/verdict-render.mjs` | Additive envelope fields to the existing stdout JSON (nothing moves — consumers keep working): `schema`, `contract`, `gate: "qa-verdict"`, `result`/`allowed`/`action`/`reason`/`next` per the mapping below, `subject` echoed from input. Accept optional input `subject: {<path>: "sha256:…"}` (the test skill hashes `acceptance.feature` + `blueprint.md` and passes them); default `{}`. Header comment: name the envelope. |
| E2 | `ship/scripts/ship-gates.mjs` | Same additive pattern: `gate: "ship"`, `result`/`allowed`/`action`/`reason`/`next`, `subject: { "report.md": sha256, "review.md": sha256-or-empty }` — the script already reads both files, the hashes are free. `block_reason` becomes `next.reason_code` (values reused, closed vocabulary). |
| E3 | `test/scripts/test-verdict-render.mjs` | Assert the envelope fields per verdict; validate one PASS and one FAIL output against `gate-result.schema.json` via `validate-envelope.mjs` (negative half: a stripped envelope is rejected). |
| E4 | `ship/scripts/test-ship-gates.mjs` | Same assertions for the ship gate; assert `subject` digests match the fixture change-dir files' actual hashes. |
| E5 | `test/SKILL.md` | Verdict step: one line — the verdict is rendered as `skillgrid.gate-result/v1` per `_shared/craft/envelope-contract.md`; input carries the `subject` digests. Output Contract names the envelope. |
| E6 | `ship/SKILL.md` | Gates step: one line — gate verdict is `skillgrid.gate-result/v1`; the human-override screen is a `gate-consent/v1` rendering via `_shared/scripts/gate-consent.mjs`. |
| E7 | `_shared/craft/deterministic-boundary.md` | One line in the agentic-script contract: a step that emits or validates an envelope cites `_shared/craft/envelope-contract.md`; the envelope is the machine shape of the verdict. |
| E8 | `docs/03-tools.md` | Script inventory: add `validate-envelope.mjs`, `gate-consent.mjs`; note the `_shared/schemas/pipeline/` zone and that the two emitters now carry envelope fields. |
| E9 | `_shared/rules/sdd-structure.md` | § User gate contract: one line — the machine shape of a gate screen is `skillgrid.gate-consent/v1` (cited), the screen is its rendering. No contract change. |

**verdict-render mapping** (E1):

| existing `verdict` | `result` / `allowed` | `action` | `next.reason_code` / `next.step` |
|---|---|---|---|
| `PASS` | `allow` / true | `continue` | `qa-passed` / "proceed to the next phase per the phase table" |
| `WAIVED` | `allow` / true | `continue` | `qa-waived` / "record the waiver in report.md § Gate Decision" |
| `CONCERNS` | `blocked` / false | `user-decision` | `qa-concerns` / "human decides: record the override in report.md and re-render, or fix the floor area" |
| `FAIL` | `blocked` / false | `fix-first` | `qa-failed` / "re-run area `<floor_area>` with passing evidence, then re-render" |

**ship-gates mapping** (E2): `blocked: true` → `result: blocked`, `allowed: false`;
`reason_code` reuses `block_reason` (`qa-failed` / `qa-critical-unresolved` / `qa-unverifiable`);
`qa-unverifiable` → `action: resume-pipeline` ("render the QA gate verdict first"), the others
→ `fix-first`. Review-side states stay advisory in `context`/`findings` — unchanged.

## 5. Work units

| WU | Content |
|---|---|
| WU1 | Schemas + fixtures (3 schemas, 6 fixtures) + `envelope-contract.md` craft doc |
| WU2 | `validate-envelope.mjs` + test (subset validator + in-band rule, negative halves first) |
| WU3 | E1–E4: both emitters + both test suites adopt the envelope |
| WU4 | E5–E9: one-line wiring in `test`/`ship` SKILL.md, craft cite, docs, sdd-structure cite; `gate-consent.mjs` + test |

WU1–WU2 land with zero behavior change (schemas + fixtures + validator only). WU3 is additive
JSON fields — no existing consumer breaks.

## 6. Verification

- [ ] Every fixture validates against its schema via `validate-envelope.mjs`; each broken-fixture negative case is rejected with the right finding.
- [ ] `verdict-render` outputs for all four verdicts validate; `next.step` is runnable for the two blocked cases (no `<placeholder>` segments).
- [ ] `ship-gates` output on the `2026-11-22-pi-worktree-plugin` change dir validates; `subject` digests equal `sha256` of the actual `report.md`/`review.md` bytes.
- [ ] `gate-consent` renders all five named gates; an unknown gate exits 2.
- [ ] No emitter changed an existing field's name or location (diff of old vs new output keys).

## 7. What this pass deliberately excludes

Phase authority (`phase-status.mjs` status/advance), CAS + journal, hooks, the
`state-drift-check.mjs` resurrection, router rewiring, the 4 untested scripts — all remain in
the full proposal's work-unit list, untouched. `phase-status.schema.json` is the only forward
piece: the shape is frozen now so the later authority script emits into an already-agreed
envelope.
