package memory

// Chunk codec (spec 01-memory/006 §5 — Task 16): the content-addressed
// export/import format, ported from Engram's internal/cloud/chunkcodec.
//
// A chunk is the canonical bytes of one mutation-equivalent unit (an
// entity payload). The chunk id is the first 8 hex of the chunk's
// SHA-256 — content-addressed, key-order independent (canonicalization
// re-keys the JSON with stable key order and stamps the export project).
// An archive is <dir>/manifest.json + <dir>/chunks/<chunkid>.jsonl.gz
// (gzip JSONL — one canonical payload per line).
//
// Idempotency is the chunk-id dedupe: Import skips chunks already in the
// local sync_chunks table, so re-importing the same archive changes
// nothing (the brief's re-import guarantee).

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ChunkID is the first 8 hex of the chunk's SHA-256 (Engram
// chunkcodec.go:13-16 — the content-addressed chunk identity).
func ChunkID(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])[:8]
}

// CanonicalizeForProject re-keys the payload's JSON (stable key order —
// Go's encoding/json marshals struct/map keys deterministically) and stamps
// the export project (a blank project inherits it — spec 002-schema §3.1
// payload rule). It returns the canonical bytes (what is hashed and
// written to the chunk file).
//
// ponytail: canonicalization is a JSON re-key + project stamp (the
// Engram codec's core, minus the per-entity field validation that Engram's
// larger schema needs). A non-object payload is an error (a chunk is an
// object — a scalar/array chunk cannot be canonically re-keyed).
func CanonicalizeForProject(payload []byte, project string) []byte {
	if b, err := canonicalize(payload, project); err == nil {
		return b
	}
	// a non-canonicalizable payload is returned as-is (the chunk is still
	// content-addressed — the id is stable for identical inputs). The
	// export path uses CanonicalizeError to surface the failure loudly.
	return payload
}

// CanonicalizeError is CanonicalizeForProject with the error (the export
// path refuses a non-canonicalizable chunk instead of silently hashing raw
// bytes).
func CanonicalizeError(payload []byte, project string) error {
	_, err := canonicalize(payload, project)
	return err
}

func canonicalize(payload []byte, project string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, fmt.Errorf("sync: chunk is not a json object: %v", err)
	}
	// blank-project inheritance (spec 002-schema §3.1 payload rule).
	if p, ok := doc["project"].(string); ok && strings.TrimSpace(p) == "" {
		doc["project"] = project
	}
	// re-key: encoding/json marshals map keys in sorted order — the stable
	// canonical form. (The input may have a different key order; the
	// re-marshal normalizes it, so key-order-different payloads hash
	// identically.)
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("sync: canonicalize encode: %v", err)
	}
	return out, nil
}

// ── export / import ────────────────────────────────────────────────────────

// Manifest is the export archive's manifest.json (the brief's shape:
// project, chunk list, export time, schema version).
type Manifest struct {
	Project  string   `json:"project"`
	Chunks   []string `json:"chunks"`
	Exported string   `json:"exported_at"`
	Schema   string   `json:"schema_version"`
}

// manifestFile is <dir>/manifest.json.
func manifestFile(dir string) string { return filepath.Join(dir, "manifest.json") }

// chunksDir is <dir>/chunks.
func chunksDir(dir string) string { return filepath.Join(dir, "chunks") }

