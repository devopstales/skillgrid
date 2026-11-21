# Code Quality

Duplication and complexity via jscpd, plus the coverage / mutation / lint / typecheck / P0 / P1 gates from config. This area answers "is the code clean, or did the change add maintenance cost?"

**When this area matters most:** a pure refactor or rename — that is the change shape where duplication is most often introduced or revealed. For any change, run at least the exact-clone pass.

## jscpd

Copy-paste detector across 220+ languages. Run it per `skillgrid:jscpd` — the `ai` reporter is the compact, token-efficient output for agents.

### The passes (run in this order, one family at a time)

1. **Exact clones** (default) — the token sequences are identical. An exact clone is almost always a real copy-paste. This is the only pass you gate CI on by default.

   ```bash
   npx jscpd --reporters ai --ignore "**/node_modules/**,**/dist/**" <target>
   ```

2. **Renamed clones** (Type-2, noisier) — copies that differ only in identifiers/literals. A *lead to read*, not a defect. Raise `--min-tokens` (identifiers ignored → a short block is mostly placeholders).

   ```bash
   npx jscpd --reporters ai --ignore-identifiers --min-tokens 70 <target>
   ```

3. **Near-miss clones** (Type-3, noisiest) — copies with a few edited lines, or functions rewritten with the same shape. Keep the settings tight.

   ```bash
   npx jscpd --reporters ai --max-gap-lines 1 --similarity 0.85 <target>
   ```

**Target:** `duplication.target` from config (default `changed-files` — only the files this change touched). `duplication.formats` limits languages; `duplication.ignore` adds globs.

### Reading the output

- A clone line is `path/file.ts:10-25 ~ other.ts:42-57`. Suffixes: `(renamed)` = Type-2, `[~0.91 gap]` = Type-3 gap-merge, `[~0.75 ast]` = Type-3 structure. No suffix = exact.
- **`--summary`** appends the refactoring hotspots: top files/folders by tokens, with a `dup%` column (the share of each file's lines covered by detected clones). A large file with high `dup%` is the best refactoring target. Use it to decide where to look first.
- **`--health`** gives a 0–100 score (duplication, dead code, complexity); the lowest sub-score names where the codebase hurts most. `n/a` on a dimension means jscpd could not measure it (dead-code detection needs JS/TS/Python to be a real share) — skip that dimension, don't treat it as clean.
- **`--dead-code`** finds unused files, exports, symbols, and imports (JS/TS/Python) with a 0–100 confidence each. Report as SUGGESTION unless a threshold is set.

**Noisy-pass rules:** run the renamed/near-miss passes only after the exact clones are handled, one family at a time, with tight settings. Treat every `(renamed)` or `[~…]` line as a lead, not a defect. Never claim "N duplicates found" from a normalized run without saying which flags produced it.

### Refactoring the clones

Once exact clones are identified, refactor them per `skillgrid:dry-refactoring` (extract function, parameterize, module, constant, base class). For a broader "clean up this codebase" pass — duplication → dead code → largest/most-complex files, prioritized from `--health`/`--dashboard` and re-measured at the end — use `skillgrid:codebase-refactoring`.

### Gate

If `duplication.fail_on` is `> 0` and the duplication % (of `duplication.target`) exceeds it, the code-quality gate FAILs. If `fail_on` is `0`, duplication is advisory — report it, never block on it.

## The Other Code-Quality Gates

Run the commands from config. Each has a threshold under `quality:`.

| Gate | Command | Threshold (config key) | Default | FAIL when |
|------|---------|------------------------|---------|-----------|
| Coverage | `testing.coverage` | `quality.coverage_min` | 80 | Coverage < threshold |
| Mutation | mutation command (if configured) | `quality.mutation_min` | 80 | Mutation score < threshold |
| Lint | `commands.lint` | — | — | Any error (warnings are SUGGESTION) |
| Typecheck | `commands.typecheck` | — | — | Any error |
| P0 pass rate | run P0 tests | `quality.p0_pass_rate` | 100 | Any P0 test fails |
| P1 pass rate | run P1 tests | `quality.p1_pass_rate` | 95 | P1 pass rate < threshold |

**Mutation testing:** a surviving mutant is a concrete specification of missing coverage — treat it as a failing test. If no mutation command is configured, this gate is N/A (not a FAIL).

**A gate at threshold 0 is N/A.** It cannot FAIL. Do not render FAIL for a disabled gate.

## Writing the Report Section

```
## Code Quality
jscpd: <pass(es) run> on <target> → <dup%> (<exact N, renamed N, near-miss N>)
Hotspots: <top file/folder, dup%>
Health: <score> (dup <n>, dead <n|n/a>, cx <n>)
Dead code: <N> findings (<top>, confidence)

| Gate | Command | Threshold | Result | Status |
|---|---|---|---|---|
| Coverage | <cmd> | 80 | <pct> | PASS / FAIL / N/A |
| Mutation | <cmd> | 80 | <pct> | PASS / FAIL / N/A |
| Lint | <cmd> | — | <n errors> | PASS / FAIL |
| Typecheck | <cmd> | — | <n errors> | PASS / FAIL |
```
