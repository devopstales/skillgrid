// Package dep is the dependency-graph module over the 050 scan-findings
// schema (TICKET-05): it ingests SBOM dependency data, upserts the
// dependencies table by purl (soft-retiring purls that disappear between
// ingests — never deleting rows), rebuilds the per-ingest dependency edges,
// and answers reverse-dependency (affected) and whole-graph queries.
package dep

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// maxAffectedDepth caps the reverse dependency BFS (TICKET-05 spec: depth 10).
const maxAffectedDepth = 10

// Service is the dependency-graph handle over one project store.
type Service struct {
	db *sql.DB
}

// New wraps the project store's database handle.
func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// Package is one dependencies-table row (the read model for List/Get/Graph).
type Package struct {
	Purl      string  `json:"purl"`
	Name      string  `json:"name"`
	Version   string  `json:"version"`
	Ecosystem *string `json:"ecosystem,omitempty"`
	Manifest  *string `json:"manifest,omitempty"`
	Retired   bool    `json:"retired"`
	LastSeen  *string `json:"last_seen,omitempty"`
}

// Graph is the whole dependency graph: nodes (dependency purls) + edges.
type Graph struct {
	Nodes []string    `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphEdge is one stored dependency edge in the graph read model.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Ingest parses a CycloneDX SBOM and applies it to the store in one
// transaction: upsert dependencies by purl (clearing retired, setting
// last_seen), rebuild dep_edges for the ingested set, and soft-retire
// (retired=1, never deleted) every purl absent from this SBOM.
func (s *Service) Ingest(ctx context.Context, sbomData string) error {
	if s == nil || s.db == nil {
		return errors.New("dep service not initialized")
	}
	pkgs, edges, err := ParseSBOM([]byte(sbomData))
	if err != nil {
		return err
	}

	purls := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		purls = append(purls, p.Purl)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin dep ingest: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	manifest := "sbom"
	for _, p := range pkgs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dependencies (purl, name, version, ecosystem, manifest, retired, last_seen)
			VALUES (?, ?, ?, ?, ?, 0, ?)
			ON CONFLICT(purl) DO UPDATE SET
				name = excluded.name,
				version = excluded.version,
				retired = 0,
				last_seen = excluded.last_seen`,
			p.Purl, p.Name, p.Version, p.Ecosystem, manifest, now); err != nil {
			return fmt.Errorf("upsert dependency %s: %w", p.Purl, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM dep_edges WHERE from_purl IN (`+placeholders(len(purls))+`)`,
		args(purls)...); err != nil {
		return fmt.Errorf("rebuild dep_edges: %w", err)
	}
	for _, e := range edges {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dep_edges (from_purl, to_purl) VALUES (?, ?)
			ON CONFLICT(from_purl, to_purl) DO NOTHING`, e.From, e.To); err != nil {
			return fmt.Errorf("insert dep edge %s -> %s: %w", e.From, e.To, err)
		}
	}

	// Soft-retire purls absent from this SBOM: they survive, retired=1, and
	// last_seen is preserved (no column is touched).
	if _, err := tx.ExecContext(ctx, `
		UPDATE dependencies SET retired = 1 WHERE purl NOT IN (`+placeholders(len(purls))+`)`,
		args(purls)...); err != nil {
		return fmt.Errorf("soft-retire absent deps: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit dep ingest: %w", err)
	}
	return nil
}

// Affected returns the transitive set of purls that depend on purl (a reverse
// BFS over dep_edges WHERE to_purl = current, depth-capped at 10). It does not
// include purl itself.
func (s *Service) Affected(ctx context.Context, purl string) ([]string, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("dep service not initialized")
	}
	seen := map[string]bool{purl: true}
	queue := []string{purl}
	depth := map[string]int{purl: 0}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if depth[cur] >= maxAffectedDepth {
			continue
		}
		rows, err := s.db.QueryContext(ctx,
			`SELECT from_purl FROM dep_edges WHERE to_purl = ?`, cur)
		if err != nil {
			return nil, fmt.Errorf("affected query %s: %w", cur, err)
		}
		var parents []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err != nil {
				rows.Close()
				return nil, fmt.Errorf("affected scan: %w", err)
			}
			parents = append(parents, p)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("affected iterate %s: %w", cur, err)
		}
		rows.Close()
		for _, p := range parents {
			if seen[p] {
				continue
			}
			seen[p] = true
			depth[p] = depth[cur] + 1
			queue = append(queue, p)
		}
	}
	out := make([]string, 0, len(seen)-1)
	for p := range seen {
		if p != purl {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out, nil
}

// Graph returns the whole dependency graph: every dependency purl (node) and
// every stored edge.
func (s *Service) Graph(ctx context.Context) (Graph, error) {
	if s == nil || s.db == nil {
		return Graph{}, errors.New("dep service not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT purl FROM dependencies ORDER BY purl`)
	if err != nil {
		return Graph{}, fmt.Errorf("graph nodes: %w", err)
	}
	var nodes []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return Graph{}, fmt.Errorf("graph node scan: %w", err)
		}
		nodes = append(nodes, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Graph{}, fmt.Errorf("graph nodes iterate: %w", err)
	}
	rows.Close()

	erows, err := s.db.QueryContext(ctx, `SELECT from_purl, to_purl FROM dep_edges ORDER BY from_purl, to_purl`)
	if err != nil {
		return Graph{}, fmt.Errorf("graph edges: %w", err)
	}
	var edges []GraphEdge
	for erows.Next() {
		var e GraphEdge
		if err := erows.Scan(&e.From, &e.To); err != nil {
			erows.Close()
			return Graph{}, fmt.Errorf("graph edge scan: %w", err)
		}
		edges = append(edges, e)
	}
	if err := erows.Err(); err != nil {
		erows.Close()
		return Graph{}, fmt.Errorf("graph edges iterate: %w", err)
	}
	erows.Close()

	return Graph{Nodes: nodes, Edges: edges}, nil
}

// List returns dependency packages filtered by retirement state. retired=true
// returns only retired rows; retired=false returns only active rows.
func (s *Service) List(ctx context.Context, retired bool) ([]Package, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("dep service not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT purl, name, version, ecosystem, manifest, retired, last_seen
		FROM dependencies
		WHERE retired = ?
		ORDER BY purl`, retiredBool(retired))
	if err != nil {
		return nil, fmt.Errorf("list deps: %w", err)
	}
	defer rows.Close()
	var out []Package
	for rows.Next() {
		var p Package
		if err := rows.Scan(&p.Purl, &p.Name, &p.Version, &p.Ecosystem,
			&p.Manifest, &p.Retired, &p.LastSeen); err != nil {
			return nil, fmt.Errorf("scan dep: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns one dependency package by purl, or a not-found error.
func (s *Service) Get(ctx context.Context, purl string) (Package, error) {
	if s == nil || s.db == nil {
		return Package{}, errors.New("dep service not initialized")
	}
	var p Package
	err := s.db.QueryRowContext(ctx, `
		SELECT purl, name, version, ecosystem, manifest, retired, last_seen
		FROM dependencies WHERE purl = ?`, purl).
		Scan(&p.Purl, &p.Name, &p.Version, &p.Ecosystem, &p.Manifest, &p.Retired, &p.LastSeen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Package{}, fmt.Errorf("dep %s not found", purl)
		}
		return Package{}, fmt.Errorf("get dep %s: %w", purl, err)
	}
	return p, nil
}

func retiredBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func args(purls []string) []any {
	out := make([]any, len(purls))
	for i, p := range purls {
		out[i] = p
	}
	return out
}
