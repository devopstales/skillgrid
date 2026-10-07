// Package scan shells out to scanner CLIs, persists their raw output, and
// upserts normalized findings into the 050 scans/findings tables. Start is
// fail-open (ADR-0016): a failing scanner records status='error' on the scans
// row and returns no error.
package scan

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// RunnerFunc runs a scanner binary and returns its stdout.
type RunnerFunc func(ctx context.Context, bin string, args ...string) ([]byte, error)

// Service persists scanner runs against one project store.
type Service struct {
	store   *store.Store
	dataDir string
	// Run defaults to an exec.Command runner; tests stub it.
	Run RunnerFunc
}

// Scan is the scans-table row returned by Start.
type Scan struct {
	ID           string
	Tool         string
	Target       string
	Scanners     string
	Status       string
	StartedAt    string
	FinishedAt   string
	FindingCount int
	RawPath      string
	Error        string
}

// New creates a scan service bound to the store and its data dir.
func New(st *store.Store, dataDir string) *Service {
	return &Service{store: st, dataDir: dataDir, Run: execRunner}
}

func execRunner(ctx context.Context, bin string, args ...string) ([]byte, error) {
	var stdout strings.Builder
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run %s: %w", bin, err)
	}
	return []byte(stdout.String()), nil
}

// scannerCommands maps each tool to the CLI args Start invokes. The target is
// appended after these args by Start.
var scannerCommands = map[string][]string{
	"trivy":   {"fs", "--format", "json"},
	"wapiti":  {"--url", "-", "-o", "json"},
	"nuclei":  {"-u"},
	"semgrep": {"scan", "--json"},
}

