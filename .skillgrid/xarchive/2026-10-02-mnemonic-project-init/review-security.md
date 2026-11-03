# Security Review — mnemonic-project-init

> Specialist: Security (OWASP + parallel-code-review/security.md)
> Range: `f739d371..17652346`
> Diff: `.skillgrid/sdd/2026-10-02-mnemonic-project-init/review-range.diff`
> Focus: path traversal / `--docs` jail, AGENTS.md writes, `memory.Save` ingest, symlink escapes
> Mode: read-only evaluation

Security Review Progress:
- [x] Step 1: Map entry points and trust boundaries
- [x] Step 2: Load the references this code needs
- [x] Step 3: Sweep for candidate issues
- [x] Step 4: Triage every candidate
- [x] Step 5: Report findings

## Entry points and trust boundaries

| Entry | Control | Sink |
|-------|---------|------|
| CLI `skillgrid init [dir]` | operator / agent cwd | `writeBootFile`, `RunCodeIndex`, `ingestPaths` |
| `--docs path` (repeatable) | operator / onboarding agent | `ingestExtra` → `os.ReadFile` → `memory.Save` |
| `--force` | operator | preamble rewrite in boot file |
| Project tree contents | **untrusted if repo is malicious** | default ingest roots + boot-file path selection |
| Onboarding SKILL.md | agent reads user-named docs paths | forwards as `--docs` |

Trust boundary the jail claims to enforce: **no read/write outside the project directory**. Lexical `filepath.Abs` + `filepath.Rel` + `HasPrefix(..)` is insufficient when the kernel follows symlinks on `Stat`/`ReadFile`/`WriteFile`.

Threat model (reachable): clone or open a crafted project, run `skillgrid init` (or let onboarding do it). No network auth surface in this diff.

## What was checked (and cleared)

| Candidate | Verdict |
|-----------|---------|
| Lexical `--docs /etc/passwd` / `../outside` | Jail rejects (`TestInitRejectsDocsOutsideProject`) |
| Nested symlink **files** under walked dirs (`docs/`, artifacts, `--docs <dir>`) | `WalkDir` + `d.Type().IsRegular()` skips symlink entries |
| Symlink **directory** as `--docs` root | `Stat` sees dir; `WalkDir` `Lstat`s root and does not descend — no leak (n=0) |
| Command/SQL injection via ingest | Reads bytes into `Save` only; no exec |
| New network/auth/CORS surface | None |
| Secrets committed in diff | None |

QA’s “path jail covered” claim is true for **lexical** outside paths only; it does not cover symlink escapes.

## Findings (highest impact first)

### 1. `--docs` jail bypass via in-project file symlink (read)

**[High] Symlink escapes lexical docs jail into host files (CWE-59, OWASP A01:2025 Broken Access Control, ASI02 Tool Misuse)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:113-154` (jail at `:118-122`; read at `:150` → `saveIngestFile` `:163`)
- **Path:** `--docs <symlink-inside-project>` → `filepath.Abs` (no `EvalSymlinks`) → `Rel` stays in-tree → `os.Stat`/`os.ReadFile` follow link → `memory.Save`
- **Impact:** Malicious repo plants `leak.md` → `/etc/passwd` (or SSH keys, cloud creds). Init persists host file contents into the project memory store under `init/docs/leak.md`.
- **Evidence:** Lexical Rel of an in-project symlink is `leak.md` (jail allows); `ReadFile` returns outside content. No `Lstat` / `EvalSymlinks` / `O_NOFOLLOW`.
- **Confidence:** Confirmed
- **Fix:** Resolve both project root and candidate with `filepath.EvalSymlinks` (or open with `O_NOFOLLOW`), re-run `Rel` on real paths, reject `ModeSymlink` before read; add `TestInitRejectsDocsSymlinkEscape`.

### 2. Default single-file ingest follows symlinks (read)

**[High] README/ASSUMPTIONS/ARCHITECTURE symlink reads host files (CWE-59, A01:2025, LLM02 Sensitive Information Disclosure)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:79-87` → `saveIngestFile:163`
- **Path:** default root `README.md` (etc.) is a symlink → `os.Stat` reports regular (target mode) → `ReadFile` follows → `memory.Save`
- **Impact:** Same as (1) without needing `--docs`. Walked trees are safer; these three single-file roots are not.
- **Confidence:** Confirmed
- **Fix:** Same confined-open helper for all ingest reads; reject non-regular after `Lstat`.

### 3. Boot-file write follows symlinks (write)

