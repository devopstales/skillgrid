// Package service is the shared facade over memory, codeindex, webcache, and search.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/files"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/process"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/webcache"
)

// Service is the shared facade over memory, codeindex, webcache, and search.
type Service struct {
	dataDir string
	// distillEnabled is the opt-in switch for the session-close distillation
	// pass (change 013, step 02). It is false by default so session close is
	// unchanged unless the operator opts in; the hook is async + best-effort
	// (a distill failure never breaks session close).
	distillEnabled bool
	// distillLLM is the optional LLM for the distill pass (nil = no-LLM floor).
	distillLLM layer.LLM
	// distillHook, when non-nil, is the opt-in session-close distillation hook.
	// It is armed by EnableDistill (the runtime opt-in switch) and fired by
	// memory.SessionEnd in a detached goroutine. Because the hook is a pointer
	// field read at close time (not captured at store-open time), a close fires
	// the hook iff distillation is enabled AT THAT MOMENT — preserving the
	// default-off behavior (nil → no hook) even for stores opened before the
	// operator enabled distillation.
	distillHook func(ctx context.Context, projectID, sessionID, summary string)
	// distillTestMu guards the two test-only fields below. The hook closure runs
	// in a detached goroutine (session close) and the accessors are called from
	// the test's main goroutine, so the counter and last-result are written and
	// read concurrently.
	distillTestMu sync.Mutex
	// distillHookFired is a test-only signal counting how many times the
	// session-close distill hook actually ran (set by the hook closure).
	distillHookFired int
	// distillLastResult is a test-only record of the most recent hook result.
	distillLastResult layer.DistillResult
	// budgetOverrides is a per-project read-budget override (change 013,
	// step 03). The CLI sets it from its --item/--char/--timeout flags so they
	// take precedence over the config-loaded budget for a BudgetedRetrieval
	// read; the MCP path leaves it empty (config in openProject is the source
	// of truth). Guarded by budgetMu (set before reads, read at read time).
	budgetMu        sync.Mutex
	budgetOverrides map[string]memory.BudgetConfig
}

// SetBudgetOverride sets a per-project read-budget override (change 013,
// step 03). A zero field in the override falls back to its default, and the
// override takes precedence over the config-loaded budget for subsequent
// BudgetedRetrieval reads of that project. The CLI uses it to apply its
// --item/--char/--timeout flags; the MCP path does not (config is its source
// of truth in openProject).
func (s *Service) SetBudgetOverride(projectID string, cfg memory.BudgetConfig) {
	if s == nil {
		return
	}
	s.budgetMu.Lock()
	defer s.budgetMu.Unlock()
	if s.budgetOverrides == nil {
		s.budgetOverrides = make(map[string]memory.BudgetConfig)
	}
	s.budgetOverrides[projectID] = cfg
}

// budgetOverrideFor returns the per-project read-budget override, if any.
func (s *Service) budgetOverrideFor(projectID string) (memory.BudgetConfig, bool) {
	if s == nil {
		return memory.BudgetConfig{}, false
	}
	s.budgetMu.Lock()
	defer s.budgetMu.Unlock()
	cfg, ok := s.budgetOverrides[projectID]
	return cfg, ok
}

// DistillHookFired returns how many times the session-close distill hook ran.
// Test-only: lets an end-to-end close test assert the detached goroutine fired.
func (s *Service) DistillHookFired() int {
	if s == nil {
		return 0
	}
	s.distillTestMu.Lock()
	defer s.distillTestMu.Unlock()
	return s.distillHookFired
}

// DistillLastResult returns the most recent hook distill result. Test-only.
func (s *Service) DistillLastResult() layer.DistillResult {
	if s == nil {
		return layer.DistillResult{}
	}
	s.distillTestMu.Lock()
	defer s.distillTestMu.Unlock()
	return s.distillLastResult
}

// SetDistillHookForTest replaces the session-close distill hook. Test-only: it
// lets an end-to-end close test inject a hook that records the opt-in firing
// and returns a captured best-effort error (so the test can prove a distill
// error is surfaced in DistillResult.Err yet never breaks session close).
func (s *Service) SetDistillHookForTest(fn func(ctx context.Context, projectID, sessionID, summary string)) {
	if s == nil {
		return
	}
	s.distillHook = fn
}

// RecordDistillTestResult records a test hook's distill result so an
// end-to-end close test can assert the captured best-effort error. Test-only.
func (s *Service) RecordDistillTestResult(res layer.DistillResult) {
	if s != nil {
		s.distillTestMu.Lock()
		s.distillLastResult = res
		s.distillTestMu.Unlock()
	}
}

// New creates a service using dataDir for per-project SQLite stores.
func New(dataDir string) *Service {
	return &Service{dataDir: dataDir}
}

// EnableDistill turns on the opt-in session-close distillation pass. It is a no
// operation on the rest of the facade: it only arms the async best-effort hook
// that runs at session end. The hook is stored on the service (not the handle)
// so it is read at close time, honoring the opt-in state regardless of when the
// store was opened; memory.SessionEnd reads it and, when non-nil, runs it in a
// detached goroutine (the production fire-and-forget path).
func (s *Service) EnableDistill() error {
	if s == nil {
		return errors.New("service not initialized")
	}
	s.distillEnabled = true
	s.distillHook = func(ctx context.Context, projectID, sessionID, summary string) {
		res, _ := s.DistillSession(ctx, projectID, sessionID, summary, nil)
		s.distillTestMu.Lock()
		s.distillHookFired++
		s.distillLastResult = res // test-only: last hook result
		s.distillTestMu.Unlock()
	}
	return nil
}

// DistillSession runs the session-close distillation for projectID/sessionID
// (L0 → L1 atoms → L2 scenario → L3 persona delta, provenance-linked). It is
// the best-effort seam: when distillation is not enabled the pass is a no-op,
// and a distill error is captured in DistillResult.Err (not returned) so it
// never breaks session close.
//
// The method itself is synchronous and returns the populated result (so the
// hook's effect is observable and testable); the session-close hook is async
// because the handler (02.4) runs this in a detached goroutine that discards
// the result, which is what "async, never breaks close" means in production.
// The wg parameter (nil for fire-and-forget) is signaled when the pass has
// run, so a caller can await completion in tests.
//
// summary is the L0 text distilled. When non-empty it is persisted into the
// session row (as the L0 record the distill reads) so a summary-less close
// still distills from the close summary; when empty the pass falls back to the
// session row's existing summary.
func (s *Service) DistillSession(ctx context.Context, projectID, sessionID, summary string, wg *sync.WaitGroup) (layer.DistillResult, error) {
	if s == nil {
		return layer.DistillResult{}, errors.New("service not initialized")
	}
	if !s.distillEnabled {
		if wg != nil {
			wg.Add(1)
			wg.Done()
		}
		return layer.DistillResult{}, nil
	}
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}
	// Reopen the store, retrying on a transient lock: the session-close hook
	// runs in a detached goroutine a beat after the handler's store closes, so
	// the open can race the handler's WAL close. The best-effort nature of the
	// hook (a distill error is captured, never propagated) means we just retry
	// a few times rather than fail close.
	var (
		h       *ProjectHandle
		cleanup func()
		err     error
	)
	for i := 0; i < 5; i++ {
		h, cleanup, err = s.openProject(projectID, ".")
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return layer.DistillResult{Err: err}, nil
	}
	defer cleanup()
	res := layer.DistillResult{}
	// A summary-less close carries the L0 in `summary`; persist it so the
	// distill (which reads the session row) has the L0 text to work on.
	if strings.TrimSpace(summary) != "" {
		if err := h.Memory().SessionSummary(ctx, sessionID, summary); err != nil {
			res.Err = err
			return res, nil
		}
	}
	// Best-effort: the distill error is captured, not propagated, so it never
	// breaks the caller (session close).
	res, _ = layer.Distill(ctx, h.Memory(), sessionID, layer.DistillOptions{LLM: s.distillLLM})
	return res, nil
}

// Layers returns the L0→L1→L2→L3 chain for a session, with each layer's
// provenance link — the service seam behind mem_layers.
func (s *Service) Layers(ctx context.Context, projectID, sessionID string) (layer.Chain, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return layer.Chain{}, err
	}
	defer cleanup()
	return layer.Inspect(ctx, h.Memory(), sessionID)
}

// DefaultDataDir returns the mnemonic data directory from env or ~/.skillgrid/mnemonic.
func DefaultDataDir() (string, error) {
	if v := os.Getenv("SKILLGRID_MNEMONIC_DATA_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skillgrid", "mnemonic"), nil
}