// Start runs the scanner, persists its raw output, and records the scans row.
// Fail-open (ADR-0016): a scanner failure yields a row with status='error'
// and a non-empty error text, and no returned error. Only a failed scans-row
// write returns an error.
func (s *Service) Start(ctx context.Context, tool, target string) (*Scan, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("mint scan id: %w", err)
	}
	scanID := id.String()
	now := time.Now().UTC().Format(time.RFC3339)

	run := s.Run
	if run == nil {
		run = execRunner
	}
	var stdout []byte
	args, ok := scannerCommands[tool]
	if !ok {
		err = fmt.Errorf("unsupported scanner tool %q", tool)
	} else {
		full := append(append([]string{}, args...), target)
		stdout, err = run(ctx, tool, full...)
	}

	row := Scan{
		ID:        scanID,
		Tool:      tool,
		Target:    target,
		Status:    "ok",
		StartedAt: now,
	}
	if st, ok := scannerScanTypes[tool]; ok {
		row.Scanners = st
	}
	if err != nil {
		row.Status = "error"
		row.Error = err.Error()
	} else {
		rawPath, werr := WriteRaw(s.dataDir, scanID, stdout)
		if werr != nil {
			row.Status = "error"
			row.Error = fmt.Sprintf("write raw output: %v", werr)
		} else {
			row.RawPath = rawPath
		}
	}

	_, err = s.store.DB.ExecContext(ctx, `
		INSERT INTO scans (id, tool, target, scanners, status, started_at, raw_path, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID, row.Tool, row.Target, row.Scanners, row.Status, row.StartedAt,
		nullString(row.RawPath), nullString(row.Error),
	)
	if err != nil {
		return nil, fmt.Errorf("insert scans row: %w", err)
	}
	return &row, nil
}

// StoreFindings parses a stored scan's raw output and upserts its findings.
// It recomputes scans.finding_count and moves the row to ok (or partial when
// the raw data failed to parse). Unknown scan ids and already-error scans are
// reported as errors; parse failures are fail-open (row marked partial, error
// returned) per ADR-0016.
func (s *Service) StoreFindings(ctx context.Context, scanID string) (int, error) {
	var tool, status string
	var errText sql.NullString
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT tool, status, error FROM scans WHERE id = ?`, scanID,
	).Scan(&tool, &status, &errText)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("scan %s not found", scanID)
	}
	if err != nil {
		return 0, fmt.Errorf("load scan %s: %w", scanID, err)
	}
	if status == "error" {
		return 0, fmt.Errorf("scan %s failed: %s", scanID, errText.String)
	}

	raw, err := ReadRaw(s.dataDir, scanID)
	if err != nil {
		return 0, fmt.Errorf("read raw for scan %s: %w", scanID, err)
	}
	findings, err := Parse(tool, raw)
	if err != nil {
		s.markScanPartial(ctx, scanID, err.Error())
		return 0, fmt.Errorf("parse raw output of scan %s: %w", scanID, err)
	}

	for _, f := range findings {
		f.Tool = tool
		hash := DedupHash(tool, f.RuleID, f.Severity, f.Package, f.Version, f.File, f.Line)
		f.Severity = NormalizeSeverity(tool, f.Severity)
		links, _ := json.Marshal(f.Links)
		if _, err := s.store.DB.ExecContext(ctx, `
			INSERT INTO findings (
				scan_id, tool, dedup_hash, severity, title, cve_id, rule_id,
				package, version, fixed_version, file, line, message, links
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(scan_id, dedup_hash) DO UPDATE SET
				severity = excluded.severity,
				title = excluded.title,
				cve_id = excluded.cve_id,
				rule_id = excluded.rule_id,
				package = excluded.package,
				version = excluded.version,
				fixed_version = excluded.fixed_version,
				file = excluded.file,
				line = excluded.line,
				message = excluded.message,
				links = excluded.links`,
			scanID, tool, hash, f.Severity, f.Title, nullString(f.CVEID), nullString(f.RuleID),
			nullString(f.Package), nullString(f.Version), nullString(f.FixedVersion),
			nullString(f.File), f.Line, nullString(f.Message), string(links),
		); err != nil {
			return 0, fmt.Errorf("upsert finding %s/%s: %w", scanID, f.RuleID, err)
		}
	}

	var count int
	if err := s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM findings WHERE scan_id = ?`, scanID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count findings: %w", err)
	}
	if _, err := s.store.DB.ExecContext(ctx, `
		UPDATE scans SET finding_count = ?, status = 'ok', finished_at = ?
		WHERE id = ?`, count, time.Now().UTC().Format(time.RFC3339), scanID,
	); err != nil {
		return 0, fmt.Errorf("update scans row: %w", err)
	}
	return count, nil
}

func (s *Service) markScanPartial(ctx context.Context, scanID, why string) {
	// Fail-open (ADR-0016): a raw-parse failure lands on the row, not the caller.
	if _, err := s.store.DB.ExecContext(ctx, `
		UPDATE scans SET status = 'partial', error = ?, finished_at = ?
		WHERE id = ?`, why, time.Now().UTC().Format(time.RFC3339), scanID); err != nil {
		fmt.Fprintf(os.Stderr, "scan %s: mark partial: %v\n", scanID, err)
	}
}

// DedupHash is the stable identity hash of a finding: the same CVE or rule in
// the same package/version/file/line hashes identically across scans.
//
// The severity argument is intentionally not part of the material: a finding
// must survive a severity reclassification, so the hash is the finding's
// identity (tool, rule, package, version, file, line) only. The parameter is
// kept for call-site readability and future tightening (ADR-0030).
func DedupHash(tool, ruleID, severity, pkg, version, file string, line int) string {
	_ = severity
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s|%d",
		tool, ruleID, pkg, version, file, line)))
	return hex.EncodeToString(sum[:])
}

// scannerScanTypes records the scanners string Start writes for each tool.
var scannerScanTypes = map[string]string{
	"trivy": "vuln",
}

// DiffResult is the set difference of two scans' findings by dedup_hash.
// Removed findings are NOT deleted — the old scan's rows survive for audit.
type DiffResult struct {
	Added     []string
	Removed   []string
	Unchanged []string
}

// ListFilter narrows List: empty Tool or Status means no filter; Limit caps
// the result (0 = no cap).
type ListFilter struct {
	Tool   string
	Status string
	Limit  int
}

// StatusRow is the per-tool aggregate returned by Status.
type StatusRow struct {
	SeverityCounts map[string]int
	Latest         time.Time
}

// Diff returns the set difference of findings between two scans by dedup_hash.
// Read-only: no rows are inserted, updated, or deleted.
func (s *Service) Diff(ctx context.Context, oldID, newID string) (DiffResult, error) {
	oldSet, err := s.hashSet(ctx, oldID)
	if err != nil {
		return DiffResult{}, err
	}
	newSet, err := s.hashSet(ctx, newID)
	if err != nil {
		return DiffResult{}, err
	}
	var added, removed, unchanged []string
	for h := range newSet {
		if _, ok := oldSet[h]; ok {
			unchanged = append(unchanged, h)
		} else {
			added = append(added, h)
		}
	}
	for h := range oldSet {
		if _, ok := newSet[h]; ok {
			continue
		}
		removed = append(removed, h)
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(unchanged)
	return DiffResult{Added: added, Removed: removed, Unchanged: unchanged}, nil
}

func (s *Service) hashSet(ctx context.Context, scanID string) (map[string]struct{}, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT dedup_hash FROM findings WHERE scan_id = ?`, scanID)
	if err != nil {
		return nil, fmt.Errorf("query findings for scan %s: %w", scanID, err)
	}
	defer rows.Close()
	set := make(map[string]struct{})
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("scan dedup_hash: %w", err)
		}
		set[h] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings for scan %s: %w", scanID, err)
	}
	return set, nil
}

