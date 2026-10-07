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

// scannerCommands maps each tool to the CLI args Start invokes.
var scannerCommands = map[string][]string{
	"trivy": {"fs", "--format", "json"},
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

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
