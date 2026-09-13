package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/tiered"
)

// CorpusLTM is the default semantic_search corpus (long-term memories only).
const CorpusLTM = "ltm"

// CorpusAll includes every registered tiered_contents path.
const CorpusAll = "all"

// ErrPathNotFound is returned by LoadFullDetails for unknown paths.
var ErrPathNotFound = errors.New("path not found")

// SemanticHit is an L1-only search result (never includes L2 body).
type SemanticHit struct {
	Overview string  `json:"overview"`
	Abstract string  `json:"abstract"`
	FullPath string  `json:"full_path"`
	Title    string  `json:"title,omitempty"`
	Score    float64 `json:"score,omitempty"`
}

// SemanticSearchResult is the JSON shape for semantic_search.
type SemanticSearchResult struct {
	Results []SemanticHit `json:"results"`
	TrailID int64         `json:"trail_id"`
}

type tierCandidate struct {
	fullPath     string
	abstractPath string
	overviewPath string
	title        string
}

// SemanticSearch ranks L1 overviews for projectID. corpus defaults to ltm.
// Results never include full L2 markdown bodies.
func (s *Service) SemanticSearch(ctx context.Context, projectID, query, corpus string, limit int) (*SemanticSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	corpus = normalizeCorpus(corpus)
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cands, err := listCandidates(ctx, h.store.DB, projectID, corpus)
	if err != nil {
		return nil, err
	}

	emb := embedder.Default()
	var qVec memory.Vector
	if emb != nil && strings.TrimSpace(query) != "" {
		qVec, _ = emb.Embed(ctx, query)
	}

	type scored struct {
		hit   SemanticHit
		score float64
	}
	var ranked []scored
	var dirs, files []string
	dirSet := map[string]struct{}{}

	for _, c := range cands {
		abstract := readOptional(c.abstractPath, c.fullPath+".abstract")
		overview := readOptional(c.overviewPath, c.fullPath+".overview")
		title := c.title
		if title == "" {
			title = filepath.Base(c.fullPath)
		}
		score := fallbackScore(query, title, abstract, overview)
		if emb != nil && len(qVec.Data) > 0 {
			if blob, model, ok := loadPathEmbedding(ctx, h.store.DB, projectID, c.fullPath); ok {
				_ = model
				if v, err := memory.DecodeVector(blob); err == nil {
					score = memory.CosineSimilarity(qVec, v)
				}
			} else {
				// Embed title+abstract on the fly for ranking when no stored vector.
				text := strings.TrimSpace(title + "\n" + abstract)
				if text == "" {
					text = overview
				}
				if v, err := emb.Embed(ctx, text); err == nil {
					score = memory.CosineSimilarity(qVec, v)
				}
			}
		}
		ranked = append(ranked, scored{
			hit: SemanticHit{
				Overview: overview,
				Abstract: abstract,
				FullPath: c.fullPath,
				Title:    title,
				Score:    score,
			},
			score: score,
		})
		files = append(files, c.fullPath)
		d := filepath.Dir(c.fullPath)
		if _, ok := dirSet[d]; !ok {
			dirSet[d] = struct{}{}
			dirs = append(dirs, d)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].hit.FullPath < ranked[j].hit.FullPath
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	results := make([]SemanticHit, len(ranked))
	for i := range ranked {
		results[i] = ranked[i].hit
	}
	var resultPath string
	if len(results) > 0 {
		resultPath = results[0].FullPath
	}
	trailID, err := insertTrail(ctx, h.store.DB, projectID, query, corpus, dirs, files, resultPath)
	if err != nil {
		return nil, err
	}
	return &SemanticSearchResult{Results: results, TrailID: trailID}, nil
}

// BudgetedRetrieval is the change-013 step-03 layered + budgeted read facade.
// It runs layered retrieval (L2/L3-first bootstrap, or the L1/L0 RRF fallback
// for a specific fact) and then applies the uniform read budget (item cap +
// char budget + context timeout, with the timeout enforced by a deadline-bound
// read context). It is the production entry point the CLI `mem search` routes
// through, so the L2/L3-first + RRF-fallback path is real end-to-end, not
// test-only. mem_get_observation is NOT budgeted — it stays the only
// full-content path (the in-list hits here are the truncated snippets; the full
// content is fetched by the hit's get_observation_id).
//
// For a specific fact (mode "fact") with a non-empty readerOwner, the RRF leg is
// run owner-gated (SearchOwnerScoped) so the step-01 per-owner visibility filter
// still holds: a private observation of another owner is absent from the
// in-list result even when the specific fact matches. The budget is applied to
// the visible hits only.
func (s *Service) BudgetedRetrieval(ctx context.Context, projectID, mode, query string, limit int) (memory.BudgetResult, error) {
	return s.BudgetedRetrievalAs(ctx, projectID, "", mode, query, limit)
}

// BudgetedRetrievalAs is BudgetedRetrieval with a reader identity for the
// owner-visibility gate. readerOwner names the live reader (blank =
// owner-identity / single-operator store, see everything); a non-blank reader
// scopes the L1/L0 RRF fallback to what that reader may see (step-01). The CLI
// `mem search` passes its --reader-owner here, which is the real production
// caller that routes a live read through the layered L2/L3-first + RRF-fallback
// path with the per-owner gate held.
func (s *Service) BudgetedRetrievalAs(ctx context.Context, projectID, readerOwner, mode, query string, limit int) (memory.BudgetResult, error) {
	return s.BudgetedRetrievalAsRoot(ctx, projectID, ".", readerOwner, mode, query, limit)
}

// BudgetedRetrievalAsRoot is BudgetedRetrievalAs with an explicit config root:
// the read budget is tuned from the mnemonic.retrieval_budget section found by
// walking up from configRoot (the CLI passes its --dir / data directory here so
// its config — or its flags — are honored). This is the production entry point
// the CLI `mem search` routes through.
func (s *Service) BudgetedRetrievalAsRoot(ctx context.Context, projectID, configRoot, readerOwner, mode, query string, limit int) (memory.BudgetResult, error) {
	root := configRoot
	if root == "" {
		root = "."
	}
	if abs, absErr := filepath.Abs(root); absErr == nil {
		root = abs
	}
	// Route through the layered read path with the per-owner gate held (step
	// 01). For a specific fact the L1/L0 RRF leg is owner-scoped
	// (SearchOwnerScoped); bootstrap (L2/L3) is project-scoped and unaffected
	// by the gate. The bound context enforces the budget's context timeout on
	// the query (a slow read is cut, never hung).
	return s.budgetedRetrievalHits(ctx, root, projectID, readerOwner, mode, query, "", limit)
}

// BudgetedRetrievalAsRootFTS is BudgetedRetrievalAsRoot with an FTS match
// mode for the L1/L0 fact leg (change 014, step 02): "trigram" / "prefix"
// reshape the FTS query (buildFTSQuery); an empty matchMode is the existing
// behavior. The CLI `mem search --mode` routes through this variant; the
// MCP path keeps BudgetedRetrievalAsRoot so the tool contract is unchanged.
func (s *Service) BudgetedRetrievalAsRootFTS(ctx context.Context, projectID, configRoot, readerOwner, mode, query, matchMode string, limit int) (memory.BudgetResult, error) {
	root := configRoot
	if root == "" {
		root = "."
	}
	if abs, absErr := filepath.Abs(root); absErr == nil {
		root = abs
	}
	return s.budgetedRetrievalHits(ctx, root, projectID, readerOwner, mode, query, matchMode, limit)
}

// budgetedRetrievalHits runs the layered read path with the per-owner gate
// (step 01) and the budget tuned from config + per-project override, and
// returns the budgeted result. It is the shared body of
// BudgetedRetrievalAsRoot (matchMode "") and BudgetedRetrievalAsRootFTS.
func (s *Service) budgetedRetrievalHits(ctx context.Context, root, projectID, readerOwner, mode, query, matchMode string, limit int) (memory.BudgetResult, error) {
	st, err := store.Open(s.dataDir, projectID)
	if err != nil {
		return memory.BudgetResult{}, err
	}
	defer st.Close()
	mem := memory.New(st, projectID)
	// Budget (change 013, step 03): tune the read budget from config
	// (mnemonic.retrieval_budget). Zero fields fall back to the memory package
	// defaults, so a config without the section is the default budget.
	cfg := config.Load(root)
	rb := cfg.RetrievalBudget
	mem.SetBudget(memory.BudgetConfig{Items: rb.Items, Chars: rb.Chars, TimeoutNs: rb.TimeoutNs})
	// A per-project override (CLI flags) takes precedence over the config.
	if p, ok := s.budgetOverrideFor(projectID); ok {
		mem.SetBudget(p)
	}
	// Self-improvement feedback loop (014 step 08): the mem_* read paths
	// (mem_context / mem_timeline) also honor the mnemonic.improve opt-in so
	// the re-rank is consistent across every search surface. Disabled by
	// default (byte-identical to pre-improve).
	impr := cfg.Improvement
	mem.SetImprove(memory.ImproveConfig{
		Enabled:   impr.Enabled,
		Threshold: impr.Threshold,
		MaxUsage:  impr.MaxUsage,
		BoostRate: impr.BoostRate,
		DecayRate: impr.DecayRate,
		Cooldown:  impr.Cooldown,
	})
	// AKL importance scoring (014 step 13): the mem_* read paths honor the
	// mnemonic.importance decay rate + tier thresholds (zero fields fall
	// back to the memory package defaults inside SetImportance).
	imp := cfg.Importance
	mem.SetImportance(memory.ImportanceConfig{
		DecayRate: imp.DecayRate,
		Thresholds: memory.TierThresholds{
			MatureAgeDays:    imp.TierThresholds.MatureAgeDays,
			ArchivalAgeDays:  imp.TierThresholds.ArchivalAgeDays,
			UnusedArchivalDays: imp.TierThresholds.UnusedArchivalDays,
		},
	})
	hits, err := mem.SearchOwnerScopedRetrieveFTS(mem.Budget().Bound(ctx), readerOwner, mode, query, matchMode, limit)
	if err != nil {
		return memory.BudgetResult{}, err
	}
	// Convert the layered hits into in-list observations so the uniform read
	// budget (item cap + char budget + context timeout) applies identically to
	// search/context/timeline. Each in-list result keeps its full-content
	// fetch id (GetObservationID → Observation.ID) so the agent can pull full
	// content via mem_get_observation.
	var obs []memory.Observation
	for _, h := range hits {
		obs = append(obs, memory.Observation{
			ID:      h.GetObservationID,
			Type:    h.Layer,
			Title:   h.Title,
			Content: h.Content,
			Project: h.TargetKind,
		})
	}
	return mem.Budget().Apply(ctx, obs), nil
}

// LoadFullDetails returns L2 markdown for a registered path.
func (s *Service) LoadFullDetails(ctx context.Context, projectID, path string) (string, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return "", err
	}
	defer cleanup()
	path = filepath.Clean(path)
	ok, err := pathRegistered(ctx, h.store.DB, projectID, path)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPathNotFound, path)
	}
	body, err := tiered.ReadL2(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrPathNotFound, path)
		}
		return "", err
	}
	return body, nil
}

