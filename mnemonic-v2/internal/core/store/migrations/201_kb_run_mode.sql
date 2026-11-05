-- 201: the run mode (init/update/full) for the kb run checkpoint (011 §3).
ALTER TABLE kb_run_state ADD COLUMN mode TEXT NOT NULL DEFAULT '';
