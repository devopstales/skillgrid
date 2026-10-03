# Briefing — Shared LLM Provider (Claude OS parity, P2)

> **STATUS:** `accepted` (2026-10-02)

**Topic:** 2026-10-02-mnemonic-llm-provider
**Date:** 2026-10-02
**Classification:** standard (T2)
**Build shape:** Smallest usable whole
**Priority:** P2 in the Claude OS parity program
**Queued behind:** `2026-10-02-mnemonic-project-init` (after init ships; memory-checkpoint must also be clear first)
**Supersedes tracker debt:** `.backlog/tasks/task-029 - FOLLOWUP-shared-LLM-client-attach-for-ask-extraction-dedup-seams.md`
**Findings:** `08-second-brain-roadmap.md`, `09-claude-os-deep-dive.md` (Claude OS Local/Cloud provider), task-029

## Problem / Intent

Second-brain capabilities already expose LLM seams (`AskLLM`, `DedupLLM`, `ExtractionLLM`, Dream consolidate). Only tests attach them. When `mnemonic.dedup.llm` / `extraction.llm` / `mem_ask mode=llm` is on, the live path returns “no LLM configured” and falls back to the deterministic floor. Claude OS lets the user pick Local (Ollama) or Cloud (OpenAI) at install. Skillgrid needs one production completion client so wiring a model populates every seam once.

## Purpose & Success Criteria

- **Purpose:** Operators configure one OpenAI-compatible chat endpoint (Ollama via `/v1`, OpenAI, or any compatible host); process start attaches that client to every existing LLM seam; opt-in flags still gate when the LLM is tried; failures stay fail-open to deterministic floors. `skillgrid install` treats embedding/LLM provider setup as a **natural** step (code indexing already needs Ollama or an external embedder) — Local ensures Ollama + wires chat/embed; External wires the same OpenAI-compatible host for both.
- **Success criteria:** See Requirements. With a mock HTTP server and `mnemonic.llm` enabled, ask / dedup / extraction / consolidate invoke the same client; with the client down or unset, floors still pass. A normal install (or `--yes`) runs provider setup: Local ensures binary/service/models and home llm+embedder config; External writes base_url/api_key for both; re-run is ensure (missing pieces only); `--skip-provider` leaves that step out.
- **Out of scope:** New retrieval engine; Anthropic-native SDK; separate Ollama `/api/chat` client; changing prompt text for ask/dedup/extraction; human browse (G); passive-learning product UX beyond attaching the extraction seam; project-init; dashboard; new MCP tools; Redis; treating Ollama as an unrelated optional bolt-on.

## Context

Fits existing patterns: **yes**.

Seams already exist as small interfaces + package-level `Set*` (`service.SetAskLLM`, `SetDedupLLMFunc`, `memory.SetExtractionLLM`, Dream `SetLLM`). Config already has per-feature opt-ins (`mnemonic.extraction.llm`, `mnemonic.dedup.llm`). Embedder already uses OpenAI-compatible + Ollama HTTP — **code indexing already needs a real embedder** (Ollama or external; the ONNX path is weak/stubbed). Install already has skip-if-present / fail-soft tool steps (security scanners). This change adds a **completion** client, a single attach site at service/CLI boot, and a **natural provider-setup** step on `skillgrid install` (same Local/External choice that indexing already implies) — not a second store, not new seams.

Locked: Go 1.22+, no new dependencies without an ADR (use `net/http` + stdlib JSON only), serial development, fail-open floors per ADR-0016.

## Approaches Considered

- **Chosen: A — One OpenAI-compatible HTTP client, all seams + natural provider setup on install.** Config `mnemonic.llm.{enabled,base_url,model,api_key,timeout}`; Ollama users set `base_url: http://localhost:11434/v1`. One `AttachLLM(client)` wires ask + dedup + extraction + dream. Feature flags still opt in per path. Install’s provider step is a natural update: Local (ensure Ollama + wire chat/embed) or External (wire OpenAI-compatible host for embedder + Completer) — because indexing already depends on one of those embedders.
- **Rejected: B — Separate Ollama + OpenAI clients.** Two HTTP shapes, double tests, same coverage via `/v1`.
- **Rejected: C — Ask-only first.** Leaves task-029 half-done; next change re-touches boot wiring.
- **Rejected: D — Opt-in bolt-on (`--with-ollama`, default no).** Pretends Ollama/external is optional when code indexing already needs a real embedder; makes re-install forget the dependency.

## Requirements

