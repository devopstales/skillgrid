# ASSUMPTIONS

Durable project understanding. First file a reader opens. Tiers:
VERIFIED (observed facts), INFERRED (hypotheses, not yet verified),
LOCKED (ADR index — pointer rows only), Open Questions.

## VERIFIED

- The repo is `skillgrid-skills` (remote `git@github.com:devopstales/skillgrid.git`):
  a docs-first skill family — markdown skills under `.agents/skills/`,
  shared policy under `.agents/skills/_shared/`, repo docs under `docs/`.
- Skills are tested by `test-*.mjs` node scripts in each skill's
  `scripts/` dir; scenario evals live in `evals/`.
- `wt` (worktrunk) is the tier-2 worktree tool per
  `.agents/skills/_shared/work-unit-commits/README.md` § Tooling tiers;
  tier 1 is a harness-native tool, tier 3 is `git worktree add`.
- The worktree standard's constants (branch allow-list, `refs/skillgrid/`
  scratch namespace, 50/session cap, 10MiB / 200-file filters,
  `{{ repo_path }}/.worktree/{{ branch | sanitize }}` template) are the
  contract the `pi-worktree` plugin must implement (R13 contract-drift
  guard enforces the direction).
- Active change `2026-11-22-pi-worktree-plugin`: the plugin builds in
  `plugins/pi/pi-worktree` **inside this repo** — programmer override of
  the briefing's original "new sibling npm repo" assumption (2026-11-22).
- Active change `2026-11-23-mnemonic-core`: the new local-first memory
  binary `mnemonic` builds in `mnemonic/` **inside this repo** (Go 1.26,
  module path TBC); v1 scope is memory-core only; the CLI is the
  multi-tool form `mnemonic <tool> <verb>` with `memory` as the first tool
  (programmer-locked 2026-11-23; ADR-0003…0011). Briefing status: draft.

## INFERRED (HYPOTHESIS)

- The `plugins/` family pattern (one dir per harness plugin, shared
  conventions) will be formalized as the first plugin (`pi-worktree`)
  ships; the manifest/README/changelog conventions it sets become the
  family precedent.
- `wt` is installed on the programmer's machine (stated 2026-11-22);
  it was not on PATH in this session — re-verify at execution setup.

## LOCKED

| Decision | Path |
|---|---|
| Adopt the `.agents/` shared layout, the adversarial-authoring pattern, and the minimal grill-me skill from intent-driven-template | `decisions/0001-adopt-intent-driven-template-patterns.md` |
| Adopt a `git-discipline` skill as a separate first-class skill (not folded into `work-unit-commits`) | `decisions/0002-git-discipline-as-separate-skill.md` |
| One SQLite file per project — portable brain under `~/.mnemonic/` | `decisions/0003-per-project-sqlite-file.md` |
| Vectors live inside the SQLite file — BLOB + vec0 (pure-Go, spike-gated) + brute-force floor; no external vector DB | `decisions/0004-vectors-in-file.md` |
| C/CGo-free is the target build — pure-Go ONNX (spike-gated); CGo onnxruntime is the pre-decided fallback | `decisions/0005-c-cgo-free-target-build.md` |
| Local ONNX CPU model is the default embedder tier; API (Ollama-compatible) opt-in; hash-64 floor | `decisions/0006-local-onnx-default-embedder.md` |
| Single handler, two adapters — CLI + MCP over one Handler, byte-identical envelopes (K6 parity) | `decisions/0007-single-handler-two-adapters.md` |
| v1 = memory core only — code/web tools, serve, projectwiki, team_*, LLM curator, importer explicitly deferred | `decisions/0008-memory-core-only-v1.md` |
| Concurrency: WAL + busy_timeout only — no leases, no write daemon in v1 | `decisions/0009-wal-only-concurrency.md` |
| Multi-tool binary — `mnemonic <tool> <verb>`, per-tool MCP servers, top-level init/doctor | `decisions/0010-multi-tool-binary.md` |
| Degradation contract — `degraded: true` never silent; floors never block save/search | `decisions/04-adr-0011-degradation-contract.md` |
| Go module path — `github.com/devopstales/skillgrid/mnemonic` (same path as the old module; branch handover) | `decisions/04-adr-0012-go-module-path.md` |

## Open Questions

- Minimum `wt` version pin for the `WORKTRUNK_DIRECTIVE_CD_FILE`
  protocol (confirm exact release at build step 2, record in plugin
  README).
- Does the family want a shared `@devopstales/pi-*` manifest
  convention doc (first plugin sets the precedent)?
- Where does `wt` live? (not on PATH at spec time)
- Embedding + vec stack selection — research (2026-11-23, `findings.md`) named
  `knights-analytics/hugot` Go backend (GoMLX) and
  `ncruces/go-sqlite3` + `sqlite-vec-go-bindings/ncruces`; the v0.2 spike
  must demonstrate MiniLM end-to-end on the Go backend (opset + latency) and
  vec0+FTS5 in one CGO-free build.
