package secondbrain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// lifecycle.go — mem_lifecycle (2026-09-30-mnemonic-second-brain Task 6):
// health / dedup / consolidate / archive over the existing store, writing an
// append-only audit row to the 043 lifecycle_log table for every mutating op.
// Soft-only (reversible): archive/restore toggle the 043 archive_reason +
// archived_at pair; dedup/consolidate soft-archive the merged-away rows and
// record consolidated_from provenance on the canonical.

const (
	// dedupCosineThreshold is the cosine similarity above which two
	// observations are treated as near-duplicates.
	dedupCosineThreshold = 0.85
	// healthCacheTTL bounds how long a computed health report is reused.
	healthCacheTTL = 24 * time.Hour
	// healthCacheDir is the OS temp dir holding per-project health caches
	// (ephemeral, like the web-cache; never the project data dir).
	healthCacheDir = "mnemonic-lifecycle-health"
	// defaultStaleDays is the archive stale threshold.
	defaultStaleDays = 90
	// consolidateLLMTimeout bounds the LLM summarize call (fail-open).
	consolidateLLMTimeout = 3 * time.Second
)

// HealthRecommendation is a single rule-based suggestion the health report
// surfaces (dedup pressure, cleanup pressure, stale coverage, embedding gaps).
type HealthRecommendation struct {
	Severity string `json:"severity"` // HIGH | MEDIUM | LOW
	Message  string `json:"message"`
}

// HealthReport is the mem_lifecycle health output. It is the HIGH/CRIT signal
// source for the later _health_warnings layer. DuplicateDensity is the
// fraction (0..1) of live observations that belong to a near-duplicate cluster.
type HealthReport struct {
	Total             int                     `json:"total"`
	CountsByType      map[string]int          `json:"counts_by_type"`
	EmbeddingCoverage float64                 `json:"embedding_coverage"`
	AgeDistribution   map[string]int          `json:"age_distribution"`
	DuplicateDensity  float64                 `json:"duplicate_density"`
	Recommendations   []HealthRecommendation  `json:"recommendations"`
	GeneratedAt       string                  `json:"generated_at"`
	Cached            bool                    `json:"_cached"`
}

// DedupCluster is one near-duplicate group: the canonical (kept) id plus the
// near-duplicate ids that would be soft-archived on a merge. Degraded is true
// when the cluster was derived by content hash (no embedder active).
type DedupCluster struct {
	Canonical int64   `json:"canonical"`
	Members   []int64 `json:"members"`
	Degraded  bool    `json:"degraded"`
}

// DedupScanResult is the dedup scan output: clusters plus the duplicate
// density (fraction of live observations in a non-singleton cluster).
type DedupScanResult struct {
	Clusters []DedupCluster `json:"clusters"`
	Density  float64        `json:"density"`
	Degraded bool           `json:"degraded"`
}

// ArchiveResult is the archive action output. Affected lists the ids the
// action touched (archived / restored / listed / flagged stale); Stale is set
// only by the stale subaction (same shape as Affected).
type ArchiveResult struct {
	Affected []int64    `json:"affected"`
	Stale    []int64    `json:"stale,omitempty"`
	Listed   []Archived `json:"listed,omitempty"`
}

// Archived is one row of the archive list subaction: an archived observation
// with its reason and stamp.
type Archived struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	ArchiveReason string `json:"archive_reason"`
	ArchivedAt   string `json:"archived_at"`
}

// --- Health ---

// healthRow is the per-observation facts Health needs (id, type, created_at,
// whether an embedding blob exists). Kept minimal so the query is cheap.
type healthRow struct {
	ID        int64
	Type      string
	CreatedAt string
	HasEmbed  bool
}

// healthProbe is the injectable metric computation. A non-nil override (set by
// the test package's withHealthProbe) replaces the real pass so the "never
// throws" and "cached" contracts are testable without a broken store. Declared
// in the test file; this file only reads it (a nil value is the real path).
var healthProbe func() (HealthReport, error)