// ProjectHandle is the deep single-project seam: one opened project with
// direct access to its memory, webcache, and store. Callers outside the
// service package (MCP, HTTP) open a handle once and work through it instead
// of the wide open-delegate-close facade.
type ProjectHandle struct {
	store     *store.Store
	projectID string
	root      string // workspace directory for ContentPlane (.skillgrid/files)
	memory    *memory.Service
	web       *webcache.Service
	content   *files.ContentPlane
	// distillService is the owning *Service (change 013, step 02). The handle
	// exposes it to memory.SessionEnd via DistillHook so a session close reads
	// the opt-in hook at close time (the hook itself is armed on the service by
	// EnableDistill). Non-nil for handles opened through the service.
	distillService *Service
}

func (s *Service) openProject(projectID, configRoot string) (*ProjectHandle, func(), error) {
	if s == nil {
		return nil, nil, fmt.Errorf("service not initialized")
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, nil, fmt.Errorf("project id is required")
	}
	st, err := store.Open(s.dataDir, projectID)
	if err != nil {
		return nil, nil, err
	}
	root := configRoot
	if abs, absErr := filepath.Abs(configRoot); absErr == nil {
		root = abs
	}
	cfg := config.Load(root)
	mem := memory.New(st, projectID)
	// Budget (change 013, step 03): tune the mem_* read budget from config
	// (mnemonic.retrieval_budget). Zero fields fall back to the memory package
	// defaults, so a config without the section is the default budget.
	rb := cfg.RetrievalBudget
	mem.SetBudget(memory.BudgetConfig{
		Items:     rb.Items,
		Chars:     rb.Chars,
		TimeoutNs: rb.TimeoutNs,
	})
	h := &ProjectHandle{
		store:          st,
		projectID:      projectID,
		root:           root,
		memory:         mem,
		web:            webcache.New(st, projectID, cfg.WebCache),
		content:        files.NewContentPlane(root),
		distillService: s,
	}
	// Attach the handle as the memory service's hook provider so SessionEnd can
	// fire the opt-in distill hook at close time. The hook itself lives on the
	// service (armed by EnableDistill), so this does NOT capture the opt-in
	// state at open time — a close fires the hook iff distillation is enabled
	// when it happens, and every real close path (MCP mem_session_end + HTTP)
	// fires it through memory.SessionEnd.
	mem.SetDistillHookProvider(h)
	return h, func() { st.Close() }, nil
}

// DistillHook returns the opt-in session-close distill hook for this handle, or
// nil when distillation is not enabled. It reads the hook from the owning
// service at close time (not at open time) so the opt-in state is honored
// whenever a close happens. It satisfies memory.distillHookProvider (the
// exported method is required to cross the package boundary).
func (h *ProjectHandle) DistillHook() func(ctx context.Context, projectID, sessionID, summary string) {
	if h == nil || h.distillService == nil {
		return nil
	}
	return h.distillService.distillHook
}

func (s *Service) openProjectForDirectory(directory string) (*ProjectHandle, func(), error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve directory: %w", err)
	}
	res, resErr := project.ResolveDetailed(absDir)
	if resErr != nil {
		// Hard abort for writes: never open/create under an ambiguous
		// directory-hash fallback (or other resolve failures such as binding
		// write errors). Recover via MNEMONIC_PROJECT or explicit project=.
		return nil, nil, fmt.Errorf("resolve project: %w", resErr)
	}
	// Best-effort: fold any pre-identity directory-hash store for this path
	// into the canonical identity bucket so prior memories are reachable and
	// future alias-named writes route here. Idempotent and read-mostly.
	if res.SeedID != "" && res.SeedID != res.ID && res.Source == project.SourceIdentity {
		if _, _, err := s.MergeProjects(context.Background(), res.SeedID, res.ID); err == nil {
			// recorded
		}
	}
	return s.openProject(res.ID, absDir)
}

func (s *Service) openProjectFromCWD() (*ProjectHandle, func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	return s.openProjectForDirectory(cwd)
}

// ListProjects returns the project IDs with a store in dataDir, sorted.
func (s *Service) ListProjects() ([]string, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sqlite") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".sqlite")
		if strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// ResolveProject returns the project ID for directory.
func (s *Service) ResolveProject(directory string) (string, error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	return project.Resolve(absDir)
}

// SessionStart creates a workspace session for directory. title is the
// session name shown in the web dashboard (mem-sessions).
func (s *Service) SessionStart(ctx context.Context, directory, title string) (sessionID, projectID string, err error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	sessionID, err = h.memory.SessionStart(ctx, directory, title)
	if err != nil {
		return "", "", err
	}
	return sessionID, h.projectID, nil
}

// SessionStartByClientID registers a session under the caller's ID (idempotent).
func (s *Service) SessionStartByClientID(ctx context.Context, sessionID, directory, title string) (id, projectID string, existed bool, err error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return "", "", false, err
	}
	defer cleanup()
	id, projID, existed, err := h.memory.SessionStartByClientID(ctx, sessionID, directory, title)
	if err != nil {
		return "", "", false, err
	}
	return id, projID, existed, nil
}

// PromptInput is a captured user prompt.
type PromptInput = memory.PromptInput

// PassiveInput is free text for server-side learnings extraction.
type PassiveInput = memory.PassiveInput

// MigrateProjects rolls data recorded under oldProject into the newProject
// store. In Mnemonic each project has its own SQLite file, so the data lives
// in <oldProject>.sqlite while new writes go to <newProject>.sqlite. We open
// the old store, ATTACH the new one, copy every row tagged oldProject across
// (observations, sessions, web_cache, prompts, files), re-tag it as
// newProject, and record the migration for idempotency.
//
// Idempotent: if no rows tagged oldProject remain, nothing is copied and we
// just ensure the record exists. If the old store file is missing, this is a
// clean no-op (0 moved, nil error).
//
// Called from the agent plugin on first run when the project id it computed
// differs from one previously recorded in the data dir.
func (s *Service) MigrateProjects(ctx context.Context, oldProject, newProject string) (int, error) {
	oldProject = strings.TrimSpace(oldProject)
	newProject = strings.TrimSpace(newProject)
	if oldProject == "" || newProject == "" || oldProject == newProject {
		return 0, nil
	}

	oldPath := filepath.Join(s.dataDir, oldProject+".sqlite")
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return 0, nil // nothing to migrate
		}
		return 0, err
	}
	// Ensure the destination store exists (and is migrated) before we attach.
	newStore, err := store.Open(s.dataDir, newProject)
	if err != nil {
		return 0, fmt.Errorf("open destination store: %w", err)
	}
	defer newStore.Close()
	oldStore, err := store.Open(s.dataDir, oldProject)
	if err != nil {
		return 0, fmt.Errorf("open source store: %w", err)
	}
	defer oldStore.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	attachSQL := "ATTACH DATABASE ? AS newdb"
	res, err := oldStore.DB.ExecContext(ctx, attachSQL, newStore.Path())
	if err != nil {
		return 0, fmt.Errorf("attach new store: %w", err)
	}
	defer func() {
		_, _ = oldStore.DB.Exec("DETACH DATABASE newdb")
	}()
	_ = res

	t := oldStore.DB

	// Bulk copy across attached DBs must disable FK checks: sessions must land
	// before observations, and OR IGNORE / schema drift can otherwise leave
	// dangling session_id references mid-copy.
	if _, err := t.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return 0, fmt.Errorf("disable foreign_keys: %w", err)
	}
	defer func() { _, _ = t.Exec(`PRAGMA foreign_keys=ON`) }()

	var total int
	copied := []string{}
	// sessions first: observations.session_id REFERENCES sessions(id).
	// sessions/prompts have no deleted_at column — do not filter on it.
	for _, table := range []string{"sessions", "prompts", "observations", "web_cache", "files"} {
		hasDeletedAt := table == "observations" || table == "web_cache" || table == "files"
		countSQL := `SELECT COUNT(*) FROM ` + table + ` WHERE project = ?`
		if hasDeletedAt {
			countSQL += ` AND deleted_at IS NULL`
		}
		var n int
		err := t.QueryRowContext(ctx, countSQL, oldProject).Scan(&n)
		if err != nil || n == 0 {
			continue
		}
		insertSQL := `INSERT OR IGNORE INTO newdb.` + table + ` SELECT * FROM ` + table + ` WHERE project = ?`
		if hasDeletedAt {
			insertSQL += ` AND (deleted_at IS NULL OR deleted_at = '')`
		}
		res, err := t.ExecContext(ctx, insertSQL, oldProject)
		if err != nil {
			if strings.Contains(err.Error(), "no such table") || strings.Contains(err.Error(), "duplicate column") || strings.Contains(err.Error(), "datatype mismatch") {
				continue
			}
			return total, fmt.Errorf("copy %s: %w", table, err)
		}
		affected, _ := res.RowsAffected()
		total += int(affected)
		copied = append(copied, fmt.Sprintf("%s=%d", table, affected))
		// Re-tag copied rows in the destination.
		if _, err := t.ExecContext(ctx, `
			UPDATE newdb.`+table+` SET project = ? WHERE project = ?`,
			newProject, oldProject); err != nil {
			if !strings.Contains(err.Error(), "no such table") {
				return total, fmt.Errorf("retag %s: %w", table, err)
			}
		}
	}

	// Record the migration in the destination for idempotency.
	if _, err := newStore.DB.ExecContext(ctx, `
		INSERT INTO project_migrations (old_project, new_project, migrated_at)
		VALUES (?, ?, ?)`,
		oldProject, newProject, now); err != nil && !strings.Contains(err.Error(), "no such table") {
		// ignore — best effort
	}

	return total, nil
}