func normalizeCorpus(corpus string) string {
	switch strings.ToLower(strings.TrimSpace(corpus)) {
	case "", CorpusLTM, "long_term", "long-term", "ltm_only":
		return CorpusLTM
	case CorpusAll, "tiered", "all_tiered":
		return CorpusAll
	default:
		return CorpusLTM
	}
}

func listCandidates(ctx context.Context, db *sql.DB, projectID, corpus string) ([]tierCandidate, error) {
	_ = ctx
	if corpus == CorpusAll {
		rows, err := db.Query(`
			SELECT full_path, COALESCE(abstract_path,''), COALESCE(overview_path,''), COALESCE(title,'')
			FROM tiered_contents WHERE project = ?`, projectID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanCandidates(rows)
	}
	rows, err := db.Query(`
		SELECT full_path, COALESCE(abstract_path,''), COALESCE(overview_path,''), COALESCE(title,'')
		FROM long_term_memories WHERE project = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidates(rows)
}

func scanCandidates(rows *sql.Rows) ([]tierCandidate, error) {
	var out []tierCandidate
	for rows.Next() {
		var c tierCandidate
		if err := rows.Scan(&c.fullPath, &c.abstractPath, &c.overviewPath, &c.title); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func pathRegistered(ctx context.Context, db *sql.DB, projectID, path string) (bool, error) {
	_ = ctx
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT 1 FROM tiered_contents WHERE project = ? AND full_path = ?
			UNION ALL
			SELECT 1 FROM long_term_memories WHERE project = ? AND full_path = ?
		)`, projectID, path, projectID, path).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func readOptional(primary, fallback string) string {
	for _, p := range []string{primary, fallback} {
		if strings.TrimSpace(p) == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err == nil {
			return string(b)
		}
	}
	return ""
}

func fallbackScore(query, title, abstract, overview string) float64 {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0
	}
	hay := strings.ToLower(title + " " + abstract + " " + overview)
	if strings.Contains(hay, q) {
		return 1
	}
	score := 0.0
	for _, tok := range strings.Fields(q) {
		if strings.Contains(hay, tok) {
			score += 0.25
		}
	}
	return score
}

func loadPathEmbedding(ctx context.Context, db *sql.DB, projectID, path string) ([]byte, string, bool) {
	_ = ctx
	var blob []byte
	var model sql.NullString
	err := db.QueryRow(`
		SELECT embedding, embedding_model FROM path_embeddings
		WHERE project = ? AND path = ? AND embedding IS NOT NULL`, projectID, path).Scan(&blob, &model)
	if err != nil || len(blob) == 0 {
		return nil, "", false
	}
	return blob, model.String, true
}

func insertTrail(ctx context.Context, db *sql.DB, projectID, query, corpus string, dirs, files []string, resultPath string) (int64, error) {
	_ = ctx
	now := time.Now().UTC().Format(time.RFC3339)
	dj, _ := json.Marshal(dirs)
	fj, _ := json.Marshal(files)
	res, err := db.Exec(`
		INSERT INTO retrieval_trails (project, query, directories_json, files_json, result_path, corpus, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		projectID, query, string(dj), string(fj), nullEmpty(resultPath), corpus, now)
	if err != nil {
		return 0, fmt.Errorf("insert retrieval_trails: %w", err)
	}
	return res.LastInsertId()
}

func nullEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
