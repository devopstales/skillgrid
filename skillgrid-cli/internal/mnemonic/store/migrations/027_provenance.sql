-- 027: provenance chain metadata on observations (change 014, step 15).
--
-- Additive: one new nullable TEXT column on observations. No existing schema
-- is touched; pre-027 rows read back with provenance NULL (no-provenance
-- saves keep working unchanged).
--
--   provenance — a JSON object recording the curation chain that produced
--                the observation:
--                  {"session_id": "...", "curate_command": "mem_save ...",
--                   "source_files": ["src/a.go"], "llm_reasoning": "..."}
--                Set ONLY on the initial save; the update path preserves it
--                (immutable once set — only the first write may establish
--                the chain).

ALTER TABLE observations ADD COLUMN provenance TEXT;