// Export walks the project's entities (observations — the canonical
// payloads) and writes one chunk per observation to
// <dir>/chunks/<chunkid>.jsonl.gz plus the manifest. It returns the
// manifest.
//
// ponytail: the export walks observations (the entity the brief's
// round-trip test asserts — "20 saved observations"). Sessions/prompts/
// relations join the walk when their journal rows land (their owning
// tasks); the chunk format already carries them (one chunk per unit).
func Export(ctx context.Context, st store.Store, project, dir string) (Manifest, error) {
	_ = ctx
	man := Manifest{Project: project, Schema: "v1"}
	man.Chunks = []string{}

	// the observations' canonical payloads (the export units).
	obs, err := projectObservations(ctx, st, project)
	if err != nil {
		return man, err
	}

	if err := os.MkdirAll(chunksDir(dir), 0o755); err != nil {
		return man, fmt.Errorf("sync: export dir: %v", err)
	}

	for _, o := range obs {
		payload := observationPayload(o)
		// the chunk is the canonical payload (key-order stable, project
		// stamped). A non-canonicalizable payload is an export error (the
		// chunk is content-addressed — raw bytes would be unstable).
		if err := CanonicalizeError(payload, project); err != nil {
			return man, fmt.Errorf("sync: export observation %s: %v", o.SyncID, err)
		}
		canonical := CanonicalizeForProject(payload, project)
		id := ChunkID(canonical)
		if err := writeChunk(chunksDir(dir), id, canonical); err != nil {
			return man, fmt.Errorf("sync: write chunk %s: %v", id, err)
		}
		man.Chunks = append(man.Chunks, id)
	}

	man.Exported = nowStampExport()
	manJSON, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return man, fmt.Errorf("sync: manifest encode: %v", err)
	}
	if err := os.WriteFile(manifestFile(dir), manJSON, 0o644); err != nil {
		return man, fmt.Errorf("sync: write manifest: %v", err)
	}
	return man, nil
}

// Import reads the archive's manifest, skips chunks already in the local
// sync_chunks table (idempotency — re-import changes nothing), and applies
// each new chunk through the sync apply seam (SyncService.ApplyPulled —
// the same through-store path, INV-4 holds for imports too). It returns
// the number of chunks applied.
func Import(ctx context.Context, st store.Store, project, dir string) (int, error) {
	man, err := readManifest(dir)
	if err != nil {
		return 0, err
	}
	if man.Project != project {
		return 0, fmt.Errorf("sync: archive project %q does not match %q", man.Project, project)
	}
	svc := &SyncService{st: st}
	applied := 0
	for _, id := range man.Chunks {
		// the dedupe: a chunk already imported is skipped (the brief's
		// re-import guarantee).
		if chunkImported(st, id) {
			continue
		}
		canonical, err := readChunk(chunksDir(dir), id)
		if err != nil {
			return applied, fmt.Errorf("sync: read chunk %s: %v", id, err)
		}
		m := chunkToMutation(canonical)
		if m.EntityKey == "" {
			// a chunk with no sync id is not an apply unit (a session/
			// prompt without identity) — record it as imported but do not
			// apply (the apply seam needs an entity key).
			if err := markChunkImported(st, id); err != nil {
				return applied, fmt.Errorf("sync: record chunk %s: %v", id, err)
			}
			continue
		}
		if err := svc.ApplyPulled(ctx, m); err != nil {
			return applied, fmt.Errorf("sync: apply chunk %s: %v", id, err)
		}
		if err := markChunkImported(st, id); err != nil {
			return applied, fmt.Errorf("sync: record chunk %s: %v", id, err)
		}
		applied++
	}
	return applied, nil
}

// ── chunk file I/O ────────────────────────────────────────────────────────

// writeChunk writes the canonical payload as a gzip JSONL file
// (<dir>/<id>.jsonl.gz — one line).
func writeChunk(dir, id string, canonical []byte) error {
	f, err := os.Create(filepath.Join(dir, id+".jsonl.gz"))
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(append(canonical, '\n')); err != nil {
		return err
	}
	return gz.Close()
}