// Health is the never-throws entry: it computes (or serves from the 24h file
// cache) a HealthReport. A panic or error yields an empty report (no error
// returned) so mem_lifecycle health never breaks a session.
func Health(ctx context.Context, svc *service.Service, projectID string) (HealthReport, error) {
	defer func() {
		if r := recover(); r != nil {
			// Swallow: the report stays at its (empty) zero value.
		}
	}()
	if svc == nil || projectID == "" {
		return HealthReport{}, nil
	}

	if rep, ok := healthCacheRead(projectID); ok {
		rep.Cached = true
		return rep, nil
	}

	var rep HealthReport
	var err error
	if healthProbe != nil {
		rep, err = healthProbe()
	} else {
		rep, err = healthComputeReal(ctx, svc, projectID)
	}
	if err != nil {
		return HealthReport{}, nil // never throws: an error is an empty report
	}
	if rep.GeneratedAt == "" {
		rep.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	healthCacheWrite(projectID, rep)
	return rep, nil
}

// healthComputeReal is the real metric pass over the store: counts-by-type,
// embedding coverage, age distribution, and duplicate density (union-find over
// vec0 cosine, degrading to a normalized-hash cluster when no embeddings).
func healthComputeReal(ctx context.Context, svc *service.Service, projectID string) (HealthReport, error) {
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return HealthReport{}, err
	}
	defer cleanup()

	var rows []healthRow
	rx, err := h.Store().DB.QueryContext(ctx, `
		SELECT id, type, created_at, CASE WHEN embedding IS NOT NULL THEN 1 ELSE 0 END
		FROM observations
		WHERE project = ? AND deleted_at IS NULL`, projectID)
	if err != nil {
		return HealthReport{}, err
	}
	defer rx.Close()
	for rx.Next() {
		var r healthRow
		if err := rx.Scan(&r.ID, &r.Type, &r.CreatedAt, &r.HasEmbed); err != nil {
			return HealthReport{}, err
		}
		rows = append(rows, r)
	}
	if err := rx.Err(); err != nil {
		return HealthReport{}, err
	}
	if len(rows) == 0 {
		return HealthReport{
			CountsByType:      map[string]int{},
			AgeDistribution:   map[string]int{},
			EmbeddingCoverage: 0,
			DuplicateDensity:  0,
			Recommendations:   []HealthRecommendation{},
		}, nil
	}

	byType := map[string]int{}
	age := map[string]int{}
	embedded := 0
	for _, r := range rows {
		byType[r.Type]++
		age[ageBucket(r.CreatedAt)]++
		if r.HasEmbed {
			embedded++
		}
	}
	coverage := float64(embedded) / float64(len(rows))

	// Duplicate density: union-find over embeddings when present, else the
	// deterministic normalized-hash floor (same as DedupScan's degrade path).
	density := healthDuplicateDensity(ctx, h.Memory(), projectID, rows)

	recs := healthRecommendations(len(rows), byType, embedded, coverage, density, age)
	return HealthReport{
		Total:             len(rows),
		CountsByType:      byType,
		EmbeddingCoverage: coverage,
		AgeDistribution:   age,
		DuplicateDensity:  density,
		Recommendations:   recs,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ageBucket maps an RFC3339 created_at to a coarse age bucket label.
func ageBucket(createdAt string) string {
	ts, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return "unknown"
	}
	days := int(time.Since(ts).Hours() / 24)
	switch {
	case days < 1:
		return "<1d"
	case days < 7:
		return "1-7d"
	case days < 30:
		return "7-30d"
	case days < 90:
		return "30-90d"
	case days < 365:
		return "90-365d"
	default:
		return ">1y"
	}
}

// healthDuplicateDensity returns the fraction of live observations that share
// a cluster of size >1. It runs the same clustering as DedupScan (cosine
// union-find, hash-degrade) and is reused for the recommendation rules.
func healthDuplicateDensity(ctx context.Context, mem *memory.Service, projectID string, rows []healthRow) float64 {
	if len(rows) < 2 {
		return 0
	}
	_, density, _ := runDedupScan(ctx, mem, projectID)
	return density
}

// healthRecommendations are the pure rules over the computed metrics
// (blueprint Task 6): HIGH duplicate pressure, MEDIUM cleanup/stale/coverage.
func healthRecommendations(total int, byType map[string]int, embedded int, coverage, density float64, age map[string]int) []HealthRecommendation {
	var recs []HealthRecommendation
	if density > 0.05 {
		recs = append(recs, HealthRecommendation{Severity: "HIGH",
			Message: fmt.Sprintf("high duplicate density (%.0f%%); run dedup merge", density*100)})
	}
	if total > 0 && coverage < 0.5 {
		recs = append(recs, HealthRecommendation{Severity: "MEDIUM",
			Message: fmt.Sprintf("embedding coverage low (%.0f%% of %d); consider re-embedding", coverage*100, total)})
	}
	old90 := age["90-365d"] + age[">1y"]
	if total > 10 && old90 > total/2 {
		recs = append(recs, HealthRecommendation{Severity: "MEDIUM",
			Message: fmt.Sprintf("%d of %d observations older than 90d; consider archive stale", old90, total)})
	}
	if recs == nil {
		recs = []HealthRecommendation{}
	}
	return recs
}

// --- Health file cache (24h, per-project) ---

type healthCacheFile struct {
	GeneratedAt string       `json:"generated_at"`
	Report      HealthReport `json:"report"`
}

func healthCachePath(projectID string) string {
	sum := sha256.Sum256([]byte(projectID))
	return filepath.Join(os.TempDir(), healthCacheDir, hex.EncodeToString(sum[:16])+".json")
}

func healthCacheRead(projectID string) (HealthReport, bool) {
	b, err := os.ReadFile(healthCachePath(projectID))
	if err != nil {
		return HealthReport{}, false
	}
	var f healthCacheFile
	if err := json.Unmarshal(b, &f); err != nil {
		return HealthReport{}, false
	}
	g, err := time.Parse(time.RFC3339, f.GeneratedAt)
	if err != nil {
		return HealthReport{}, false
	}
	if time.Since(g) > healthCacheTTL {
		return HealthReport{}, false
	}
	f.Report.GeneratedAt = f.GeneratedAt
	return f.Report, true
}

func healthCacheWrite(projectID string, rep HealthReport) {
	p := healthCachePath(projectID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	f := healthCacheFile{GeneratedAt: rep.GeneratedAt, Report: rep}
	b, err := json.Marshal(f)
	if err != nil {
		return
	}
	_ = os.WriteFile(p, b, 0o644)
}

// --- Dedup ---

// DedupScan clusters near-duplicates. With an embedder it runs union-find over
// vec0 cosine (threshold 0.85); with no embedder it degrades to clustering by
// the deterministic normalized_hash and marks every cluster degraded. dryRun
// returns clusters without writing anything.
func DedupScan(ctx context.Context, svc *service.Service, projectID string, dryRun bool) ([]DedupCluster, float64, error) {
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return nil, 0, err
	}
	defer cleanup()
	_ = dryRun // scan is read-only; dryRun documents intent for the handler

	clusters, density, err := runDedupScan(ctx, h.Memory(), projectID)
	if err != nil {
		return nil, 0, err
	}
	return clusters, density, nil
}

// runDedupScan is the shared clustering core (DedupScan + Health density).
func runDedupScan(ctx context.Context, mem *memory.Service, projectID string) ([]DedupCluster, float64, error) {
	ids, hashByID, err := liveObservations(ctx, mem, projectID)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []DedupCluster{}, 0, nil
	}

	// Degraded when there is no usable vector leg: embedder disabled, or an
	// embedder active but no rows carry embeddings. Cluster by normalized_hash
	// in both cases.
	degrade := !memory.EmbeddingEnabled()
	if !degrade {
		emb, ok := embeddingsByID(ctx, mem, projectID, ids)
		degrade = !ok || len(emb) < 2
	}
	if degrade {
		clusters := hashClusters(ids, hashByID)
		for i := range clusters {
			clusters[i].Degraded = true
		}
		return clusters, clusterDensity(clusters, len(ids)), nil
	}

	// Cosine path: union-find over the fetched threshold pairs.
	emb, _ := embeddingsByID(ctx, mem, projectID, ids)
	clusters := cosineClusters(ids, emb)
	return clusters, clusterDensity(clusters, len(ids)), nil
}

