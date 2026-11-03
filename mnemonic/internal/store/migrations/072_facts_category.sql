-- 072_facts_category: two-target fact store (change
-- 2026-10-24-global-llm-wiki, Change A, TICKET-01). One-way: the runner
-- applies it once (tracked in index_meta); no down-migration.
--
-- The facts table (011) gains a category dimension splitting it into the
-- two Hermes-style targets: 'memory' (agent operational notes, default) and
-- 'user' (user profile). Existing rows take the NOT NULL DEFAULT 'memory'
-- value, so the change is purely additive for pre-existing stores and for
-- every per-project store that migrates later.

ALTER TABLE facts ADD COLUMN category TEXT NOT NULL DEFAULT 'memory';
