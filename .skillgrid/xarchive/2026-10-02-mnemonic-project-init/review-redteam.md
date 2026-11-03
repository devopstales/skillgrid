# Red Team Review — mnemonic-project-init

**Range:** `f739d371`..`17652346`  
**Lens:** Cross-cutting gaps after Standards / Spec / Edge / Verification / Security / Performance  
**Mode:** read-only; hunt only what merged findings missed

## Method

Re-read `review-range.diff` and the production init surfaces (`init_cmd.go`, `init_boot.go`, `init_ingest.go`, onboarding Step 9). Compared every candidate against `review-merged-for-redteam.md` and dropped anything already filed (sentinel clobber, CLAUDE-only boot path, symlink jail, SessionEnd, walk swallow, verification holes, perf double-open, etc.).

## What others covered well

Path jail (lexical), symlink escapes, marker mismatch duplication, non-fatal index/ingest contract, TDD/spec-zone process failures, and per-file Save cost are already on the board. No re-report.

## Missed gaps (integration / ordering / caller assumptions)

1. **Default `dir="."` becomes project name `"."`** — `writeBootFile` uses `filepath.Base(dir)` without Abs. Onboarding’s prescribed `skillgrid init` from the project root therefore stamps `Project: **.**` into the every-prompt sentinel. Tests always pass absolute temp dirs, so the seam never fired.

2. **`--docs` is Abs’d against process CWD, not the init target** — `ingestExtra` calls `filepath.Abs(raw)` while jail/`Rel` use the project `absDir`. `skillgrid init /abs/proj --docs README.md` (or any relative extra) from another cwd resolves the wrong file or false-jails. Lenses checked jail math and symlinks, not the cwd-vs-`dir` binding.

3. **Flags after the positional directory are silently dropped** — Go `flag.Parse` stops at the first non-flag. `skillgrid init myproj --docs docs/api.md` sets `dir=myproj` and leaves `--docs` unparsed; only `Args()[0]` is consumed. Help/verification argued about usage text, not argv shape.

4. **Onboarding commits AGENTS, then init mutates it again** — SKILL.md Step 8 commits config + AGENTS; Step 9 runs `skillgrid init`, which upserts preamble/sentinel (and may touch CLAUDE). The “deterministic finish” leaves a dirty tree after the chore commit. Standards caught *wrong* sentinel content on re-init; not this commit→mutate ordering.

5. **Default roots ∪ `--docs` double-Save and double-count `Ingested`** — defaults and extras are independent loops with no dedupe. `--docs README.md` or `--docs docs` after defaults upserts the same topic keys twice and increments `Ingested` per Save. Perf counted Save cost; nobody checked the overlap seam or the result contract.

6. **Warm re-init reports `indexed: 0` on success** — `FilesIndexed` is the delta for this Scan, not store inventory. A second successful init prints `indexed: 0` with empty Errors—the same shape QA ranked as “boot ok, index empty.” Perf noted warm Scan cost; edge noted error+partial → 0; not successful warm → false empty signal.

7. **Boot runs before project resolve/open** — `writeBootFile` always mutates the tree first; only then does `initRunIndex` / `OpenForDirectory` discover ambiguous or unopenable projects. Those failures are non-fatal (exit 0 + Errors), so monorepo/ambiguous cwd still gets a boot file and no index/ingest. Spec endorsed non-fatal *after* boot; nobody challenged boot-before-resolve as the ordering that makes partial init the default for resolve failures.

## Clean intersections

No additional findings on SessionEnd (edge), symlink write/read (security/edge), tracker/memory placeholder clobber (standards), or help-test broken verification (verif/spec).

## Deliverable

JSON findings array returned to the coordinator (no severities; coordinator grades).