// liveObservations returns live (non-deleted) ids and their normalized_hash.
// The hash is deterministic over title+content+type; a NULL column (legacy
// rows) falls back to a recomputed hash of the same shape so clustering stays
// total.
func liveObservations(ctx context.Context, mem *memory.Service, projectID string) ([]int64, map[int64]string, error) {
	db := mem.DB()
	if db == nil {
		return nil, nil, fmt.Errorf("memory service not initialized")
	}
	rx, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(normalized_hash,''), COALESCE(title,''), COALESCE(content,''), COALESCE(type,'')
		FROM observations
		WHERE project = ? AND deleted_at IS NULL ORDER BY id`, projectID)
	if err != nil {
		return nil, nil, err
	}
	defer rx.Close()
	var ids []int64
	hashByID := map[int64]string{}
	for rx.Next() {
		var id int64
		var hash, title, content, typ string
		if err := rx.Scan(&id, &hash, &title, &content, &typ); err != nil {
			return nil, nil, err
		}
		if hash == "" {
			sum := sha256.Sum256([]byte(title + content + typ))
			hash = hex.EncodeToString(sum[:])
		}
		ids = append(ids, id)
		hashByID[id] = hash
	}
	if err := rx.Err(); err != nil {
		return nil, nil, err
	}
	return ids, hashByID, nil
}

// embeddingsByID decodes the stored embedding blobs for the given ids. ok is
// false when no usable vectors exist (none embedded, or dimension mismatch).
func embeddingsByID(ctx context.Context, mem *memory.Service, projectID string, ids []int64) (map[int64]memory.Vector, bool) {
	db := mem.DB()
	if db == nil {
		return nil, false
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, projectID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := fmt.Sprintf(`SELECT id, embedding FROM observations WHERE project = ? AND id IN (%s) AND embedding IS NOT NULL`,
		strings.Join(placeholders, ","))
	rx, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, false
	}
	defer rx.Close()
	emb := map[int64]memory.Vector{}
	for rx.Next() {
		var id int64
		var blob []byte
		if err := rx.Scan(&id, &blob); err != nil {
			continue
		}
		v, err := memory.DecodeVector(blob)
		if err != nil {
			continue
		}
		emb[id] = v
	}
	if err := rx.Err(); err != nil {
		return nil, false
	}
	if len(emb) < 2 {
		return nil, false
	}
	// All vectors must share a dimension for cosine to be meaningful.
	first := -1
	for _, v := range emb {
		if first == -1 {
			first = len(v.Data)
			continue
		}
		if len(v.Data) != first {
			return nil, false
		}
	}
	return emb, true
}

// hashClusters groups ids that share a normalized_hash (the no-embedder floor).
// normalized_hash is deterministic over title+content+type, so any two
// content-identical rows share a signature.
func hashClusters(ids []int64, hashByID map[int64]string) []DedupCluster {
	groups := map[string][]int64{}
	for _, id := range ids {
		h := hashByID[id]
		if h == "" {
			continue
		}
		groups[h] = append(groups[h], id)
	}
	var out []DedupCluster
	for _, members := range groups {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return members[i] < members[j] })
		canonical := members[0]
		dups := members[1:]
		out = append(out, DedupCluster{Canonical: canonical, Members: dups})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

// cosineClusters runs union-find over the pairwise cosine distances and
// returns one cluster per connected component of size >1. The lowest id is
// the canonical; the rest are members.
func cosineClusters(ids []int64, emb map[int64]memory.Vector) []DedupCluster {
	n := len(ids)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}
	idxOf := map[int64]int{}
	for i, id := range ids {
		idxOf[id] = i
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			sim := memory.CosineSimilarity(emb[ids[i]], emb[ids[j]])
			if sim >= dedupCosineThreshold {
				union(i, j)
			}
		}
	}
	// Group by root.
	byRoot := map[int][]int64{}
	for i, id := range ids {
		byRoot[find(i)] = append(byRoot[find(i)], id)
	}
	var out []DedupCluster
	for _, members := range byRoot {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return members[i] < members[j] })
		out = append(out, DedupCluster{Canonical: members[0], Members: members[1:]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

// clusterDensity is the fraction of live observations that are in a cluster.
func clusterDensity(clusters []DedupCluster, total int) float64 {
	if total == 0 {
		return 0
	}
	inDupes := 0
	seen := map[int64]bool{}
	for _, c := range clusters {
		if !seen[c.Canonical] {
			seen[c.Canonical] = true
			inDupes++
		}
		for _, id := range c.Members {
			if !seen[id] {
				seen[id] = true
				inDupes++
			}
		}
	}
	return float64(inDupes) / float64(total)
}

// DedupMerge soft-archives every near-duplicate of the given canonical id and
// records consolidated_from provenance on the canonical. It writes one
// lifecycle_log audit row (pending → completed/failed).
func DedupMerge(ctx context.Context, svc *service.Service, projectID string, canonical int64) error {
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return err
	}
	defer cleanup()

	clusters, _, err := runDedupScan(ctx, h.Memory(), projectID)
	if err != nil {
		return err
	}
	var members []int64
	for _, c := range clusters {
		if c.Canonical == canonical {
			members = c.Members
			break
		}
	}
	if members == nil {
		// The canonical has no detected near-duplicates: nothing to merge is
		// not a failure (a value no-op), but we still record the audit row.
		return logLifecycle(ctx, h.Memory(), projectID, "dedup", "merge", []int64{canonical}, nil, "no near-duplicates")
	}
	members = append([]int64{canonical}, members...)
	// Soft-archive each duplicate (canonical is never archived).
	for _, id := range members {
		if id == canonical {
			continue
		}
		if err := setArchived(ctx, h.Memory(), id, projectID, true, "dedup merge"); err != nil {
			return err
		}
	}
	// Consolidated_from provenance on the canonical (source column).
	if err := setConsolidatedFrom(ctx, h.Memory(), canonical, projectID, members); err != nil {
		return err
	}
	return logLifecycle(ctx, h.Memory(), projectID, "dedup", "merge", members, nil,
		fmt.Sprintf("soft-archived %d dupes into canonical %d", len(members)-1, canonical))
}

// --- Consolidate ---

// Consolidate merges the given observations into one new observation. It
// summarizes via the LLM seam (fail-open to a deterministic provenance note
// when no LLM is configured or the call errors). The new observation carries
// consolidated_from: [ids] provenance; the sources are soft-archived. Returns
// the new observation id.
func Consolidate(ctx context.Context, svc *service.Service, projectID string, obsIDs []int64, newTitle string) (int64, error) {
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return 0, err
	}
	defer cleanup()
	if len(obsIDs) == 0 {
		return 0, fmt.Errorf("consolidate: no observation ids")
	}
	if newTitle == "" {
		newTitle = "Consolidated notes"
	}

	// Gather the source content for the summary.
	sources := make([]memory.Observation, 0, len(obsIDs))
	for _, id := range obsIDs {
		o, gerr := h.Memory().Get(ctx, id)
		if gerr == nil {
			sources = append(sources, o)
		}
	}
	if len(sources) == 0 {
		return 0, fmt.Errorf("consolidate: none of the given observations exist")
	}
	content := consolidateContent(sources)
	// The new observation inherits the first source's session so it lands in
	// the same workspace bucket (Save requires a session row reference).
	sessionID := sources[0].SessionID
	if sessionID == "" {
		sessionID = projectID
	}
	newID, err := h.Memory().Save(ctx, memory.SaveInput{
		Title:     newTitle,
		Type:      "learning",
		Content:   content,
		Scope:     "project",
		SessionID: sessionID,
		Source:    "consolidate",
	})
	if err != nil {
		return 0, err
	}
	if err := setConsolidatedFrom(ctx, h.Memory(), newID, projectID, obsIDs); err != nil {
		return 0, err
	}
	for _, id := range obsIDs {
		if err := setArchived(ctx, h.Memory(), id, projectID, true, "consolidated"); err != nil {
			return 0, err
		}
	}
	if err := logLifecycle(ctx, h.Memory(), projectID, "consolidate", "", obsIDs, []int64{newID},
		fmt.Sprintf("created %d from %d sources", newID, len(obsIDs))); err != nil {
		return 0, err
	}
	return newID, nil
}

// consolidateContent produces the new observation body: LLM prose when the
// seam is available (fail-open), else a deterministic provenance note listing
// the merged sources.
func consolidateContent(sources []memory.Observation) string {
	provenance := deterministicProvenanceNote(sources)
	llm := service.AskLLMSeam()
	if llm == nil {
		return provenance
	}
	var b strings.Builder
	b.WriteString("Merge the following notes into one concise summary. Cite nothing; just summarize.\n\nSources:\n")
	for _, s := range sources {
		fmt.Fprintf(&b, "[%d] (%s) %s: %s\n", s.ID, s.Type, s.Title, s.Content)
	}
	deadline := time.Now().Add(consolidateLLMTimeout)
	cctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	prose, err := llm.Complete(cctx, "You are a concise summarizer.", b.String())
	if err != nil {
		return provenance // fail open
	}
	prose = strings.TrimSpace(prose)
	if prose == "" {
		return provenance
	}
	return prose
}

// deterministicProvenanceNote is the fail-open body: a stable, human-readable
// concatenation of the merged sources with their ids.
func deterministicProvenanceNote(sources []memory.Observation) string {
	var b strings.Builder
	b.WriteString("Consolidated from the following notes:\n")
	for _, s := range sources {
		fmt.Fprintf(&b, "- [obs:%d] (%s) %s\n", s.ID, s.Type, s.Title)
	}
	return b.String()
}

// --- Archive ---

// Archive dispatches the archive action's subactions:
//   - "archive"  soft-archives the ids with a reason + archived_at
//   - "restore"  clears the archive flag (reversible)
//   - "list"     returns the archived observations (empty when none)
//   - "stale"    returns ids older than staleDays
//
// Mutating subactions write a lifecycle_log audit row. Reads (list/stale)
// return an empty result on a missing store, never an error field.
func Archive(ctx context.Context, svc *service.Service, projectID string, subaction string, obsIDs []int64, reason string, staleDays int) (*ArchiveResult, error) {
	if staleDays <= 0 {
		staleDays = defaultStaleDays
	}
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return &ArchiveResult{Affected: []int64{}}, nil // read-ish no-op on open error
	}
	defer cleanup()

	switch subaction {
	case "archive":
		if err := applyArchive(ctx, h.Memory(), obsIDs, projectID, true, reason); err != nil {
			return nil, err
		}
		_ = logLifecycle(ctx, h.Memory(), projectID, "archive", "archive", obsIDs, obsIDs,
			fmt.Sprintf("archived with reason %q", reason))
		return &ArchiveResult{Affected: obsIDs}, nil
	case "restore":
		if err := applyArchive(ctx, h.Memory(), obsIDs, projectID, false, ""); err != nil {
			return nil, err
		}
		_ = logLifecycle(ctx, h.Memory(), projectID, "archive", "restore", obsIDs, obsIDs, "restored")
		return &ArchiveResult{Affected: obsIDs}, nil
	case "list":
		listed, err := listArchived(ctx, h.Memory(), projectID)
		if err != nil {
			return &ArchiveResult{Listed: []Archived{}}, nil
		}
		return &ArchiveResult{Listed: listed}, nil
	case "stale":
		stale, err := staleIDs(ctx, h.Memory(), projectID, staleDays)
		if err != nil {
			return &ArchiveResult{Stale: []int64{}}, nil
		}
		return &ArchiveResult{Stale: stale, Affected: stale}, nil
	default:
		return nil, fmt.Errorf("archive: unknown subaction %q", subaction)
	}
}

// applyArchive toggles the 043 archive flag on the given ids.
func applyArchive(ctx context.Context, mem *memory.Service, ids []int64, projectID string, archiving bool, reason string) error {
	for _, id := range ids {
		if err := setArchived(ctx, mem, id, projectID, archiving, reason); err != nil {
			return err
		}
	}
	return nil
}

// setArchived sets (archiving=true) or clears (false) archived_at and
// archive_reason on one observation.
func setArchived(ctx context.Context, mem *memory.Service, id int64, projectID string, archiving bool, reason string) error {
	db := mem.DB()
	if db == nil {
		return fmt.Errorf("memory service not initialized")
	}
	if archiving {
		if reason == "" {
			reason = "archived"
		}
		_, err := db.ExecContext(ctx, `
			UPDATE observations
			SET archived_at = ?, archive_reason = ?, updated_at = ?
			WHERE id = ? AND project = ? AND deleted_at IS NULL`,
			time.Now().UTC().Format(time.RFC3339), reason,
			time.Now().UTC().Format(time.RFC3339), id, projectID)
		return err
	}
	_, err := db.ExecContext(ctx, `
		UPDATE observations
		SET archived_at = NULL, archive_reason = NULL, updated_at = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), id, projectID)
	return err
}

