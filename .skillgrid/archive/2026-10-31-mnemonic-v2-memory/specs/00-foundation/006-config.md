# 006 — Config

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `000-architecture`, `001-storage`, `003-intelligence`

---

## 1. Capability

All configuration for the memory function. Static config is a YAML file (the project's
`.skillgrid/config.yaml` or a `config.d/` set); dynamic state is environment. Nothing is
hard-coded.

## 2. Config file shape

```yaml
mnemonic:
  enabled: true

  # --- storage (001-storage) ---
  data_dir: ~/.mnemonic/mnemonic        # default; override via env
  datasource:
    dialect: sqlite                      # sqlite | postgres
    # postgres only:
    dsn: ""                              # or from MNEMONIC_PG_DSN
    max_conns: 8

  # --- project (core/project) ---
  project:
    env: MNEMONIC_PROJECT                # explicit project override
    detect: true                         # cwd → config → git → basename

  # --- intelligence (003-intelligence) ---
  llm:
    provider: openai-compat              # openai-compat | local
    base_url: ""                         # openai-compat
    model: ""
    api_key_env: MNEMONIC_LLM_API_KEY
    local:                               # local provider (Ollama-compatible)
      base_url: http://127.0.0.1:11434
      model: ""
    timeout_ms: 60000
    max_retries: 1                       # degradation: bounded, never unbounded
  embedder:
    provider: openai-compat              # openai-compat | local | none
    model: ""
    dimensions: 0                        # 0 = infer from provider

  # --- sync (01-memory/006-sync) ---
  sync:
    enabled: false                       # off = local-only (journal is a no-op)
    target: cloud                        # the target_key
    auto: false                          # background autosync (MNEMONIC_SYNC_AUTO=1)
    debounce_ms: 500
    push_limit: 100
    max_push_bytes: 8388608              # 8 MiB

  # --- memory lifecycle (01-memory/002-sessions, 006) ---
  memory:
    review:
      config_months: 3
      preference_months: 3
      discovery_months: 6
      pattern_months: 6
      decision_months: 6
      architecture_months: 12
      bugfix_months: 12
      policy_months: 12
    content_max_bytes: 32000             # truncation limit (Engram-style)
    private_tag: "<private>"             # stripped before storage
```

## 3. Environment overrides

| Env | Overrides | Notes |
|---|---|---|
| `MNEMONIC_DATA_DIR` | `data_dir` | store location |
| `MNEMONIC_PROJECT` | `project.env` | explicit project |
| `MNEMONIC_PG_DSN` | `datasource.dsn` | Postgres connection |
| `MNEMONIC_LLM_API_KEY` | `llm.api_key_env` | the key itself is never in the file |
| `MNEMONIC_SYNC_AUTO` | `sync.auto` | `1` = background autosync |
| `MNEMONIC_HTTP_TOKEN` | (serve alias) | Bearer auth for mutating HTTP routes |
| `MNEMONIC_TIMEZONE` | (display) | IANA zone for TUI/CLI timestamps |
| `MNEMONIC_DISABLE_CACHE` | (storage) | bypass the store handle pool (rollback boundary) |

## 4. Resolution order

For every setting: **flag > env > config file > default.** Secrets (API keys, DSN) are
env-only or reference an env var — never committed in the config file.

## 5. Validation

- `datasource.dialect=postgres` with empty `dsn` → error at open (not at first query).
- `llm.provider=local` requires `local.base_url`.
- `embedder.dimensions` must match the vector column on first write (mismatch → error,
  don't silently truncate).
- `mnemonic doctor` (CLI) reports the resolved config + capability (embed round-trip,
  FTS present, vector index present) and sets `degraded` on any gap.

## 6. Invariants

INV-2 (dual-mode via `datasource`), INV-3 (LLM-first, provider pluggable).

## 7. Open items

None.