**[High] AGENTS.md/CLAUDE.md symlink overwrite outside project (CWE-59, A01:2025, ASI02)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_boot.go:91` (`WriteFile` boot path); `:145` (`ensureClaudePointer` writes `CLAUDE.md`)
- **Path:** crafted `AGENTS.md` → sensitive host path → `resolveBootPath` treats it as existing → `os.WriteFile` follows symlink → host file replaced with preamble/sentinel
- **Impact:** Destructive write outside the project tree when the operator runs init on a malicious checkout (classic malicious-repo local-tool attack).
- **Confidence:** Confirmed (WriteFile-follows-symlink reproduced)
- **Fix:** `Lstat` boot targets; refuse symlinks (or create exclusively with `O_EXCL`/`O_NOFOLLOW`); never follow links when writing AGENTS/CLAUDE.

### 4. In-project `--docs` can dump arbitrary files into persistent memory

**[Medium] No sensitive-path filter before memory.Save (CWE-200, LLM02, ASI06 Memory & Context Poisoning)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:90-97`, `:131-145`, `:157-185`
- **Path:** operator/agent `--docs .env` / `--docs .` / `--docs secrets/` → walk regular files → full content `Save` as `init/docs/<relpath>`
- **Impact:** Secrets already in the repo (or broad trees) become durable agent-retrievable memory. Onboarding explicitly forwards user-named paths as `--docs`. Jail only stops *outside* the project.
- **Confidence:** Confirmed (by design breadth; exposure is real)
- **Fix:** Allowlist doc-like extensions/roots for extras; deny `.env*`, `*.pem`, `*credentials*`, `.git/`, etc.; cap bytes per file; document that `--docs` is not a secret vault.

### 5. Unbounded `ReadFile` into Save

**[Low] Unbounded ingest read enables local memory pressure (CWE-400, LLM06 Unbounded Consumption)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:163-167`
- **Path:** huge regular file under default/`--docs` tree → full `[]byte` in process + store
- **Impact:** Local DoS / store bloat; needs a large file in-tree (or via symlink if (1)/(2) unfixed).
- **Confidence:** Confirmed
- **Fix:** Cap read size (e.g. 1–2 MiB); skip/error oversized files in `Errors`.

## Three-tier boundary notes

| Tier | Item |
|------|------|
| Always | Validate paths at the FS boundary with realpath/`Lstat`; do not trust lexical Rel alone |
| Ask first | Broad `--docs` directory ingest into long-lived agent memory (PII/secrets retention) |
| Never | — (no secrets in code; no eval) |

## Spec / test gap

Blueprint threat matrix lists path jail with only lexical outside-path tests. Symlink escape is an **implementation defect relative to the stated boundary** (“must resolve inside the project directory”), not a request to weaken the spec. Add RED tests for file-symlink `--docs`, symlink `README.md`, and symlink `AGENTS.md` write.

## Machine-readable findings

```json
[
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:118",
    "issue": "Docs jail is lexical only; file symlinks escape",
    "attack_or_exposure": "In-project symlink --docs reads host files into memory.Save",
    "fix": "EvalSymlinks both sides or Lstat-reject links; re-jail; test"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:79",
    "issue": "Default file ingest uses Stat/ReadFile following symlinks",
    "attack_or_exposure": "Symlink README/ASSUMPTIONS steals host file into memory",
    "fix": "Lstat require regular file; confined read helper for all roots"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:91",
    "issue": "Boot WriteFile follows AGENTS.md/CLAUDE.md symlinks",
    "attack_or_exposure": "Malicious repo overwrites host files via init write",
    "fix": "Refuse symlink boot targets; O_NOFOLLOW or Lstat before write"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:90",
    "issue": "--docs ingests any in-project path into durable memory",
    "attack_or_exposure": "Agent/user can persist .env or whole trees for later recall",
    "fix": "Allowlist doc paths/extensions; deny secret name patterns; size cap"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:163",
    "issue": "os.ReadFile has no size bound before Save",
    "attack_or_exposure": "Huge in-tree file exhausts memory or bloats store",
    "fix": "Enforce max bytes; skip oversized files with Errors entry"
  }
]
```

## Specialist summary

- **Findings:** 5 (3 High symlink boundary breaks, 1 Medium secret/memory retention, 1 Low size DoS)
- **Clean areas:** lexical outside-path rejection; no exec of ingested docs; WalkDir skips nested symlink files
- **Grade (lens quality):** A — entry points mapped, symlink behavior verified against Go FS semantics, false positives on directory-symlink/`WalkDir` dropped
