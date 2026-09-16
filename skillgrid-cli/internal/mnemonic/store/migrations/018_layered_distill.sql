-- 018: layered distillation (change 013, step 02).
--
-- Additive on top of 017_layered_memory_governance.sql (step 01) and 005's
-- observation/session store. Every statement is CREATE TABLE IF NOT EXISTS so
-- an existing database upgrades without a rewrite; no 005 / 017 table is
-- dropped or rebuilt.
--
-- The layering is PROVENANCE-LINKED: a distilled record (L1 atom, L2 scenario,
-- L3 persona delta) is never created unless its L0 source (session and/or
-- topic) resolves to a live row. A distilled record whose source cannot be
-- resolved is simply not written — a layer is never orphaned from its
-- provenance.
--
--   observation_layers — the L0→L1→L2→L3 link table. Each row links a derived
--   layer record to its resolvable L0 source. The L0 source is a session_id
--   (the raw session record) and/or a topic (the session summary). An L1 atom
--   is an existing observation (observation_id, its session is the L0); an
--   L2/L3 record lives in personas (persona_id). The same derived record may
--   be re-linked to a changed L0 source on re-distill (dedup by
--   project+layer+target+source+content_hash).
--
--   personas — the L2 scenario + L3 persona-delta records. A scenario is a
--   working-context block distilled from a session; a persona delta is a
--   long-term profile increment. Both carry their content hash so a re-distill
--   on an unchanged L0 source is a cache hit (no re-distillation).

CREATE TABLE IF NOT EXISTS observation_layers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    layer TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    target_id INTEGER,
    source_session TEXT NOT NULL,
    source_topic TEXT,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, layer, target_kind, target_id, source_session, content_hash)
);

CREATE TABLE IF NOT EXISTS personas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- distill_llm_cache — the content-hash cache for the optional LLM pass. A row
-- per (project, source_hash) holds the L2 scenario + L3 persona delta the LLM
-- produced for that L0 source. Re-running the pass on an unchanged source is a
-- cache hit (no re-distillation, no second LLM call); a changed source misses.
CREATE TABLE IF NOT EXISTS distill_llm_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    scenario TEXT NOT NULL,
    delta TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, source_hash)
);

CREATE INDEX IF NOT EXISTS idx_layers_session
    ON observation_layers (project, source_session, layer);
CREATE INDEX IF NOT EXISTS idx_layers_target
    ON observation_layers (project, layer, target_kind, target_id);
CREATE INDEX IF NOT EXISTS idx_personas_kind
    ON personas (project, kind);