// ProjectDrift mirrors the memory package's drift report, for API consumers.
type ProjectDrift = memory.ProjectDrift

// CheckProjectDrift returns a drift report when projectName is a known alias
// of a canonical project. It probes the store that most likely holds the
// alias row (the canonical project's store, then the current CWD store, then
// every known store) and does not modify state.
func (s *Service) CheckProjectDrift(ctx context.Context, projectName string) (*ProjectDrift, error) {
	name := strings.TrimSpace(projectName)
	if name == "" {
		return nil, nil
	}
	probe := func(storeID string) (*ProjectDrift, bool) {
		if storeID == "" {
			return nil, false
		}
		h, cleanup, err := s.openProject(storeID, ".")
		if err != nil {
			return nil, false
		}
		d, err := h.memory.CheckProjectDrift(ctx, name)
		cleanup()
		if err != nil {
			return nil, false
		}
		if d != nil {
			return d, true
		}
		return nil, false
	}
	// 1) The canonical project (if the alias points at one of our stores).
	if canonical := s.canonicalForAlias(ctx, name); canonical != "" {
		if d, ok := probe(canonical); ok {
			return d, nil
		}
	}
	// 2) The explicitly-named store (it may hold its own alias row).
	if n := storeIDFor(name); n != "" {
		if d, ok := probe(n); ok {
			return d, nil
		}
	}
	// 3) Current CWD store.
	if cwd, err := os.Getwd(); err == nil {
		if pid, err := s.ResolveProject(cwd); err == nil {
			if d, ok := probe(pid); ok {
				return d, nil
			}
		}
	}
	return nil, nil
}

// storeIDFor normalizes a project name into its store ID (best-effort — no IO).
func storeIDFor(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return ""
	}
	return n
}

// canonicalForAlias scans existing stores for a project_aliases row naming
// projectName as an alias, and returns the canonical name. Best-effort.
func (s *Service) canonicalForAlias(ctx context.Context, alias string) string {
	projects, err := s.ListProjects()
	if err != nil {
		return ""
	}
	for _, p := range projects {
		h, cleanup, err := s.openProject(p, ".")
		if err != nil {
			continue
		}
		var canonical string
		err = h.store.DB.QueryRowContext(ctx, `
			SELECT canonical FROM project_aliases WHERE alias = ? LIMIT 1`, alias,
		).Scan(&canonical)
		cleanup()
		if err == nil && canonical != "" {
			return canonical
		}
	}
	return ""
}

// MergeProjects consolidates data recorded under the source project name into
// the canonical project, then records a project alias. Returns the number of
// rows moved and the canonical name.
//
// Behaviour:
//   - If source == canonical or either is blank, no-op (0, canonical).
//   - If the source store file does not exist, no-op (0, canonical) — but the
//     alias is still recorded so future writes to the source name land in the
//     canonical store.
//   - Otherwise: copies observations/sessions/web_cache/prompts/files rows
//     tagged source into the canonical store (INSERT OR IGNORE so re-runs do
//     not fail on PK collisions), re-tags them as canonical, and records the
//     migration + alias (idempotent via unique primary key).
func (s *Service) MergeProjects(ctx context.Context, source, canonical string) (int, string, error) {
	source = strings.TrimSpace(source)
	canonical = strings.TrimSpace(canonical)
	if source == "" || canonical == "" || source == canonical {
		return 0, canonical, nil
	}

	// Record the alias first so that future resolution always lands in the
	// canonical store regardless of whether the copy below succeeds.
	if err := s.recordProjectAlias(ctx, canonical, source); err != nil {
		return 0, canonical, fmt.Errorf("record alias: %w", err)
	}

	// No data to move if the source store file is missing.
	srcPath := filepath.Join(s.dataDir, source+".sqlite")
	if _, err := os.Stat(srcPath); err != nil {
		if os.IsNotExist(err) {
			return 0, canonical, nil
		}
		return 0, canonical, err
	}

	moved, err := s.MigrateProjects(ctx, source, canonical)
	if err != nil {
		return 0, canonical, err
	}
	return moved, canonical, nil
}

// recordProjectAlias marks source as an alias of canonical in the canonical
// store. Idempotent: re-merge keeps the first merge time.
func (s *Service) recordProjectAlias(ctx context.Context, canonical, source string) error {
	newStore, err := store.Open(s.dataDir, canonical)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such table") {
			return nil // no canonical store yet — alias is implicit
		}
		return err
	}
	defer newStore.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = newStore.DB.ExecContext(ctx, `
		INSERT INTO project_aliases (alias, canonical, merged_at)
		VALUES (?, ?, ?)
		ON CONFLICT(alias) DO NOTHING`,
		source, canonical, now,
	)
	return err
}

