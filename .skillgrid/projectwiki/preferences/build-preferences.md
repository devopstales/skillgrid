---
type: Preference
title: "Build preferences (local-first, C/CGo-free, multi-tool)"
status: stable
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md (2026-11-23 interview)"
  - ".skillgrid/config.yaml mnemonic.embedder (ollama, embeddinggemma:300m)"
---

# Build preferences (local-first, C/CGo-free, multi-tool)

Preferences the programmer expressed while locking the [[entities/mnemonic]]
design (2026-11-23 interview + follow-ups). Each is a default for future
tools in this repo, not a one-off.

- **Local-first, offline by default.** New tooling must work with zero network
  I/O on its default path; remote endpoints (embedder APIs, LLMs) are opt-in
  and must degrade to a local floor when absent. Source: v1 scope + ADR-9 in
  the briefing.
- **C/CGo-free is the target build.** When a capability has both a C-backed
  and a pure-Go implementation, the pure-Go one is the target default and the
  C-backed one is the fallback profile; the decision is spike-gated on
  opset/latency evidence, not assumed. Source: the ONNX exchange
  ("will ONNX be pure go so c and cgo free?") → ADR-3.
- **Local model over daemon dependency.** A bundled model file running
  in-process beats a daemon (e.g. Ollama) for the default path — a daemon is
  an acceptable *optional* path via a standard API tier, not a default
  requirement. Source: the Ollama exchange → ADR-4. (Note: the programmer
  does run Ollama locally — `embeddinggemma:300m` per
  `.skillgrid/config.yaml` — so the API tier must stay Ollama-compatible.)
- **CLI tools live under a multi-tool binary.** A tool that will grow
  siblings is addressed as `binary <tool> <verb>` from day one (here:
  `mnemonic memory …`, `mnemonic memory mcp`), with cross-tool commands
  (`init`, `doctor`) at the top level. Source: the CLI interjection
  ("mnemonic will contain multiple tools in the future").
- **SQLite file is the unit of portability.** One `.sqlite` per project,
  `cp`-able, no side stores; vector data lives inside the file, never in an
  external database. Source: v1 storage decisions → ADR-1/ADR-2.

## Related

- Supersedes: none.
- Consumed by: [[syntheses/mnemonic-core]] (every deferred tool inherits
  these defaults).
- Depends on: [[entities/mnemonic]], [[concepts/degradation-floor]]
  (the floor discipline these preferences enforce).
