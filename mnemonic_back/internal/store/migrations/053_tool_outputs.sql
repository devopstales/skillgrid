-- 053_tool_outputs: Context Harness sandbox store (ADR-0025).
-- Session-scoped FTS5 sandbox for gated tool output; purged at session end.
CREATE TABLE IF NOT EXISTS tool_outputs (
    id          INTEGER PRIMARY KEY,
    session_id  TEXT NOT NULL,
    project_id  TEXT NOT NULL,
    tool_name   TEXT NOT NULL,
    output      TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tool_outputs_session ON tool_outputs(session_id);

CREATE VIRTUAL TABLE IF NOT EXISTS tool_outputs_fts USING fts5(
    tool_name,
    output,
    content='tool_outputs',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS tool_outputs_fts_insert AFTER INSERT ON tool_outputs BEGIN
    INSERT INTO tool_outputs_fts(rowid, tool_name, output) VALUES (new.id, new.tool_name, new.output);
END;
CREATE TRIGGER IF NOT EXISTS tool_outputs_fts_delete AFTER DELETE ON tool_outputs BEGIN
    INSERT INTO tool_outputs_fts(tool_outputs_fts, rowid, tool_name, output)
    VALUES ('delete', old.id, old.tool_name, old.output);
END;