// SaveObservationInput holds fields for saving an observation (HTTP + MCP).
type SaveObservationInput struct {
	Title     string `json:"title"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Scope     string `json:"scope"`
	TopicKey  string `json:"topic_key"`
	SessionID string `json:"session_id"`
	// CapturePrompt, when true, best-effort links the session's most recent
	// user prompt to the observation (mem_save capture_prompt).
	CapturePrompt bool `json:"capture_prompt"`
	// ProjectName, when non-empty, names the logical project for the save. It
	// is validated against project_aliases and a drift warning surfaced by the
	// caller when the name has been retired by mem_merge_projects.
	ProjectName string `json:"project_name"`
	// ToolName is optional provenance for which tool produced the save.
	ToolName string `json:"tool_name"`
}

// envSearchParallel is the rollback boundary for the parallel cross-store
// search (change 014, step 03): setting it to "0" forces the sequential
// loop, so a concurrency regression can be rolled back at runtime without a
// code deploy. Unset (or any other value) keeps the parallel path active.
const envSearchParallel = "SKILLGRID_SEARCH_PARALLEL"

// searchParallelEnabled reports whether the parallel cross-store search path
// is active (opt-out via SKILLGRID_SEARCH_PARALLEL=0).
func searchParallelEnabled() bool {
	return os.Getenv(envSearchParallel) != "0"
}

// semaphoreAcquireTimeout bounds how long a store search may wait for a
// semaphore slot (change 014, step 03): if the connection pool is exhausted
// and the slot stays held for this long, the store is skipped with a warning
// instead of blocking the whole search indefinitely.
const semaphoreAcquireTimeout = 5 * time.Second

// searchStoreLatency is a test-only artificial per-store latency (nanoseconds)
// so the parallel-vs-sequential timing assertion is stable on CI. Production
// leaves it at zero.
var searchStoreLatency atomic.Int64

// scopedSearchFunc is the per-store search body of SearchObservationsAll.
// Production wires it to Service.SearchObservationsScoped; tests may replace
// it to track concurrent goroutines (change 014, step 03). Guarded by
// scopedSearchMu (written by tests before the search, read per store).
var scopedSearchMu sync.Mutex
var scopedSearchFunc func(s *Service, ctx context.Context, projectID, query, matchMode, scope string, limit int) ([]memory.Observation, error)

// scopedSearch is the per-store search body of SearchObservationsAll. By
// default it is Service.SearchObservationsScoped; tests may swap
// scopedSearchFunc (e.g. to track concurrent goroutines). The test-only
// latency is applied on top so timing assertions stay stable on CI.
func scopedSearch(s *Service, ctx context.Context, projectID, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	scopedSearchMu.Lock()
	fn := scopedSearchFunc
	scopedSearchMu.Unlock()
	if d := searchStoreLatency.Load(); d > 0 {
		time.Sleep(time.Duration(d))
	}
	if fn == nil {
		return s.SearchObservationsScoped(ctx, projectID, query, matchMode, scope, limit)
	}
	return fn(s, ctx, projectID, query, matchMode, scope, limit)
}

// SearchObservationsAll runs the same FTS query across every store in dataDir
// and returns the union, ordered by global rank (each store returns its own
// bm25-ranked list; the merged result interleaves by cross-store rank, so a
// #1 hit in one store is never buried under #5 hits from another). Used by
// mem_search all_projects=true so an agent at a parent directory can still
// find memories stored under a child project's store.
//
// Concurrency (change 014, step 03): each store is searched in its own
// goroutine, bounded by a semaphore of size min(len(stores), NumCPU) with a
// 5s acquire timeout. A store that fails (missing file, busy lock, or
// semaphore timeout) is skipped with a warning and contributes no results —
// the rest of the merge is unaffected. Set SKILLGRID_SEARCH_PARALLEL=0 to
// force the sequential loop (rollback boundary).
func (s *Service) SearchObservationsAll(ctx context.Context, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	if limit <= 0 {
		limit = 20
	}
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return []memory.Observation{}, nil
	}
	if !searchParallelEnabled() {
		return s.searchObservationsAllSequential(ctx, projects, query, matchMode, scope, limit)
	}
	return s.searchObservationsAllParallel(ctx, projects, query, matchMode, scope, limit)
}

// searchObservationsAllSequential is the rollback-boundary path (SKILLGRID_
// SEARCH_PARALLEL=0): one store after another, missing stores skipped.
func (s *Service) searchObservationsAllSequential(ctx context.Context, projects []string, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	var collected []storeRanked
	for _, pid := range projects {
		if ctx.Err() != nil {
			break
		}
		res, err := scopedSearch(s, ctx, pid, query, matchMode, scope, limit)
		if err != nil {
			warnFunc("parallel search: skipping store %s: %v", pid, err)
			continue
		}
		collected = append(collected, rankHits(res)...)
	}
	return mergeRanked(collected, limit), nil
}

// searchObservationsAllParallel searches every store concurrently, bounded by
// a buffered-channel semaphore of size min(len(stores), NumCPU). Each
// goroutine acquires a slot (5s timeout — pool exhausted → skip with warning)
// before searching, and releases it after. Per-store results land on a
// buffered channel; the merge dedups on id/project and stable-sorts by
// (rank, UpdatedAt), so the output is deterministic given identical
// per-store results regardless of goroutine completion order.
func (s *Service) searchObservationsAllParallel(ctx context.Context, projects []string, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	workers := len(projects)
	if n := runtime.NumCPU(); workers > n {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	results := make(chan []storeRanked, len(projects))
	var wg sync.WaitGroup
	for _, pid := range projects {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			case <-time.After(semaphoreAcquireTimeout):
				warnFunc("parallel search: skipping store %s: semaphore acquire timed out after %v", pid, semaphoreAcquireTimeout)
				return
			}
			defer func() { <-sem }()
			res, err := scopedSearch(s, ctx, pid, query, matchMode, scope, limit)
			if err != nil {
				warnFunc("parallel search: skipping store %s: %v", pid, err)
				return
			}
			select {
			case results <- rankHits(res):
			case <-ctx.Done():
			}
		}(pid)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	var collected []storeRanked
	for res := range results {
		collected = append(collected, res...)
	}
	return mergeRanked(collected, limit), nil
}

// storeRanked is one merged hit: the observation plus its 0-based rank within
// its own store.
type storeRanked struct {
	obs  memory.Observation
	rank int
}

// rankHits tags each hit of one store's ranked result with its position.
func rankHits(res []memory.Observation) []storeRanked {
	out := make([]storeRanked, 0, len(res))
	for i, o := range res {
		out = append(out, storeRanked{obs: o, rank: i})
	}
	return out
}

// mergeRanked dedups on id/project (first occurrence wins — deterministic,
// since the per-store result order is stable for the same inputs) and
// stable-sorts by (rank, UpdatedAt desc). The stable sort makes the merged
// output independent of goroutine completion order.
func mergeRanked(collected []storeRanked, limit int) []memory.Observation {
	seen := map[string]bool{}
	deduped := collected[:0]
	for _, r := range collected {
		key := strconv.FormatInt(r.obs.ID, 10) + "/" + r.obs.Project
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, r)
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].rank != deduped[j].rank {
			return deduped[i].rank < deduped[j].rank
		}
		return deduped[i].obs.UpdatedAt > deduped[j].obs.UpdatedAt
	})
	out := make([]memory.Observation, 0, len(deduped))
	for _, r := range deduped {
		out = append(out, r.obs)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// warnFunc is the warning sink for the parallel search (change 014, step 03).
// Tests may replace it to capture warnings; production mirrors the service's
// existing stderr "warn:" convention. Guarded by warnMu.
var warnMu sync.Mutex
var warnFunc = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warn: "+format+"\n", args...)
}

// SearchObservationsScoped runs FTS over observations, restricting to a
// visibility scope when scope is non-empty (project|user|global).
func (s *Service) SearchObservationsScoped(ctx context.Context, projectID, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.SearchWithScope(ctx, query, matchMode, scope, limit)
}

// SearchAllProjects runs the FTS query across every store in the data dir and
// returns the unified, rank-merged result set. This is the backing for
// mem_search(all_projects=true) and rescues memories stranded under a
// directory-hash store for the same logical project.
func (s *Service) SearchAllProjects(ctx context.Context, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	return s.SearchObservationsAll(ctx, query, matchMode, scope, limit)
}

// Unify consolidates one or more source project stores into a single canonical
// project. It is the admin-facing wrapper around MergeProjects for cases where
// the caller wants to fold several names at once (e.g. three directory-hash
// variants of the same repo). For each source it records the alias and
// copies + re-tags rows, so a single mem_search(all_projects=false, project=
// canonical) then returns the combined history.
func (s *Service) Unify(ctx context.Context, canonical string, sources ...string) (int, error) {
	canonical = strings.TrimSpace(canonical)
	if canonical == "" {
		return 0, fmt.Errorf("canonical project is required")
	}
	total := 0
	for _, src := range sources {
		src = strings.TrimSpace(src)
		if src == "" || src == canonical {
			continue
		}
		moved, _, err := s.MergeProjects(ctx, src, canonical)
		if err != nil {
			return total, err
		}
		total += moved
	}
	return total, nil
}

// ListReviews returns observations due for review, oldest review_after first.
func (s *Service) ListReviews(ctx context.Context, projectID string, limit int) ([]memory.ReviewDue, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.ListReviews(ctx, limit)
}

// ProjectInfo describes the resolved project for cwd: id, source, all known
// projects, plus — when cwd is a parent of several git repositories — the
// candidate list so the caller (typically the agent) can pick one and retry
// the write with the chosen name.
type ProjectInfo struct {
	Project           string   `json:"project"`
	Source            string   `json:"source"`
	Directory         string   `json:"directory"`
	Projects          []string `json:"projects"`
	Ambiguous         bool     `json:"ambiguous,omitempty"`
	AvailableProjects []string `json:"available_projects,omitempty"`
	Warning           string   `json:"warning,omitempty"`
	SeedID            string   `json:"seed_id,omitempty"`
}

// CurrentProject returns the resolved project for cwd along with all
// available projects.
func (s *Service) CurrentProject(directory string) (ProjectInfo, error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return ProjectInfo{}, err
	}
	res, resErr := project.ResolveDetailed(absDir)
	projects, err := s.ListProjects()
	if err != nil {
		return ProjectInfo{}, err
	}
	out := ProjectInfo{
		Project:   res.ID,
		Source:    string(res.Source),
		Directory: absDir,
		Projects:  projects,
	}
	if resErr != nil {
		var amb *project.AmbiguousProjectError
		if errors.As(resErr, &amb) {
			out.Ambiguous = true
			out.AvailableProjects = res.Available
		}
	}
	if res.Warning != "" {
		out.Warning = res.Warning
	}
	if res.SeedID != "" {
		out.SeedID = res.SeedID
	}
	return out, nil
}

// MemoryDoctor describes mnemonic store health for a project.
type MemoryDoctor struct {
	SchemaVersion   int            `json:"schema_version"`
	WALMode         string         `json:"wal_mode,omitempty"`
	Observations    int            `json:"observations"`
	ObservationsFTS int            `json:"observations_fts"`
	Files           int            `json:"files"`
	Chunks          int            `json:"chunks"`
	ChunksFTS       int            `json:"chunks_fts"`
	WebCache        int            `json:"web_cache"`
	WebCacheFTS     int            `json:"web_cache_fts"`
	Prompts         int            `json:"prompts"`
	ByType          map[string]int `json:"by_type"`
	DiskSizeBytes   int64          `json:"disk_size_bytes"`
	FTSIntegrityOK  bool           `json:"fts_integrity_ok"`
	FTSDrift        int            `json:"fts_drift"`
}

// OrientResult is the Tier-1 orientation answer for one symbol: its metadata,
// the file TOC (all symbols in the file), a signature, and linked rationale.
type OrientResult struct {
	Found     bool             `json:"found"`
	Symbol    map[string]any   `json:"symbol,omitempty"`
	FileTOC   []map[string]any `json:"file_toc,omitempty"`
	Signature string           `json:"signature,omitempty"`
	List      []map[string]any `json:"list,omitempty"`
	Rationale []map[string]any `json:"rationale,omitempty"`
	Reason    string           `json:"reason,omitempty"` // not-found note
	// Processes is the additive process-participation field (008 step 02):
	// the processes this symbol takes part in, each with its step position
	// (step N/M). Empty (not an error) when the symbol is in no process. It
	// never changes the fields above (005's contract is preserved).
	Processes []process.Participation `json:"processes,omitempty"`
}

// OrientSymbol returns Tier-1 orientation for a resolved symbol: signature,
// file TOC, map, list, and metadata, plus linked rationale. An unknown symbol
// returns Found=false with a not-found reason (no fabricated symbol).
func (s *Service) OrientSymbol(ctx context.Context, projectID, symbol string) (*OrientResult, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return orientSymbol(h.store.DB, symbol)
}

// CodeGrep runs a structural by-example grep over root, index-free.
func (s *Service) CodeGrep(ctx context.Context, root, pattern string) (*search.GrepResult, error) {
	if pattern == "" {
		return nil, fmt.Errorf("code_grep: pattern is required")
	}
	return search.GrepByExample(root, pattern)
}

// orientSymbol resolves symbol (exact or FTS) and returns its orientation.
func orientSymbol(db *sql.DB, symbol string) (*OrientResult, error) {
	// Resolve the symbol: exact name match first (deterministic), then
	// identifier-FTS. Multiple exact matches are the first (lowest id) —
	// orientation is a single-symbol answer, not a candidate list (that is
	// step 03's code_impact job).
	row := db.QueryRow(`
		SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
		       s.start_line, s.end_line, f.path, s.file_id
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.name = ?
		ORDER BY s.id LIMIT 1`, symbol)
	var id, fileID int64
	var name, qualified, kind, lang, sig string
	var startLine, endLine int
	var path string
	err := row.Scan(&id, &name, &qualified, &kind, &lang, &sig, &startLine, &endLine, &path, &fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback: identifier-FTS.
			hits, e2 := search.SymbolFTS(db, symbol, 1)
			if e2 != nil {
				return nil, e2
			}
			if len(hits) == 0 {
				return &OrientResult{Found: false, Reason: "symbol not found: " + symbol}, nil
			}
			hit := hits[0]
			row2 := db.QueryRow(`
				SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
				       s.start_line, s.end_line, f.path, s.file_id
				FROM symbols s INNER JOIN files f ON f.id = s.file_id
				WHERE s.id = ?`, hit.ID)
			if err := row2.Scan(&id, &name, &qualified, &kind, &lang, &sig, &startLine, &endLine, &path, &fileID); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	out := &OrientResult{
		Found: true,
		Symbol: map[string]any{
			"id":             id,
			"name":           name,
			"qualified_name": qualified,
			"kind":           kind,
			"language":       lang,
			"path":           path,
			"start_line":     startLine,
			"end_line":       endLine,
		},
		Signature: sig,
	}

	// File TOC + list: every symbol in the file, ordered by start line.
	tocRows, err := db.Query(`
		SELECT id, name, qualified_name, kind, start_line, end_line
		FROM symbols WHERE file_id = ? ORDER BY start_line, id`, fileID)
	if err != nil {
		return out, nil
	}
	for tocRows.Next() {
		var tID int64
		var tName, tQualified, tKind string
		var tStart, tEnd int
		if err := tocRows.Scan(&tID, &tName, &tQualified, &tKind, &tStart, &tEnd); err != nil {
			tocRows.Close()
			return out, nil
		}
		entry := map[string]any{
			"id":         tID,
			"name":       tName,
			"kind":       tKind,
			"start_line": tStart,
			"end_line":   tEnd,
		}
		out.FileTOC = append(out.FileTOC, entry)
	}
	tocRows.Close()
	out.List = out.FileTOC

	// Rationale linked to this symbol.
	rationaleRows, err := db.Query(`SELECT text, kind, line FROM rationale WHERE symbol_id = ? ORDER BY line`, id)
	if err == nil {
		for rationaleRows.Next() {
			var text, kind string
			var line int
			if rationaleRows.Scan(&text, &kind, &line) == nil {
				out.Rationale = append(out.Rationale, map[string]any{
					"text": text,
					"kind": kind,
					"line": line,
				})
			}
		}
		rationaleRows.Close()
	}
	return out, nil
}

// ImpactOptions tunes a blast-radius traversal (narrowing + confidence).
type ImpactOptions struct {
	File          string
	UID           string
	Kind          string
	MinConfidence string
	MaxDepth      int
}

// ImpactResultDTO is the code_impact answer. Either Target+tiers is set (a
// single resolved symbol) or Candidates is set (a ranked ambiguous list —
// never a silent pick).
type ImpactResultDTO struct {
	Found      bool               `json:"found"`
	Ambiguous  bool               `json:"ambiguous,omitempty"`
	Candidates []graph.Symbol     `json:"candidates,omitempty"`
	Target     *graph.Symbol      `json:"target,omitempty"`
	WillBreak  []graph.ImpactEdge `json:"will_break,omitempty"`
	Likely     []graph.ImpactEdge `json:"likely_affected,omitempty"`
	Excluded   int                `json:"excluded_low_confidence,omitempty"`
	Reason     string             `json:"reason,omitempty"`
}

func (r *ImpactResultDTO) Summary() string {
	if r == nil {
		return ""
	}
	if r.Ambiguous {
		return fmt.Sprintf("ambiguous: %d candidates (narrow with file/uid/kind)", len(r.Candidates))
	}
	return fmt.Sprintf("will_break: %d, likely_affected: %d", len(r.WillBreak), len(r.Likely))
}

// Impact is the risk-tiered blast-radius DTO for the graph package result.
type Impact = graph.ImpactResult

// CodeImpact returns the risk-tiered blast radius for symbol, honoring the
// narrowing + confidence options. A name matching several symbols returns a
// ranked candidate list (never a silent pick); an unknown symbol returns an
// empty (not-found) result.
func (s *Service) CodeImpact(ctx context.Context, projectID, symbol string, opts ImpactOptions) (*ImpactResultDTO, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	res, err := graph.Resolve(ctx, h.store.DB, symbol, graph.ResolveFilter{
		File: opts.File, UID: opts.UID, Kind: opts.Kind,
	})
	if err != nil {
		return nil, err
	}
	if res.NotFound {
		return &ImpactResultDTO{Found: false, Reason: "symbol not found: " + symbol}, nil
	}
	if res.Ambiguous {
		ranked, err := graph.RankCandidates(ctx, h.store.DB, res.Matches)
		if err != nil {
			return nil, err
		}
		return &ImpactResultDTO{Found: true, Ambiguous: true, Candidates: ranked}, nil
	}
	impact, err := graph.Impact(ctx, h.store.DB, res.Target, graph.ImpactOptions{
		MinConfidence: opts.MinConfidence,
		MaxDepth:      opts.MaxDepth,
	})
	if err != nil {
		return nil, err
	}
	target := res.Target
	return &ImpactResultDTO{
		Found:     true,
		Target:    &target,
		WillBreak: impact.WillBreak,
		Likely:    impact.Likely,
		Excluded:  impact.Excluded,
	}, nil
}

// NeighborsDTO is the graph neighbor answer (every edge confidence-labeled).
type NeighborsDTO struct {
	Symbol *graph.Symbol `json:"symbol,omitempty"`
	Edges  []graph.Edge  `json:"edges"`
	Reason string        `json:"reason,omitempty"`
}

// GrabNeighbors returns a symbol's confidence-labeled edges for view. An
// unknown symbol returns not-found (no invented edges).
func (s *Service) GrabNeighbors(ctx context.Context, projectID, view, symbol string) (*NeighborsDTO, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	res, err := graph.Resolve(ctx, h.store.DB, symbol, graph.ResolveFilter{})
	if err != nil {
		return nil, err
	}
	if res.NotFound {
		return &NeighborsDTO{Edges: []graph.Edge{}, Reason: "symbol not found: " + symbol}, nil
	}
	if res.Ambiguous {
		// Report ambiguity as a not-found-with-candidates rather than a
		// silent pick; graph views are single-symbol answers.
		return &NeighborsDTO{Edges: []graph.Edge{}, Reason: fmt.Sprintf("ambiguous symbol %q: %d candidates (narrow with file/uid/kind)", symbol, len(res.Matches))}, nil
	}
	v := graph.View(view)
	edges, err := graph.Neighbors(ctx, h.store.DB, res.Target, v)
	if err != nil {
		return nil, err
	}
	sym := res.Target
	return &NeighborsDTO{Symbol: &sym, Edges: edges}, nil
}

// PathDTO is the code_path answer.
type PathDTO struct {
	Found      bool              `json:"found"`
	Path       []graph.Edge      `json:"path,omitempty"`
	GraphStops *graph.GraphStops `json:"graph_stops,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

// CodePath returns the shortest edge path between two symbols or a
// where-the-graph-stops answer. Unknown endpoints yield a not-found reason.
func (s *Service) CodePath(ctx context.Context, projectID, from, to string) (*PathDTO, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	resFrom, err := graph.Resolve(ctx, h.store.DB, from, graph.ResolveFilter{})
	if err != nil {
		return nil, err
	}
	resTo, err := graph.Resolve(ctx, h.store.DB, to, graph.ResolveFilter{})
	if err != nil {
		return nil, err
	}
	if resFrom.NotFound || resTo.NotFound {
		missing := from
		if !resFrom.NotFound {
			missing = to
		}
		return &PathDTO{Found: false, Reason: "symbol not found: " + missing}, nil
	}
	res, err := graph.Path(ctx, h.store.DB, resFrom.Target, resTo.Target)
	if err != nil {
		return nil, err
	}
	return &PathDTO{Found: res.Found, Path: res.Path, GraphStops: res.GraphStops}, nil
}

// ExplainDTO is the code_explain answer.
type ExplainDTO struct {
	Found       bool          `json:"found"`
	Symbol      *graph.Symbol `json:"symbol,omitempty"`
	Degree      int           `json:"degree"`
	Connections []graph.Conn  `json:"connections"`
	Reason      string        `json:"reason,omitempty"`
}

// CodeExplain returns a symbol's node, degree, and connections ranked by the
// neighbor's degree. Unknown symbol returns not-found.
func (s *Service) CodeExplain(ctx context.Context, projectID, symbol string) (*ExplainDTO, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	res, err := graph.Resolve(ctx, h.store.DB, symbol, graph.ResolveFilter{})
	if err != nil {
		return nil, err
	}
	if res.NotFound {
		return &ExplainDTO{Found: false, Connections: []graph.Conn{}, Reason: "symbol not found: " + symbol}, nil
	}
	out, err := graph.Explain(ctx, h.store.DB, res.Target)
	if err != nil {
		return nil, err
	}
	sym := res.Target
	return &ExplainDTO{Found: true, Symbol: &sym, Degree: out.Degree, Connections: out.Connections}, nil
}

// ExploreSourceSpan is one symbol's verbatim source slice.
type ExploreSourceSpan struct {
	Symbol    string
	StartLine int
	EndLine   int
	Content   string
}

// ExploreFlowEdge is one call-flow hop between returned symbols.
type ExploreFlowEdge struct {
	From       string
	To         string
	Kind       string
	Confidence string
	Line       int
}

// ExploreResult is the composite code_explore answer.
type ExploreResult struct {
	Symbol string
	Source map[string][]ExploreSourceSpan
	Flow   []ExploreFlowEdge
	Impact *ImpactResultDTO
}

// CodeExplore assembles the composite answer: the symbol's source (grouped by
// file), its call-flow neighbors (including INFERRED dynamic-dispatch hops),
// and a blast-radius summary.
func (s *Service) CodeExplore(ctx context.Context, projectID, symbol string) (*ExploreResult, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	res, err := graph.Resolve(ctx, h.store.DB, symbol, graph.ResolveFilter{})
	if err != nil {
		return nil, err
	}
	out := &ExploreResult{Symbol: symbol, Source: map[string][]ExploreSourceSpan{}}
	if res.NotFound {
		return out, nil
	}
	target := res.Target
	// Pull verbatim source for the target symbol.
	spans, err := s.readSymbolSpans(ctx, h.store.DB, []graph.Symbol{target})
	if err == nil {
		for path, ss := range spans {
			out.Source[path] = ss
		}
	}
	// Gather the call-flow: forward (callees) and reverse (callers) edges,
	// including INFERRED dynamic-dispatch hops.
	callees, _ := graph.Neighbors(ctx, h.store.DB, target, graph.ViewCallees)
	callers, _ := graph.Neighbors(ctx, h.store.DB, target, graph.ViewCallers)
	related := []graph.Symbol{}
	seen := map[int64]bool{target.ID: true}
	for _, e := range append(append([]graph.Edge{}, callees...), callers...) {
		other := e.To
		if e.From.ID == target.ID && e.To.ID != target.ID {
			other = e.To
		}
		if e.From.ID != target.ID && e.To.ID != target.ID {
			other = e.To
		}
		if other.ID == 0 {
			continue
		}
		out.Flow = append(out.Flow, ExploreFlowEdge{
			From:       nameOf(target, other),
			To:         nameOf(other, target),
			Kind:       e.Kind,
			Confidence: e.Confidence,
			Line:       e.Line,
		})
		if !seen[other.ID] {
			seen[other.ID] = true
			related = append(related, other)
		}
	}
	if len(related) > 0 {
		if spans, err := s.readSymbolSpans(ctx, h.store.DB, related); err == nil {
			for path, ss := range spans {
				out.Source[path] = append(out.Source[path], ss...)
			}
		}
	}
	out.Impact, err = s.CodeImpact(ctx, projectID, symbol, ImpactOptions{})
	if err != nil {
		out.Impact = &ImpactResultDTO{}
	}
	return out, nil
}

// nameOf returns the from-side name for a flow hop relative to target.
func nameOf(a, b graph.Symbol) string {
	if a.ID != 0 {
		return a.Name
	}
	if b.ID != 0 {
		return b.Name
	}
	return ""
}

// readSymbolSpans returns each symbol's verbatim source grouped by file.
func (s *Service) readSymbolSpans(ctx context.Context, db *sql.DB, syms []graph.Symbol) (map[string][]ExploreSourceSpan, error) {
	out := map[string][]ExploreSourceSpan{}
	for _, sym := range syms {
		if sym.ID == 0 {
			continue
		}
		res, err := readIndexedCode(db, sym.Path, sym.StartLine, sym.EndLine)
		if err != nil {
			continue
		}
		text, _ := res["text"].(string)
		out[sym.Path] = append(out[sym.Path], ExploreSourceSpan{
			Symbol:    sym.Name,
			StartLine: sym.StartLine,
			EndLine:   sym.EndLine,
			Content:   text,
		})
	}
	return out, nil
}

// RunCodeIndex runs incremental code indexing for directory.
func (s *Service) RunCodeIndex(ctx context.Context, directory string) (codeindex.Stats, error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return codeindex.Stats{}, err
	}
	defer cleanup()
	cfg := config.Load(directory)
	idxCfg := codeindex.Config{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		ChunkLines:   cfg.ChunkLines,
		ChunkOverlap: cfg.ChunkOverlap,
		MaxFileSize:  cfg.MaxFileSize,
	}
	idx := codeindex.New(h.store)
	if emb := resolveEmbedder(h.root); emb != nil {
		idx = idx.WithEmbedder(emb)
	}
	return idx.Run(ctx, directory, idxCfg)
}

