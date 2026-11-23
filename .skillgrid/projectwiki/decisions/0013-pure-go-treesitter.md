# Tree-sitter in pure Go (gotreesitter), not a CGo binding

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The `code` tool needs tree-sitter AST extraction (the graphify pattern:
deterministic, local, every edge explained). Tree-sitter is written in C; the
common Go bindings (`tree-sitter/go-tree-sitter`, `smacker/go-tree-sitter`)
wrap the C runtime through CGo, which conflicts with ADR-0005's C/CGo-free
target build. The programmer's steer (2026-11-23): pure Go, but with
tree-sitter.

The predecessor (`../skillgrid/mnemonic`) already runs the answer:
`odvcencio/gotreesitter` (v0.55.1 there; current release v0.58.0) — a
ground-up pure-Go reimplementation of the tree-sitter runtime (parser, lexer,
query engine, incremental reparsing, arena allocator, external scanners,
tree cursor). Grammar blobs are extracted from upstream `parser.c` files by
`ts2go`, compressed, and embedded; 206 grammars ship in the registry;
standalone per-grammar packages and `grammars.DetectLanguage` exist.

## Considered Options

- **CGo binding** (`go-tree-sitter` / `smacker/go-tree-sitter`): mature and
  C-faithful, but a C cross-toolchain per target, no `go install` without a C
  compiler, bugs hidden from `go test -race` — and it breaks the ADR-0005
  target for the whole binary.
- **Pure-Go `gotreesitter`**: no C dependency at all; cross-compiles to any
  `GOOS`/`GOARCH` (incl. `wasip1`); the grammar blob is the only input;
  per-grammar standalone packages keep the binary lean; MIT. Trade-offs: the
  full 206-grammar catalog bloats the binary ~20 MB (v1 imports only the
  supported grammars); the default `grammars` catalog includes four copyleft
  grammars (caddy, disassembly, jq — GPL-3.0; nim — MPL-2.0), excluded via
  `-tags gotreesitter_no_copyleft` or per-grammar imports; a subset of
  grammars is still converging to C-faithful trees (34 certified via `ccore`
  at v0.58.0).
- **Pure-Go without tree-sitter** (go/ast + regex/indent tiers): the
  simplest build, but the weakest AST fidelity outside Go — it is the floor,
  not the plan.

## Decision Outcome

Chosen option: **`odvcencio/gotreesitter`** is the `code` tool's AST layer —
the predecessor's exact binding, version-bumped to the current release at
scaffold. v1 links only the supported grammars via standalone packages
(`grammars/go`, `grammars/typescript`, …), not the full catalog; the
copyleft grammars are excluded. No CGo anywhere in the default build:
ADR-0005 holds for the entire binary, and the CGo build-tag fork the
interview initially contemplated is not needed.

### Consequences

- Good: the C/CGo-free target survives the code tool — one build profile,
  full AST fidelity for every supported language.
- Good: the predecessor's 3-tier ladder (rich 6 → per-language node maps →
  regex floor) transfers as design on the same binding.
- Bad: a young runtime (v0.58.0; per-grammar convergence state varies) — the
  ladder's circuit breaker (panicking grammar quarantines to regex) plus
  per-language extraction test fixtures contain the risk.
- Bad: grammar blobs add binary weight even per-grammar — bounded by the v1
  language set; the regex floor keeps unrecognized extensions functional.

## Not adopted

- **CGo binding** — while the pure-Go runtime is plausible; revisit only if
  gotreesitter misses a load-bearing construct in a v1 language and no
  node-map fix covers it.
- **No tree-sitter** — regex/indent tiers are the floor, not the plan; the
  deterministic AST is the user's explicit requirement.

## Related

- Supersedes: none.
- Depends on: `decisions/0005-c-cgo-free-target-build.md` (the target this
  preserves), `sources/skillgrid-mnemonic-v2.md` (the predecessor's ladder on
  the same binding).
- Consumed by: the code tool's extraction layer (this change's blueprint).