1. **Config block:** `mnemonic.llm` loads from indexing/config YAML with defaults off.
   - **Current:** No `mnemonic.llm` section; only per-feature bools.
   - **Target:** `enabled: false` by default; when true, require `base_url` + `model`; `api_key` optional (env `SKILLGRID_LLM_API_KEY` or `OPENAI_API_KEY` fallback); `timeout` default 3s.
   - **Acceptance:** Config tests: default off; enabled without base_url fails validate or refuses attach; API key from env when field empty.
   - **Acceptance scenario:** `happy path llm config defaults off` → `acceptance.feature`

2. **Shared completion client:** One stdlib HTTP client implements chat completions against `{base_url}/chat/completions`.
   - **Current:** No production completion client.
   - **Target:** `Complete(ctx, system, user) (string, error)` POSTs OpenAI chat format, returns first message content; non-2xx and timeout are errors (callers fail open).
   - **Acceptance:** Mock server tests: success path; 500 → error; timeout → error; no third-party LLM SDK in go.mod.
   - **Acceptance scenario:** `happy path openai-compatible complete succeeds` → `acceptance.feature`

3. **Single attach point:** Boot attaches the client to all seams when `mnemonic.llm.enabled` and config is valid.
   - **Current:** Seams only set in tests.
   - **Target:** One function (e.g. `service.AttachSharedLLM(cfg)`) sets AskLLM, DedupLLMFunc, ExtractionLLM, and Dream LLM from the same `Complete` adapter. Nil/disabled → all seams cleared/nil.
   - **Acceptance:** After attach, each seam’s getter is non-nil; after detach/disabled, all nil; one mock call counter increments across ask+dedup+extraction invocations.
   - **Acceptance scenario:** `happy path attach wires all seams from one client` → `acceptance.feature`

4. **Feature flags stay opt-in:** Attaching a client does not force LLM paths on.
   - **Current:** `extraction.llm` / `dedup.llm` default false; ask llm mode is opt-in.
   - **Target:** Unchanged. Client attached + flag false → floor. Client attached + flag true → LLM tried then fail-open.
   - **Acceptance:** Existing floor tests still pass with client attached and flags false.
   - **Acceptance scenario:** `happy path attached client with flags off uses floors` → `acceptance.feature`

5. **Fail-open preserved:** Downed provider never breaks ask cited / hash dedup / regex extraction.
   - **Current:** ADR-0016 floors.
   - **Target:** Same floors when Complete errors.
   - **Acceptance:** Mock 500 with flags on → ask returns citations; dedup returns hash path; no process panic.
   - **Acceptance scenario:** `happy path llm error fails open to floors` → `acceptance.feature`

6. **Tracker debt closed:** task-029 is done by this change or marked superseded by this spec.
   - **Current:** needs-triage FOLLOWUP.
   - **Target:** Spec references task-029; ticketing closes or links it when the change executes.
   - **Acceptance:** task-029 body points at this topic; DoD of this change includes closing it.
   - **Acceptance scenario:** `happy path task-029 superseded by this change` → `acceptance.feature`

7. **Natural provider setup on install:** `skillgrid install` always offers (and `--yes` defaults to) embedding/LLM provider setup — Local Ollama ensure + wire, or External OpenAI-compatible wire — because code indexing already needs Ollama or an external embedder.
   - **Current:** Install never touches Ollama or home llm/embedder provider wiring.
   - **Target:** A normal install step (after tools/security): interactive Local vs External (Claude OS shape); `--provider=local|external` for non-interactive; `--yes` ⇒ `local`; `--skip-provider` skips the whole step (CI/airgap). **Local:** (1) if `ollama` missing — brew on darwin, `curl -fsSL https://ollama.com/install.sh | sh` on linux (hint + continue if unsupported); (2) if `http://localhost:11434/api/tags` fails — start (`brew services start ollama` or `ollama serve`) and re-probe; warn + continue on failure (non-fatal); (3) `ollama pull` only when absent — chat `llama3.2:3b`, embed `nomic-embed-text`; (4) merge `~/.skillgrid/` home config: `llm.enabled=true`, `base_url=http://localhost:11434/v1`, `model=<chat>`, and `embedder.provider=ollama` with matching `base_url`/`model=nomic-embed-text`. **External:** prompt or flags for `base_url` + `api_key` (env fallbacks OK); merge home config enabling `llm` and `embedder.provider=external` against that host (no Ollama binary steps). Re-run is **ensure** (natural update: skip binary if present; skip pull if model listed; re-merge config keys without wiping unrelated home YAML). `--dry-run` logs intended steps, writes nothing.
   - **Acceptance:** Stubbed install tests: `--yes` runs local ensure; `--provider=external` writes external wire without ollama pull; `--skip-provider` no-ops; dry-run no side effects; ensure skips pull when models present.
   - **Acceptance scenario:** `happy path install yes ensures local ollama and wires config` → `acceptance.feature`