// RunCodeIndexPDG runs incremental code indexing for directory with the opt-in
// --pdg (per-function CFG + PDG) and --lsp (LSP-resolved member-call edges)
// passes toggled. It mirrors RunCodeIndex so the CLI `index --pdg/--lsp` flags
// reach the same incremental transaction; both default to the static index when
// off, so pdg=false/lsp=false is byte-for-byte the non---pdg path.
func (s *Service) RunCodeIndexPDG(ctx context.Context, directory string, pdg, lsp bool) (codeindex.Stats, error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return codeindex.Stats{}, err
	}
	defer cleanup()
	cfg := config.Load(directory)
	idxCfg := codeindex.Config{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		ChunkLines:   cfg.ChunkLines,
		ChunkOverlap: cfg.ChunkOverlap,
		MaxFileSize:  cfg.MaxFileSize,
		PDG:          pdg,
		LSP:          lsp,
	}
	idx := codeindex.New(h.store)
	if emb := resolveEmbedder(h.root); emb != nil {
		idx = idx.WithEmbedder(emb)
	}
	// The PDG/LSP passes are driven by Config.PDG/Config.LSP in Run; the
	// flags set them above so `index --pdg/--lsp` reach the same transaction.
	return idx.Run(ctx, directory, idxCfg)
}

