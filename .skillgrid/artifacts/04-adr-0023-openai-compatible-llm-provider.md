# Mnemonic chat LLM is OpenAI-compatible HTTP only; one client attaches all seams

---
status: "accepted"
supersedes:
date: 2026-10-02
---

**Related:** `04-adr-0016-second-brain-capability-layer.md` (fail-open floors and existing Ask/Dedup/Extraction/Dream seams). Spec: `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/`. Roadmap P2 item J in `08-second-brain-roadmap.md`. Closes FOLLOWUP task-029.

## Context and Problem Statement

Second-brain seams (`AskLLM`, `DedupLLM`, `ExtractionLLM`, Dream) already exist, but only tests attach them. Production returns “no LLM configured” and falls to deterministic floors. Code indexing already needs a real embedder — Ollama or an OpenAI-compatible external host (the ONNX path is weak). The open question: which chat-completion shape and install posture do we lock so one client wires every seam and install is a natural provider update rather than an optional bolt-on?

## Considered Options

- **One OpenAI-compatible Completer + natural Local/External install (chosen):** stdlib HTTP `POST {base_url}/chat/completions`; Ollama via `/v1`; one `AttachSharedLLM` for all seams; feature flags stay opt-in; fail-open floors stay. Install runs provider setup as a normal step (Local ensure Ollama + wire chat/embed, or External wire both) because indexing already depends on that choice; `--skip-provider` escapes.
- **Separate Ollama `/api/chat` + OpenAI clients:** two HTTP shapes, double tests, same coverage via Ollama’s `/v1`. Rejected.
- **Ask-only production attach:** leaves task-029 half-done; next change re-touches boot. Rejected.
- **Opt-in `--with-ollama` bolt-on (default no):** pretends the embedder/LLM host is optional when indexing already needs it. Rejected.
- **New third-party LLM SDK:** violates “no new dependencies without an ADR” for no gain over stdlib JSON + `net/http`. Rejected.

## Decision Outcome

Chosen option: **OpenAI-compatible HTTP Completer only; one attach; natural install provider setup.**

- Chat Completer: `Complete(ctx, system, user string) (string, error)` → `{base_url}/chat/completions`; no Anthropic-native SDK; no Ollama `/api/chat` client in v1.
- Attach: one boot function sets AskLLM, DedupLLMFunc / DedupLLM, ExtractionLLM, and DreamLLM adapters from the same Completer when `mnemonic.llm.enabled` and config is valid.
- Config: `mnemonic.llm.{enabled,base_url,model,api_key,timeout}` in the same indexing YAML stack (home `~/.skillgrid/config.d/indexing.yaml` as per-key fallback); defaults `enabled: false`; API key may come from `SKILLGRID_LLM_API_KEY` or `OPENAI_API_KEY`.
- Feature flags (`extraction.llm`, `dedup.llm`, `mem_ask mode=llm`) stay opt-in; Complete errors fail open to ADR-0016 floors.
- Install: Local vs External provider step; `--yes` ⇒ local; `--provider=local|external`; `--skip-provider` for CI/airgap. Local ensures binary/service and pulls missing `llama3.2:3b` + `nomic-embed-text`, then merges home llm + ollama embedder. External merges home llm + external embedder against the same host. Re-run is ensure (not force-refresh). Failures are non-fatal warnings.

### Consequences

- Good, because one HTTP shape covers Ollama and cloud; wiring a model is one attach site (closes task-029).
- Good, because install aligns chat Completer with the embedder operators already need for indexing.
- Good, because stdlib-only Completer needs no dependency ADR beyond this provider-shape lock.
- Bad, because Anthropic-native or Ollama-native APIs need a superseding ADR later.
- Bad, because `--yes` defaults to Local and may pull multi-GB models on first install — mitigated by `--skip-provider` / `--provider=external` and ensure-not-refresh on re-run.

Revisit triggers: (1) a required Anthropic Messages API client; (2) Ollama `/api/chat`-only features with no `/v1` parity; (3) install must never download models without an explicit flag (product policy change).