// readChunk reads a gzip JSONL chunk file (the first line is the canonical
// payload).
func readChunk(dir, id string) ([]byte, error) {
	f, err := os.Open(filepath.Join(dir, id+".jsonl.gz"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}
	// the JSONL line (strip the trailing newline).
	return bytesTrimRightNewline(data), nil
}

func bytesTrimRightNewline(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

// readManifest reads <dir>/manifest.json.
func readManifest(dir string) (Manifest, error) {
	var man Manifest
	data, err := os.ReadFile(manifestFile(dir))
	if err != nil {
		return man, fmt.Errorf("sync: read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &man); err != nil {
		return man, fmt.Errorf("sync: decode manifest: %v", err)
	}
	return man, nil
}

// chunkImported reports whether the chunk is already in sync_chunks (the
// idempotency check).
func chunkImported(st store.Store, chunkID string) bool {
	ok, err := applyPulledSeam(st).Probe(
		`SELECT COUNT(*) > 0 FROM sync_chunks WHERE chunk_id = ?`, chunkID)
	if err != nil {
		return false
	}
	return ok
}

// markChunkImported records the chunk in sync_chunks (the dedupe marker).
// The PK is (target_key, chunk_id), so the conflict target is the pair.
// It returns the insert error so a failed record is surfaced (self-healing
// — a re-import retries — but not silent).
func markChunkImported(st store.Store, chunkID string) error {
	return applyPulledSeam(st).RawExec(
		`INSERT INTO sync_chunks (target_key, chunk_id) VALUES ('local', ?)
		 ON CONFLICT(target_key, chunk_id) DO NOTHING`,
		chunkID)
}

// ── payload / mutation builders ───────────────────────────────────────────

// observationPayload is the canonical observation payload (the same shape
// the journal carries — spec 006 §4: the payload IS the canonical entity).
func observationPayload(o store.Observation) []byte {
	tool := ""
	if o.ToolName != nil {
		tool = *o.ToolName
	}
	proj := ""
	if o.Project != nil {
		proj = *o.Project
	}
	topic := ""
	if o.TopicKey != nil {
		topic = *o.TopicKey
	}
	p := syncMutationPayload{
		SyncID:    o.SyncID,
		SessionID: o.SessionID,
		Type:      o.Type,
		Title:     o.Title,
		Content:   o.Content,
		ToolName:  tool,
		Project:   proj,
		Scope:     o.Scope,
		TopicKey:  topic,
	}
	b, _ := json.Marshal(p)
	return b
}

// chunkToMutation decodes a chunk's canonical payload into the apply
// mutation (the import's through-store unit). An observation payload is an
// observation upsert; a relation payload is a relation upsert.
func chunkToMutation(canonical []byte) store.Mutation {
	var p syncMutationPayload
	if err := json.Unmarshal(canonical, &p); err == nil && p.SyncID != "" && (p.Title != "" || p.Content != "") {
		return store.Mutation{
			TargetKey: "local", Entity: "observation", EntityKey: p.SyncID,
			Op: "upsert", Payload: string(canonical), Source: "remote", Project: p.Project,
		}
	}
	var r syncRelationPayload
	if err := json.Unmarshal(canonical, &r); err == nil && r.SyncID != "" && (r.Relation != "" || r.SourceID != "") {
		return store.Mutation{
			TargetKey: "local", Entity: "relation", EntityKey: r.SyncID,
			Op: "upsert", Payload: string(canonical), Source: "remote",
		}
	}
	// ponytail: a chunk that is neither an observation nor a relation
	// payload (a session/prompt — their owning tasks) is not an apply unit
	// here; it is recorded as imported (dedupe) but not applied. The
	// EntityKey "" signals that to the caller.
	return store.Mutation{TargetKey: "local", Entity: "", EntityKey: "", Op: "upsert",
		Payload: string(canonical), Source: "remote"}
}

// projectObservations lists the project's live observations (the export
// walk). It reads through the store seam (the same raw-SQL read surface the
// sync service uses).
func projectObservations(ctx context.Context, st store.Store, project string) ([]store.Observation, error) {
	_ = ctx
	rows, err := applyPulledSeam(st).RawQuery(
		`SELECT sync_id, session_id, type, title, content, ifnull(tool_name,''), ifnull(project,''), ifnull(scope,'project'), ifnull(topic_key,'')
		 FROM observations WHERE ifnull(project,'') = ? AND deleted_at IS NULL ORDER BY id`, project)
	if err != nil {
		return nil, err
	}
	var out []store.Observation
	for _, r := range rows {
		out = append(out, store.Observation{
			SyncID:    strVal(r, 0),
			SessionID: strVal(r, 1),
			Type:      strVal(r, 2),
			Title:     strVal(r, 3),
			Content:   strVal(r, 4),
			ToolName:  strPtr(strVal(r, 5)),
			Project:   strPtr(strVal(r, 6)),
			Scope:     strVal(r, 7),
			TopicKey:  strPtr(strVal(r, 8)),
		})
	}
	return out, nil
}

// nowStampExport renders the export timestamp (the manifest's
// exported_at) — the store's shared timestamp format.
func nowStampExport() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