// ReindexStructural runs a structural-only incremental re-index for directory:
// it syncs chunks/symbols/edges (the 005 extraction) but does NOT attach an
// embedder, so no model load or embedding call happens. This is the
// pull-at-query fingerprint gate's re-index (structural-only, embedder-free)
// and the watcher's comfort-layer sync. Advisory: a failure keeps the old
// index live.
func (s *Service) ReindexStructural(ctx context.Context, directory string, cfg codeindex.Config) (codeindex.Stats, error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return codeindex.Stats{}, err
	}
	defer cleanup()
	idx := codeindex.New(h.store)
	stats, err := idx.Run(ctx, directory, cfg)
	if err != nil {
		return codeindex.Stats{}, err
	}
	// Persist the extractor stamp so the next query's fingerprint gate knows
	// the structural re-index ran under this stamp (a stamp change invalidates
	// the fingerprint and forces a re-walk). Advisory: a failure here does not
	// fail the re-index (the 005 extraction is already committed).
	if err := codeindex.StoreFingerprint(h.store.DB, directory, cfg, codeindex.ExtractorStamp()); err != nil {
		fmt.Fprintf(os.Stderr, "warn: fingerprint stamp: %v\n", err)
	}
	return stats, nil
}

// CodeHybridResult is the hybrid code search answer (per-signal provenance on
// every hit).
type CodeHybridResult = hybrid.Result