// setConsolidatedFrom appends consolidated_from provenance (the merged ids as
// JSON) to the canonical observation's source column.
func setConsolidatedFrom(ctx context.Context, mem *memory.Service, id int64, projectID string, merged []int64) error {
	db := mem.DB()
	if db == nil {
		return fmt.Errorf("memory service not initialized")
	}
	idsJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	// Read the existing source so we do not clobber a prior value.
	var existing string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(source,'') FROM observations WHERE id = ? AND project = ?`,
		id, projectID).Scan(&existing); err != nil {
		return err
	}
	source := fmt.Sprintf(`{"consolidated_from":%s}`, string(idsJSON))
	if existing != "" {
		// Merge conservatively: keep the old source as source_source.
		source = fmt.Sprintf(`{"consolidated_from":%s,"source_source":%s}`, string(idsJSON), existing)
	}
	_, err = db.ExecContext(ctx, `UPDATE observations SET source = ? WHERE id = ? AND project = ?`,
		source, id, projectID)
	return err
}

// listArchived returns the archived observations (read; empty on none).
func listArchived(ctx context.Context, mem *memory.Service, projectID string) ([]Archived, error) {
	db := mem.DB()
	if db == nil {
		return []Archived{}, nil
	}
	rx, err := db.QueryContext(ctx, `
		SELECT id, title, COALESCE(archive_reason,''), COALESCE(archived_at,'')
		FROM observations
		WHERE project = ? AND deleted_at IS NULL AND archived_at IS NOT NULL
		ORDER BY archived_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rx.Close()
	var out []Archived
	for rx.Next() {
		var a Archived
		if err := rx.Scan(&a.ID, &a.Title, &a.ArchiveReason, &a.ArchivedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rx.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Archived{}
	}
	return out, nil
}

// staleIDs returns live ids whose created_at is older than staleDays.
func staleIDs(ctx context.Context, mem *memory.Service, projectID string, staleDays int) ([]int64, error) {
	db := mem.DB()
	if db == nil {
		return []int64{}, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -staleDays).Format(time.RFC3339)
	rx, err := db.QueryContext(ctx, `
		SELECT id FROM observations
		WHERE project = ? AND deleted_at IS NULL AND archived_at IS NULL
		  AND created_at < ?
		ORDER BY id`, projectID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rx.Close()
	var out []int64
	for rx.Next() {
		var id int64
		if err := rx.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if err := rx.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []int64{}
	}
	return out, nil
}

// --- Audit log ---

// logLifecycle writes a lifecycle_log row: pending immediately, then flips it
// to completed/failed with completed_at. It is best-effort — an audit failure
// never fails the operation (the error is returned for the caller to ignore
// where appropriate, but the data mutation already succeeded).
func logLifecycle(ctx context.Context, mem *memory.Service, projectID string, action, subaction string, inputIDs, outputIDs []int64, details string) error {
	db := mem.DB()
	if db == nil {
		return fmt.Errorf("memory service not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var res interface {
		LastInsertId() (int64, error)
	}
	insert, err := db.ExecContext(ctx, `
		INSERT INTO lifecycle_log (project, action, subaction, status, input_ids, output_ids, details, created_at)
		VALUES (?, ?, ?, 'pending', ?, ?, ?, ?)`,
		projectID, action, subaction, idsToCSV(inputIDs), idsToCSV(outputIDs), details, now)
	if err != nil {
		return err
	}
	rowID, _ := insert.LastInsertId()
	// Completed: the mutation succeeded (the caller invokes this after the
	// write). status=completed + completed_at.
	_, err = db.ExecContext(ctx, `
		UPDATE lifecycle_log SET status = 'completed', completed_at = ? WHERE id = ?`, now, rowID)
	_ = res
	return err
}

// idsToCSV renders a stable comma-separated id list (empty → empty string).
func idsToCSV(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}
