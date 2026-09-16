-- 016 per-function CFG + PDG + taint schema (additive; 005 symbols/edges +
-- 012/013/014/015 untouched). Change 011, step 01.
--
-- The brief named this migration 014_pdg_taint.sql, but 014 is already taken
-- by 008's process flows; this is the next free number (016). It is additive:
-- the four tables are created here and populated ONLY by the opt-in --pdg pass
-- (a non---pdg index leaves them empty, and 005/008/010 are byte-for-byte
-- unchanged). No new grammar, no CGo — the CFG is built over the existing
-- gotreesitter AST, and the PDG + taint solver are pure Go.
--
-- cfg_blocks: basic blocks of a per-function control-flow graph. A block is
-- keyed stably by (symbol_id, start_line, end_line) so repeated builds of the
-- same function are reproducible. symbol_id references the 005 function/method
-- symbol whose body was CFG'd.
CREATE TABLE IF NOT EXISTS cfg_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    block_no INTEGER NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    kind TEXT NOT NULL DEFAULT 'normal',
    UNIQUE(symbol_id, block_no)
);
CREATE INDEX IF NOT EXISTS idx_cfg_blocks_symbol ON cfg_blocks(symbol_id);

-- cfg_edges: control-flow edges between basic blocks of the SAME function
-- (the CFG is intraprocedural; crossing a call boundary is not a CFG edge).
-- condition labels the branch condition ('true'/'false'/'loop'/'post'/'call'
-- etc.) that justified the edge.
CREATE TABLE IF NOT EXISTS cfg_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    from_block INTEGER NOT NULL,
    to_block INTEGER NOT NULL,
    condition TEXT NOT NULL DEFAULT '',
    UNIQUE(symbol_id, from_block, to_block, condition)
);
CREATE INDEX IF NOT EXISTS idx_cfg_edges_symbol ON cfg_edges(symbol_id);

-- pdg_edges: control- and data-dependence edges derived from the CFG. Every
-- edge carries a Confidence Label in EXTRACTED | INFERRED | AMBIGUOUS |
-- LSP_RESOLVED (the fourth value joins 005's three). A data-dependence is
-- EXTRACTED only where fully resolved; an unresolved boundary is
-- INFERRED/AMBIGUOUS (never a fabricated EXTRACTED); an LSP-resolved boundary
-- is LSP_RESOLVED, not AMBIGUOUS. kind is 'control' or 'data'.
CREATE TABLE IF NOT EXISTS pdg_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    from_line INTEGER NOT NULL,
    to_line INTEGER NOT NULL,
    from_name TEXT NOT NULL DEFAULT '',
    to_name TEXT NOT NULL DEFAULT '',
    confidence TEXT NOT NULL DEFAULT 'AMBIGUOUS',
    note TEXT NOT NULL DEFAULT '',
    UNIQUE(symbol_id, kind, from_line, to_line, from_name, to_name, confidence)
);
CREATE INDEX IF NOT EXISTS idx_pdg_edges_symbol ON pdg_edges(symbol_id);

-- taint_findings: the step 02 taint-solver output (tainted source -> sink
-- through a data flow). Created here so the opt-in ---pdg schema is complete;
-- step 01 leaves it empty (no taint pass yet).
CREATE TABLE IF NOT EXISTS taint_findings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    source_line INTEGER NOT NULL,
    sink_line INTEGER NOT NULL,
    source_name TEXT NOT NULL DEFAULT '',
    sink_name TEXT NOT NULL DEFAULT '',
    confidence TEXT NOT NULL DEFAULT 'AMBIGUOUS',
    note TEXT NOT NULL DEFAULT '',
    UNIQUE(symbol_id, source_line, sink_line, source_name, sink_name, confidence)
);
CREATE INDEX IF NOT EXISTS idx_taint_findings_symbol ON taint_findings(symbol_id);