// CodeHybridSearch runs the offline hybrid code search (FTS + deterministic
// signals + optional embeddings) for projectID. The embedder is resolved from
// config; a down/missing embedder degrades to FTS + signals (the floor).
func (s *Service) CodeHybridSearch(ctx context.Context, projectID, query string, limit int) (*hybrid.Result, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	emb := s.warmSearchEmbedder(h.root)
	return hybrid.Search(ctx, h.store.DB, query, hybrid.Options{
		Limit:    limit,
		Embedder: emb,
	})
}

// CodeSemanticSearch runs the vector leg only: symbol-level hits return the
// named symbol + file + line; chunk-level hits return the line range. With no
// active embedder it returns an empty result (the hybrid tool keeps the
// FTS+signals floor).
func (s *Service) CodeSemanticSearch(ctx context.Context, projectID, query string, limit int) (*hybrid.Result, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	emb := s.warmSearchEmbedder(h.root)
	return hybrid.Search(ctx, h.store.DB, query, hybrid.Options{
		Limit:    limit,
		Semantic: true,
		Embedder: emb,
	})
}

// warmSearchEmbedder returns the warm embedder (load-once, heartbeat-kept) for
// the search path, or nil when the provider is off (the FTS+signals floor). It
// is the response-path hook that keeps the ONNX model warm across search calls
// without a per-call model load.
func (s *Service) warmSearchEmbedder(configRoot string) embedder.Embedder {
	cfg := config.Load(configRoot)
	if cfg.Embedder.Provider == "" || cfg.Embedder.Provider == "off" {
		return nil // off → FTS floor (no vector leg, no RAM)
	}
	we := warmFor(cfg.Embedder.Provider, configRoot)
	// Get is the heartbeat: it records the access (resets the idle-evict window)
	// and returns the resident embedder (loading once on first use).
	return we.Get(context.Background())
}

// CodeEmbeddingStatus reports the active provider/model, vector dimension,
// embedded counts, and model-swap guard state (read-only).
func (s *Service) CodeEmbeddingStatus(ctx context.Context, projectID string) (map[string]any, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	cfg := config.Load(h.root)
	emb := resolveEmbedder(h.root)
	status := map[string]any{
		"provider": cfg.Embedder.Provider,
	}
	if emb == nil {
		status["active"] = false
		status["model"] = ""
		status["dimension"] = 0
		status["reason"] = "embedder off"
	} else {
		status["active"] = true
		status["model"] = emb.Model()
		status["dimension"] = emb.Dimension()
	}
	var symCount int
	_ = h.store.DB.QueryRow(`SELECT COUNT(*) FROM embeddings`).Scan(&symCount)
	status["embedded_symbols"] = symCount
	var model sql.NullString
	_ = h.store.DB.QueryRow(`SELECT value FROM embed_meta WHERE key = 'embedding_model'`).Scan(&model)
	status["indexed_model"] = model.String
	return status, nil
}

// toAsym converts config params to embedder params.
func toAsym(p config.EmbedderParams) embedder.AsymParams {
	return embedder.AsymParams{
		Instructions: p.Instructions,
		InputType:    p.InputType,
		MaxTokens:    p.MaxTokens,
	}
}

// warmEmbedder is the process-wide warm embedder cache (load-once, idle-evict,
// heartbeat-kept). It wraps the configured embedder so the ONNX model loads
// once and is reused across search calls (no per-call model load). It is
// created lazily on the first search and keyed by the resolved provider.
var (
	warmMu       sync.Mutex
	warmEmb      *embedder.WarmEmbedder
	warmProvider string
)

// warmFor returns the process-wide warm embedder for the given provider,
// building it on first use. The warm cache loads the model once and reuses it;
// an idle-evict timer (external to this function) calls warmEmb.CheckIdle, and
// each search call's Get is the heartbeat that keeps it alive while connected.
// configRoot is the project root (h.root) — the warm cache resolves the
// embedder from the project's config, not the CWD, so a call where h.root !=
// CWD still picks the correct embedder.
func warmFor(provider, configRoot string) *embedder.WarmEmbedder {
	warmMu.Lock()
	defer warmMu.Unlock()
	if warmEmb == nil || warmProvider != provider {
		warmEmb = embedder.NewWarm(embedder.WarmConfig{
			Factory:     func() embedder.Embedder { return buildEmbedderFromProvider(provider, configRoot) },
			IdleTimeout: embedder.DefaultIdleTimeout,
		})
		warmProvider = provider
	}
	return warmEmb
}

// buildEmbedderFromProvider builds a concrete embedder from a provider name
// (the warm cache's factory). It re-reads the project's config (configRoot, the
// project root — not the CWD) at load time so a model swap is picked up on
// reload.
func buildEmbedderFromProvider(provider, configRoot string) embedder.Embedder {
	if provider == "" || provider == "off" {
		return nil // Null Adapter (FTS floor)
	}
	cfg := config.Load(configRoot)
	if provider != cfg.Embedder.Provider {
		// A config provider different from the requested one: honor the config
		// (the warm cache is keyed by the config's actual provider).
		provider = cfg.Embedder.Provider
	}
	return resolveEmbedder(configRoot)
}

// WarmEmbedderHandle returns the process-wide warm embedder (for the idle-evict
// timer + MCP heartbeat wiring). It is nil until the first search.
func WarmEmbedderHandle() *embedder.WarmEmbedder {
	warmMu.Lock()
	defer warmMu.Unlock()
	return warmEmb
}

// resetWarmCache clears the process-wide warm cache (test hook).
func resetWarmCache() {
	warmMu.Lock()
	warmEmb = nil
	warmProvider = ""
	warmMu.Unlock()
}

// resolveEmbedder builds the process embedder from config. An empty/nil
// embedder means "off" — the FTS+signals floor applies.
func resolveEmbedder(configRoot string) embedder.Embedder {
	cfg := config.Load(configRoot)
	switch cfg.Embedder.Provider {
	case "external":
		return embedder.NewExternal(embedder.ExternalConfig{
			BaseURL:   cfg.Embedder.BaseURL,
			Model:     cfg.Embedder.Model,
			APIKey:    cfg.Embedder.APIKey,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  toAsym(cfg.Embedder.Indexing),
			Query:     toAsym(cfg.Embedder.Query),
		})
	case "off", "":
		return nil
	default: // "onnx" is the default
		return embedder.NewOnnx(embedder.OnnxConfig{
			Model:     cfg.Embedder.Model,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  toAsym(cfg.Embedder.Indexing),
			Query:     toAsym(cfg.Embedder.Query),
		})
	}
}