## Implementation Decisions

- **Modules:** new `internal/mnemonic/llm` (or `service/llm_client.go`) HTTP Complete; `AttachSharedLLM` in service boot / CLI `newMnemonicService` path; config load for `mnemonic.llm`; `internal/install` provider-setup step (local ensure + external wire; fakeable exec/HTTP probe) after security tools.
- **Interfaces:** `type Completer interface { Complete(ctx, system, user string) (string, error) }` adapted to each seam’s existing interface. Install: `Provider` (`local`|`external`|empty→prompt), `SkipProvider` on `install.Config`; `setupProvider(c *Config) error` fail-soft.
- **Data flow:** config → build client → AttachSharedLLM → seams; install provider step → Local ensure or External wire → merge home YAML for llm + embedder; no MCP surface change.
- **Error handling:** HTTP/parse errors returned; never panic; never new dependency. Provider-setup failures are warnings (install continues), matching security-tool miss hints.
- **Dependencies:** stdlib only for Completer. Ollama install shells out to brew/curl/ollama (no new Go module). ADR if we lock the OpenAI-compatible contract as the only provider shape.

## Testing Decisions

- **Good tests:** httptest mock server; config table tests; attach/detach; one shared counter across seams; fail-open with 500; install provider setup with stubbed `LookPath`/`Run`/`probe`.
- **Prior art:** `embedder` ollama/external selection tests; `dedup_llm_test.go`; `ask_test.go` fakeLLM; `installSecurityTools` dry-run tests.
- **Edge cases:** enabled but empty model; trailing slash on base_url; missing API key against a server that requires it; concurrent Complete calls; Ollama binary present but port down; brew missing on mac; model already pulled; `--skip-provider`; `--provider=external` without key (env fallback / warn).

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None this draft (installer provider-setup + operator LLM note at execute time).
- `.skillgrid/ASSUMPTIONS.md`: Add VERIFIED fact when shipped; possible ADR-0023 path row for OpenAI-compatible-only provider.
- `.skillgrid/ARCHITECTURE.md`: Note shared LLM client + natural install provider setup under Installer/Mnemonic at execute time.
- `.skillgrid/artifacts/08-second-brain-roadmap.md`: P2 item for shared LLM provider (this change).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes |
|---------------------|-------|------|--------|-------|
| Goal Clarity        | 0.92  | 0.75 | pass   | Shared client + natural Local/External install |
| Boundary Clarity    | 0.88  | 0.70 | pass   | Skip-provider escape; no Anthropic SDK; after-init |
| Constraint Clarity  | 0.86  | 0.65 | pass   | stdlib HTTP, fail-open, ensure-not-refresh |
| Acceptance Criteria | 0.86  | 0.70 | pass   | Mock Complete + install stub tests |
| **Clarity**         | 0.12  | ≤0.20| pass   | |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 1 | Queue / providers / seams | after-init; openai-only (Ollama via `/v1`); all-seams |
| 2 | Ollama install DX | ensure (not refresh); chat+embed+wire |
| 3 | Install posture | **natural update** (not opt-in bolt-on): provider setup is a normal install step because indexing already needs Ollama or external; Local vs External; `--yes`⇒local; `--skip-provider` escape |

## Open Questions & Assumptions

- **Assumption:** Ollama’s OpenAI-compatible endpoint is good enough; no native `/api/chat` client in v1.
- **Assumption:** Dream consolidate uses the same Completer when a DreamLLM adapter can wrap `Complete`; if Dream’s interface needs two prompts, the adapter maps both to Complete.
- **Assumption:** Home-config merge for llm + embedder follows the same `~/.skillgrid/` overlay path the embedder already uses (blueprint names the exact file).
- **Question:** Exact YAML file (`indexing.yaml` vs `config.d`) — follow wherever `mnemonic.extraction` already lives (blueprint names the path).

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors stay.
- Proposed at blueprint: ~~ADR-0023~~ — **accepted:** `.skillgrid/artifacts/04-adr-0023-openai-compatible-llm-provider.md`

## Terms

- [Project Init](../artifacts/02-technical-terms.md) — prior slice; this change queues after it.
- [Shared LLM Provider](../artifacts/02-technical-terms.md) — Completer + attach + natural install provider setup (Local/External).