# Stage 6 — Comprehension Test (target: `blueprint.md` + referenced files)

Persona: `comprehension.md` (Naive Reader). Mode: REVIEW.
Reading list as referenced: `briefing.md`, `acceptance.feature`, and the cited
codebase files (all read for verification in this round).
Harness isolation: part 1 enforced by fresh subagent dispatch; part 2 enforced
(writes confined to `review/`); part 3 partial — per-project instruction files not
loaded as instructions (stated, not silent); part 4 enforced (local only).

## Findings

Severity: Nit
Affected scope: Task 2 Step 3 (line 276) and Task 3 Step 3 (line 324)
Description: The spec-zone ids "K1" ("the K1 transaction surface") and "K6" ("the K6
MCP seam does not apply") are used without definition or link. A zero-context reader
cannot resolve them from the blueprint or its Terms section; they exist only in the
cited code comments (`store.go:136` "K1 — the single per-file tx surface";
`main.go:2` "the K6 byte-identical seam").
Rationale: Self-containment vector — every custom term must be defined inline or
linked. Both references are decorative (the referent is named in the same sentence:
"the store's tx seam", "obs is CLI-only"), so the reader is not blocked, but the ids
are unresolvable without the spec zone. Fix: expand on first use ("the store spec's
tx-surface invariant (K1)") or add both to the Terms section.

## Paraphrase record (per section, zero-context)

- Header (Role/Tier/Build shape/Goal/Architecture): paraphrased cleanly — a
  no-daemon event-capture layer: per-host TS adapters exec a hidden CLI verb with a
  JSON envelope on stdin; Go normalizes (tool-type map, `<private>` strip,
  truncation, sensitive-path gate, sha256) and writes one table in the per-project
  SQLite store, auto-registering the session row in the same transaction.
- Terms, Must-Haves, Key links, One-way doors, Global Constraints, File Structure,
  Ingest envelope: paraphrased cleanly; `T-1..T-9` are checkable truths with
  backstop tags; the envelope JSON block is a self-contained contract.
- Tasks 1-9: each paraphrasable with the Spec pointer to `briefing.md` (R*/D* ids
  resolve there — R5/R8/D2/D6 cited). Steps are numbered RED/GREEN TDD with
  verification commands and commit messages.
- Threat Matrix: rows paraphrasable; the subprocess-exposure paragraph names its
  mitigations (stdin cap, per-host timeouts, fixed command strings).

Assumed-prerequisites / vague-explanation / cognitive-friction vectors: no other
section required a guessed term; error and edge behavior (exit codes, fail-open,
malformed JSON, PG dialect) is described predictably.