// LatestDiff returns the Diff between the two most recent scans by started_at.
// Returns an error when fewer than two scans exist.
func (s *Service) LatestDiff(ctx context.Context) (DiffResult, string, string, error) {
	var ids []string
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id FROM scans ORDER BY started_at DESC LIMIT 2`)
	if err != nil {
		return DiffResult{}, "", "", fmt.Errorf("query recent scans: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return DiffResult{}, "", "", fmt.Errorf("scan recent scans: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return DiffResult{}, "", "", fmt.Errorf("iterate recent scans: %w", err)
	}
	if len(ids) < 2 {
		return DiffResult{}, "", "", fmt.Errorf("need at least two scans for LatestDiff, got %d", len(ids))
	}
	// ids[0] is the most recent, ids[1] is the second most recent.
	diff, err := s.Diff(ctx, ids[1], ids[0])
	if err != nil {
		return DiffResult{}, "", "", err
	}
	return diff, ids[1], ids[0], nil
}

// Status returns per-tool finding counts by severity and the latest scan time.
func (s *Service) Status(ctx context.Context) (map[string]StatusRow, error) {
	tools := make(map[string]StatusRow)

	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT tool, severity, COUNT(*) FROM findings GROUP BY tool, severity`)
	if err != nil {
		return nil, fmt.Errorf("query findings by severity: %w", err)
	}
	for rows.Next() {
		var tool, sev string
		var n int
		if err := rows.Scan(&tool, &sev, &n); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan severity counts: %w", err)
		}
		row := tools[tool]
		if row.SeverityCounts == nil {
			row.SeverityCounts = make(map[string]int)
		}
		row.SeverityCounts[sev] = n
		tools[tool] = row
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate severity counts: %w", err)
	}
	rows.Close()

	rows, err = s.store.DB.QueryContext(ctx, `
		SELECT tool, MAX(started_at) FROM scans GROUP BY tool`)
	if err != nil {
		return nil, fmt.Errorf("query latest scan per tool: %w", err)
	}
	for rows.Next() {
		var tool string
		var startedAt string
		if err := rows.Scan(&tool, &startedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan latest per tool: %w", err)
		}
		ts, perr := time.Parse(time.RFC3339, startedAt)
		if perr != nil {
			rows.Close()
			return nil, fmt.Errorf("parse started_at %q: %w", startedAt, perr)
		}
		row := tools[tool]
		row.Latest = ts
		tools[tool] = row
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate latest per tool: %w", err)
	}
	rows.Close()

	return tools, nil
}

// List returns scans ordered by started_at DESC, filtered by tool and status.
func (s *Service) List(ctx context.Context, f ListFilter) ([]Scan, error) {
	conditions := []string{}
	args := []any{}
	if f.Tool != "" {
		conditions = append(conditions, "tool = ?")
		args = append(args, f.Tool)
	}
	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	limit := ""
	if f.Limit > 0 {
		limit = fmt.Sprintf(" LIMIT %d", f.Limit)
	}
	q := `SELECT id, tool, target, scanners, status, started_at,
		COALESCE(finished_at, ''), finding_count, COALESCE(raw_path, ''),
		COALESCE(error, '') FROM scans` + where + ` ORDER BY started_at DESC` + limit

	rows, err := s.store.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query scans: %w", err)
	}
	defer rows.Close()
	var out []Scan
	for rows.Next() {
		var sc Scan
		if err := rows.Scan(&sc.ID, &sc.Tool, &sc.Target, &sc.Scanners,
			&sc.Status, &sc.StartedAt, &sc.FinishedAt, &sc.FindingCount,
			&sc.RawPath, &sc.Error); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		out = append(out, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scans: %w", err)
	}
	return out, nil
}

// Get returns one scan by id.
func (s *Service) Get(ctx context.Context, id string) (*Scan, error) {
	var sc Scan
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT id, tool, target, scanners, status, started_at,
			COALESCE(finished_at, ''), finding_count, COALESCE(raw_path, ''),
			COALESCE(error, '') FROM scans WHERE id = ?`, id,
	).Scan(&sc.ID, &sc.Tool, &sc.Target, &sc.Scanners,
		&sc.Status, &sc.StartedAt, &sc.FinishedAt, &sc.FindingCount,
		&sc.RawPath, &sc.Error)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("scan %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get scan %s: %w", id, err)
	}
	return &sc, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
