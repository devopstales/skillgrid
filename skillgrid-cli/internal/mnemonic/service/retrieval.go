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

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
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
