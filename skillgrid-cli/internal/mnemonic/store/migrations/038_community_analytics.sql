-- 038: Community structural analytics + import-cycle detection. Borrowed from
-- graphify (the 2nd external index in docs/NOTES.md), whose GRAPH_REPORT.md
-- surfaces three structural signals mnemonic did not compute:
--   1. per-community COHESION (internal-edge density) — a low number says
--      "this cluster is held together by little" and an agent reading
--      code_communities should see it per community;
--   2. IMPORT CYCLES — dependency loops (A imports B imports A) are a
--      classic structural smell graphify calls out under "Import Cycles";
--   3. the god-node ranking graphify prints — mnemonic already has that
--      (community_meta.god_nodes / code_god_nodes), so it is NOT rebuilt.
-- Mnemonic's community_meta had only label/symbol_count/god_nodes/hub_label:
-- no cohesion score, no cycle table. Both are cheap graph walks over data the
-- index already stores (the 005 symbols/edges), so they land as advisory
-- pass outputs, never load-bearing.

--
-- 1. Cohesion per community
--
-- A REAL column on community_meta (the per-community metadata table, already
-- rewritten target-state by the community pass). NULL means "not yet computed
-- by a 038 pass" (a store built before 038, or the trivial one-community
-- early return). Computed as internal_edges / C(n,2) in Go (no SQL division
-- by zero for the n<2 case).
ALTER TABLE community_meta ADD COLUMN cohesion REAL;

--
-- 2. Import cycles
--
-- A standalone table (NOT on community_meta): a cycle is a property of the
-- import subgraph, not of a single community, and cycles can span community
-- boundaries. One row per distinct cycle discovered; the canonical cycle is
-- stored as a comma-separated list of file paths in traversal order (the
-- path repeats its start to close the loop). Rebuilt target-state each index
-- run (DELETE all, then re-detect) because the import subgraph is recomputed
-- from the live edges table and cycles come and go with the code.
CREATE TABLE IF NOT EXISTS import_cycles (
    id          INTEGER PRIMARY KEY,
    cycle       TEXT NOT NULL,
    node_count  INTEGER NOT NULL,
    discovered  TEXT NOT NULL
);