// Open opens a ProjectHandle for an explicit project id (config root ".").
// It aborts with an error on empty or invalid ids — blank ids are rejected
// up front and ".."-containing ids by store.Open — and never returns a
// partial handle.
func (s *Service) Open(projectID string) (*ProjectHandle, func(), error) {
	return s.openProject(projectID, ".")
}

// OpenForCWD opens project-scoped services from the current working directory.
func (s *Service) OpenForCWD() (*ProjectHandle, func(), error) {
	return s.openProjectFromCWD()
}

// OpenForDirectory opens project-scoped services for directory.
func (s *Service) OpenForDirectory(directory string) (*ProjectHandle, func(), error) {
	return s.openProjectForDirectory(directory)
}

// ProjectID returns the project ID for an open handle.
func (h *ProjectHandle) ProjectID() string { return h.projectID }

// Memory returns the memory service for an open handle.
func (h *ProjectHandle) Memory() *memory.Service { return h.memory }

// Web returns the webcache service for an open handle.
func (h *ProjectHandle) Web() *webcache.Service { return h.web }

// Store returns the underlying store for an open handle.
func (h *ProjectHandle) Store() *store.Store { return h.store }

// Root returns the workspace directory this handle was opened for (the
// project root that owns the .skillgrid/ scratch tree, including the relay
// cleave bundle under .skillgrid/.cleave/). Used by the Session Relay to know
// where on disk the cleave files live.
func (h *ProjectHandle) Root() string { return h.root }

// CommunityResult is the code_communities answer (LLM-free labeled
// subsystems + a stable content-hash cache key).
type CommunityResult = community.Result

// CommunityOptions tunes the community pass.
type CommunityOptions = community.Options

// CodeCommunities runs the seeded Leiden community pass over projectID's
// indexed graph and returns the labeled subsystems (with the cache key and any
// non-fatal warning). The partition is advisory, never load-bearing.
func (s *Service) CodeCommunities(ctx context.Context, projectID string, opts CommunityOptions) (*CommunityResult, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return community.Detect(ctx, h.store.DB, opts)
}

// GodNode is a degree-ranked symbol (the most-connected concepts).
type GodNode = community.GodNode

// CodeGodNodes returns the most-connected symbols in projectID ranked by
// degree. excludeHubs suppresses utility super-hubs from the ranking.
func (s *Service) CodeGodNodes(ctx context.Context, projectID string, excludeHubs bool, limit int) ([]GodNode, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	rows, err := h.store.DB.Query(`SELECT id FROM symbols`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	gods, err := community.RankGodNodes(h.store.DB, ids, excludeHubs)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(gods) > limit {
		gods = gods[:limit]
	}
	return gods, nil
}

// CommunityExplanation is the code_explain_community answer: the community's
// members + key entry points (top god nodes).
type CommunityExplanation struct {
	Found    bool                `json:"found"`
	ID       int                 `json:"id"`
	Label    string              `json:"label"`
	Members  []map[string]any    `json:"members"`
	EntryPts []community.GodNode `json:"entry_points"`
	Reason   string              `json:"reason,omitempty"`
}

// CodeExplainCommunity returns a community's members + key entry points. An
// unknown community id returns Found=false with a not-found reason (no
// invented community).
func (s *Service) CodeExplainCommunity(ctx context.Context, projectID string, id int) (*CommunityExplanation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	var maxID int
	if err := h.store.DB.QueryRow(`SELECT COALESCE(MAX(id), -1) FROM communities`).Scan(&maxID); err != nil {
		return nil, err
	}
	if id < 0 || id > maxID {
		return &CommunityExplanation{Found: false, Reason: fmt.Sprintf("community %d not found (0-%d exist)", id, maxID)}, nil
	}
	rows, err := h.store.DB.Query(`SELECT symbol_id FROM communities WHERE id = ? ORDER BY symbol_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memberIDs []int64
	for rows.Next() {
		var mid int64
		if err := rows.Scan(&mid); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, mid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return &CommunityExplanation{Found: false, Reason: fmt.Sprintf("community %d not found", id)}, nil
	}
	label := "community-" + fmt.Sprintf("%d", id)
	_ = h.store.DB.QueryRow(`SELECT label FROM community_meta WHERE id = ?`, id).Scan(&label)

	out := &CommunityExplanation{Found: true, ID: id, Label: label, EntryPts: []community.GodNode{}}
	for _, mid := range memberIDs {
		var name, kind, lang, path string
		var startLine, endLine int
		if err := h.store.DB.QueryRow(`
			SELECT s.name, s.kind, COALESCE(s.language,''), f.path, s.start_line, s.end_line
			FROM symbols s INNER JOIN files f ON f.id = s.file_id WHERE s.id = ?`, mid).
			Scan(&name, &kind, &lang, &path, &startLine, &endLine); err == nil {
			out.Members = append(out.Members, map[string]any{
				"id":         mid,
				"name":       name,
				"kind":       kind,
				"language":   lang,
				"path":       path,
				"start_line": startLine,
				"end_line":   endLine,
			})
		}
	}
	gods, err := community.RankGodNodes(h.store.DB, memberIDs, false)
	if err == nil {
		out.EntryPts = gods
	}
	return out, nil
}

// ProcessResult is the code_processes answer (the precomputed flows).
type ProcessResult = process.Result

// ProcessOptions tunes the process pass.
type ProcessOptions = process.RunOptions

// ProcessEntry is one seed for the process pass.
type ProcessEntry = process.Entry

// ProcessLLM is the pluggable labeler for the process pass (nil = no LLM →
// flows cached unlabeled). Tests inject a stub; there is no CGo LLM client.
type ProcessLLM = process.LLM

// CodeProcessTrace runs the precomputed process pass over projectID's
// entries, persisting processes/process_steps (cross-community flag +
// content-hash cache + LLM labels). The pass is advisory, never
// load-bearing. Index-time wiring (Indexer.Run calling this pass with a real
// LLM impl) is step 03 scope (03.8); step 02 runs it query-time, so
// code_processes is CWD-scoped until that hook lands.
func (s *Service) CodeProcessTrace(ctx context.Context, projectID string, entries []ProcessEntry, llm ProcessLLM, opts ProcessOptions) (*ProcessResult, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return process.Run(ctx, h.store.DB, entries, llm, opts)
}

// CodeProcesses returns the stored precomputed flows for projectID (one call,
// no per-query traversal) — the backing for code_processes.
func (s *Service) CodeProcesses(ctx context.Context, projectID string) ([]process.Process, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return process.List(h.store.DB)
}

// CodeProcess returns one stored process by name with its full step-by-step
// trace — the backing for code_process <name>. Unknown name → sql.ErrNoRows.
func (s *Service) CodeProcess(ctx context.Context, projectID, name string) (*process.Process, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return process.Get(h.store.DB, name)
}

// CodeProcessParticipations returns the processes symbolID participates in,
// each with its step position (step N/M). Empty when the symbol is in none.
func (s *Service) CodeProcessParticipations(ctx context.Context, projectID string, symbolID int64) ([]process.Participation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return process.Participations(h.store.DB, symbolID)
}

func readIndexedCode(db *sql.DB, path string, startLine, endLine int) (map[string]any, error) {
	var fileID int64
	err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not indexed: %s", path)
	}
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if startLine > 0 {
		if endLine <= 0 {
			endLine = startLine
		}
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ? AND start_line <= ? AND end_line >= ?
			ORDER BY start_line`,
			fileID, endLine, startLine,
		)
	} else {
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ?
			ORDER BY start_line`,
			fileID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var parts []string
	firstLine := 0
	lastLine := 0
	for rows.Next() {
		var chunkStart, chunkEnd int
		var text string
		if err := rows.Scan(&chunkStart, &chunkEnd, &text); err != nil {
			return nil, err
		}
		if firstLine == 0 || chunkStart < firstLine {
			firstLine = chunkStart
		}
		if chunkEnd > lastLine {
			lastLine = chunkEnd
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no indexed chunks for %s", path)
	}
	return map[string]any{
		"path":       path,
		"start_line": firstLine,
		"end_line":   lastLine,
		"text":       strings.Join(parts, "\n"),
	}, nil
}
