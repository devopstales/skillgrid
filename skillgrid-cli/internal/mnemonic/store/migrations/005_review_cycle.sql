-- Mnemonic: per-observation review cycle (mem_review).
-- review_after is intentionally NOT copied on project migration (local-only
-- metadata). It is NULL until a review cycle is set.

ALTER TABLE observations ADD COLUMN review_after TEXT;
