-- 009: observation tool_name provenance.
-- Additive column so mem_save can record which tool produced the row
-- (Engram-parity tool provenance). Nullable — most rows leave it unset.

ALTER TABLE observations ADD COLUMN tool_name TEXT;
