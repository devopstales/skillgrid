package skills

// The skill improvement loop (change 2026-10-24-global-llm-wiki, TICKET-07):
// the Cognee observe → inspect → amend → evaluate cycle over the
// code-registry skills (sandboxed scripts in the skills table, NOT agent
// SKILL.md files — the C1 resolution).
//
// Observe lives in execute.go (the C2 fix): every executed run logs a
// skill_usage row with outcome 'success' | 'failure' | 'error', so the table
// is a success-rate signal, not just an activity log. This file is the other
// three legs:
//
//   - Inspect: aggregate a skill's usage rows into an evidence summary
//     (outcome counts + success rate + exact-match top-N stderr patterns).
//   - Amend: the LLM (ADR-0023) proposes a patch from the evidence and the
//     current script; it is written to <code_path>.amended — never in place,
//     the original is untouched.
//   - Evaluate: compare the outcome rate before vs after the amendment
//     window. Regression → discard the .amended file (original stays).
//     Improvement → apply .amended over the original (a .orig backup of the
//     previous content is left behind, and the discard leg never restores
//     from it — a regression means "keep the original", and the original was
//     never modified) and commit the change to git.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// topPatternN caps the distinct stderr patterns Inspect returns.
const topPatternN = 5

// Outcome classifies a sandbox result into the skill_usage.outcome column:
// 'error' for a sandbox timeout (the run was killed, not failed), 'failure'
// for a non-zero exit, 'success' otherwise.
func (r SkillResult) Outcome() string {
	switch {
	case r.TimedOut:
		return "error"
	case r.ExitCode != 0:
		return "failure"
	default:
		return "success"
	}
}

// SkillPattern is one exact-match stderr group: how many logged failures
// produced this stderr string verbatim (the top-N failure pattern).
type SkillPattern struct {
	Stderr string `json:"stderr"`
	Count  int    `json:"count"`
}

// SkillEvidence is the inspect output: the outcome counts, the success rate
// (integer percent, rounded), and the top failure patterns. TotalRuns counts
// every logged outcome; SuccessRate is SuccessRuns/TotalRuns (100 when there
// are no runs — vacuously healthy).
type SkillEvidence struct {
	TotalRuns     int            `json:"total_runs"`
	SuccessRuns   int            `json:"success_runs"`
	FailureRuns   int            `json:"failure_runs"`
	SuccessRate   int            `json:"success_rate"`
	TopPatterns   []SkillPattern `json:"top_patterns"`
	LastRunAt     string         `json:"last_run_at,omitempty"`
	AmendedExists bool           `json:"amended_exists"`
}

// Inspect aggregates the skill's skill_usage rows into the evidence summary
// that grounds Amend. Grouping is exact-match on the stderr string (not
// fuzzy): the point is "the same error keeps happening", and a normalized
// group would hide the difference between two distinct failures that merely
// rhyme. Top patterns are the most frequent first, ties broken by stderr
// text for determinism.
func (s *Store) Inspect(ctx context.Context, name string) (SkillEvidence, error) {
	if s == nil || s.db == nil {
		return SkillEvidence{}, fmt.Errorf("skills store not initialized")
	}
	var sk Skill
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, name, language, COALESCE(description,''), code_path, created_at, updated_at
		 FROM skills WHERE name = ?`, name).
		Scan(&sk.ID, &sk.Name, &sk.Language, &sk.Description, &sk.CodePath, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return SkillEvidence{}, fmt.Errorf("skill not found: %q", name)
		}
		return SkillEvidence{}, fmt.Errorf("inspect skill: %w", err)
	}

	var ev SkillEvidence
	counts := map[string]int{}
	var lastRunAt sql.NullString
	rows, err := s.db.QueryContext(ctx,
		`SELECT COALESCE(outcome, 'success'), COALESCE(stderr, ''), timestamp
		 FROM skill_usage WHERE skill_id = ? ORDER BY timestamp`, sk.ID)
	if err != nil {
		return SkillEvidence{}, fmt.Errorf("inspect usage: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var outcome, stderr string
		if err := rows.Scan(&outcome, &stderr, &lastRunAt); err != nil {
			return SkillEvidence{}, fmt.Errorf("scan usage row: %w", err)
		}
		ev.TotalRuns++
		switch outcome {
		case "success":
			ev.SuccessRuns++
		case "error":
			ev.FailureRuns++ // an error outcome counts against the skill
		default: // 'failure' and any legacy value
			ev.FailureRuns++
		}
		if outcome != "success" && stderr != "" {
			counts[stderr]++
		}
	}
	if err := rows.Err(); err != nil {
		return SkillEvidence{}, fmt.Errorf("iterate usage: %w", err)
	}
	if ev.TotalRuns > 0 {
		ev.LastRunAt = lastRunAt.String
		ev.SuccessRate = (ev.SuccessRuns*100 + ev.TotalRuns/2) / ev.TotalRuns
	} else {
		ev.SuccessRate = 100
	}
	ev.TopPatterns = topPatterns(counts)
	if _, err := os.Stat(sk.CodePath + ".amended"); err == nil {
		ev.AmendedExists = true
	}
	return ev, nil
}

// topPatterns renders the count map as a top-N, count-desc slice (ties broken
// by stderr text) so the summary is stable across runs.
func topPatterns(counts map[string]int) []SkillPattern {
	out := make([]SkillPattern, 0, len(counts))
	for stderr, n := range counts {
		out = append(out, SkillPattern{Stderr: stderr, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Stderr < out[j].Stderr
	})
	if len(out) > topPatternN {
		out = out[:topPatternN]
	}
	return out
}

// amendSystem is the fixed system prompt for the patch proposer (the store
// stays LLM-agnostic: it sends prompt + code, it never parses a diff format
// — the contract is "return the full amended script").
const amendSystem = `You improve a failing agent skill script. You are given the skill's recent execution evidence (failure counts and exact stderr messages) and its current script. Return the complete amended script only — no explanation, no fences — fixing the observed failure without changing the skill's intended behavior.`

// Amend asks the LLM for a patch grounded in the skill's failure evidence
// and writes it to <code_path>.amended (NOT in place — the original file is
// untouched, so a bad patch is always discodable without git). It fails when
// no LLM is attached (auto_amend off) or when there is no logged failure to
// ground the patch in.
func (s *Store) Amend(ctx context.Context, name string) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("skills store not initialized")
	}
	if s.LLM == nil {
		return "", fmt.Errorf("no LLM attached: skill amend needs the shared LLM (mnemonic.llm.enabled)")
	}
	var codePath string
	if err := s.db.QueryRowContext(ctx,
		`SELECT code_path FROM skills WHERE name = ?`, name).Scan(&codePath); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("skill not found: %q", name)
		}
		return "", fmt.Errorf("amend lookup: %w", err)
	}

	ev, err := s.Inspect(ctx, name)
	if err != nil {
		return "", err
	}
	if ev.FailureRuns == 0 {
		return "", fmt.Errorf("skill %q has no logged failures to amend from — nothing to ground a patch in", name)
	}
	code, err := os.ReadFile(codePath)
	if err != nil {
		return "", fmt.Errorf("read skill code: %w", err)
	}

	prompt := fmt.Sprintf("Skill: %s (language: see file)\n\nExecution evidence:\n- %d total runs, %d%% success\n",
		name, ev.TotalRuns, ev.SuccessRate)
	for i, p := range ev.TopPatterns {
		prompt += fmt.Sprintf("- failure pattern %d (seen %dx): %s\n", i+1, p.Count, p.Stderr)
	}
	prompt += fmt.Sprintf("\nCurrent script:\n%s\n\nReturn the complete amended script only.", string(code))

	patch, err := s.LLM(ctx, amendSystem, prompt)
	if err != nil {
		return "", fmt.Errorf("amend LLM: %w", err)
	}
	patch = strings.TrimSpace(patch)
	if patch == "" {
		return "", fmt.Errorf("amend LLM returned an empty script")
	}
	// The model may still wrap the script in a fence even when told not to.
	patch = stripCodeFence(patch)

	amendedPath := codePath + ".amended"
	if err := os.WriteFile(amendedPath, []byte(patch+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("write amended skill: %w", err)
	}
	return amendedPath, nil
}

// stripCodeFence removes a single wrapping ``` fence (with or without a
// language tag) around the LLM's answer — a cheap robustness floor for
// models that ignore "no fences", never more.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := strings.TrimPrefix(s, "```")
	// Drop the optional language tag on the opening fence line.
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		if !strings.Contains(rest[:i], " ") {
			rest = rest[i+1:]
		}
	}
	// Drop the closing fence at the end.
	if i := strings.LastIndex(rest, "```"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

// Verdict is an Evaluate outcome.
type Verdict string

const (
	VerdictRegression  Verdict = "regression"
	VerdictImprovement Verdict = "improvement"
	// VerdictInsufficient is a third, real outcome: not enough evidence on
	// one or both sides of the amendment window to compare rates.
	VerdictInsufficient Verdict = "insufficient_evidence"
)

// EvaluateResult is one Evaluate run's outcome.
type EvaluateResult struct {
	Verdict     Verdict `json:"verdict"`
	BeforeRate  int     `json:"before_rate"`
	AfterRate   int     `json:"after_rate"`
	BeforeRuns  int     `json:"before_runs"`
	AfterRuns   int     `json:"after_runs"`
	CommitSHA   string  `json:"commit_sha,omitempty"`
	AppliedPath string  `json:"applied_path,omitempty"`
}

// minEvaluateRuns is the floor for a side of the window: with fewer than
// this many logged runs, a rate is noise, not a signal.
const minEvaluateRuns = 2

// Evaluate compares the skill's outcome rate before vs after the amendment
// window (amendedAt, RFC3339 — the timestamp of the .amended file's
// creation, or an explicit marker the caller supplies). Both sides must have
// at least minEvaluateRuns rows, else the verdict is insufficient_evidence
// (the .amended file is left in place for a later, better-armed run).
//
// Regression → the .amended file is discarded; the original script was never
// modified by Amend, so "original restored" is the invariant it already
// holds. Improvement → the .amended content replaces the original (the
// previous original is backed up to <code_path>.orig, never overwritten)
// and the change is committed to the workspace git repo so the improvement
// is auditable and revertable from git alone.
func (s *Store) Evaluate(ctx context.Context, name, amendedAt string) (EvaluateResult, error) {
	if s == nil || s.db == nil {
		return EvaluateResult{}, fmt.Errorf("skills store not initialized")
	}
	var id int64
	var codePath string
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, code_path FROM skills WHERE name = ?`, name).Scan(&id, &codePath); err != nil {
		if err == sql.ErrNoRows {
			return EvaluateResult{}, fmt.Errorf("skill not found: %q", name)
		}
		return EvaluateResult{}, fmt.Errorf("evaluate lookup: %w", err)
	}
	amendedPath := codePath + ".amended"
	if _, err := os.Stat(amendedPath); err != nil {
		return EvaluateResult{}, fmt.Errorf("no pending amendment for %q: %s missing", name, amendedPath)
	}

	res, err := s.evaluateRates(ctx, id, amendedAt)
	if err != nil {
		return EvaluateResult{}, err
	}
	if res.Verdict == VerdictInsufficient {
		return res, nil
	}

	switch res.Verdict {
	case VerdictRegression:
		if err := os.Remove(amendedPath); err != nil {
			return res, fmt.Errorf("discard .amended: %w", err)
		}
		return res, nil
	case VerdictImprovement:
		orig, err := os.ReadFile(codePath)
		if err != nil {
			return res, fmt.Errorf("read original skill: %w", err)
		}
		// Keep the previous original (never overwritten) so an applied
		// amendment is still revertable without git.
		if err := os.WriteFile(codePath+".orig", orig, 0o644); err != nil {
			return res, fmt.Errorf("backup original: %w", err)
		}
		amended, err := os.ReadFile(amendedPath)
		if err != nil {
			return res, fmt.Errorf("read amended skill: %w", err)
		}
		if err := os.WriteFile(codePath, amended, 0o644); err != nil {
			return res, fmt.Errorf("apply amendment: %w", err)
		}
		if err := os.Remove(amendedPath); err != nil {
			return res, fmt.Errorf("remove consumed .amended: %w", err)
		}
		sha, err := s.commitAmendment(ctx, name, codePath, res)
		if err != nil {
			// The file is already applied; the commit is the audit trail.
			// Surface it, do not roll the file back silently.
			return res, fmt.Errorf("amendment applied but git commit failed: %w", err)
		}
		res.CommitSHA = sha
		res.AppliedPath = codePath
		return res, nil
	}
	return EvaluateResult{}, fmt.Errorf("unreachable verdict %q", res.Verdict)
}

// evaluateRates computes the before/after success rates around amendedAt and
// maps them to a verdict. Pure read: no FS or git side effect. The skill id
// is passed in from Evaluate (which resolves it from name) — no re-resolution
// by code_path, which would be a second DB round-trip and would silently
// assume code_path is unique.
func (s *Store) evaluateRates(ctx context.Context, id int64, amendedAt string) (EvaluateResult, error) {
	rates := func(after bool) (rate, runs int) {
		op := " < "
		if after {
			op = " >= "
		}
		var success int
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(outcome,'success') = 'success' THEN 1 ELSE 0 END), 0)
			 FROM skill_usage WHERE skill_id = ? AND timestamp`+op+`?`, id, amendedAt).
			Scan(&runs, &success); err != nil {
			return 0, 0
		}
		if runs == 0 {
			return 0, 0
		}
		return (success*100 + runs/2) / runs, runs
	}
	before, beforeRuns := rates(false)
	after, afterRuns := rates(true)

	res := EvaluateResult{BeforeRate: before, AfterRate: after, BeforeRuns: beforeRuns, AfterRuns: afterRuns}
	switch {
	case beforeRuns < minEvaluateRuns || afterRuns < minEvaluateRuns:
		res.Verdict = VerdictInsufficient
	case after < before:
		res.Verdict = VerdictRegression
	case after > before:
		res.Verdict = VerdictImprovement
	default:
		// Equal rates: the amendment did not measurably change outcomes.
		res.Verdict = VerdictInsufficient
	}
	return res, nil
}

// commitAmendment commits the applied amendment to the workspace git repo
// (the rollback safety net for the cognee loop — the original content is the
// pre-amend state, recorded in the .orig backup AND in this commit). The
// commit is created with a local author/committer identity when the ambient
// git identity is unset, so a bare CI-style environment still commits.
// Returns the new commit's short SHA.
func (s *Store) commitAmendment(ctx context.Context, name, codePath string, res EvaluateResult) (string, error) {
	// The skills dir lives under the workspace root, which is where the
	// repo (if any) is rooted — no walk-up needed: scoping git to s.root
	// commits the workspace, not some outer checkout.
	dir := s.root

	// A loop-made commit must be attributable to the loop, not to whatever
	// ambient git identity happens to be set — so the loop's identity vars
	// are force-set (duplicates in the slice lose to the first key, and the
	// loop's key comes first), while the rest of the environment passes
	// through.
	env := append([]string{}, filterOutEnv(os.Environ(), gitIdentityKeys)...)
	env = append(env,
		"GIT_AUTHOR_NAME=skillgrid-skill-loop", "GIT_AUTHOR_EMAIL=skillgrid@localhost",
		"GIT_COMMITTER_NAME=skillgrid-skill-loop", "GIT_COMMITTER_EMAIL=skillgrid@localhost")

	rel, err := filepath.Rel(dir, codePath)
	if err != nil {
		rel = filepath.Base(codePath)
	}
	// The commit subject names the skill and the measured change; the body
	// carries the before/after rates verbatim ("before N% / after M%") so
	// the audit trail is grep-able from git alone.
	msg := fmt.Sprintf(
		"skill(%s): apply amendment — outcome rate %d%% -> %d%%\n\n"+
			"Skill improvement loop (Evaluate: improvement, %d runs before / %d after).\n"+
			"before %d%%, after %d%%. The pre-amendment content is preserved at %s.",
		name, res.BeforeRate, res.AfterRate, res.BeforeRuns, res.AfterRuns,
		res.BeforeRate, res.AfterRate, filepath.Base(codePath)+".orig")

	if _, err := runGit(ctx, dir, env, "add", "--", rel); err != nil {
		return "", err
	}
	if _, err := runGit(ctx, dir, env, "commit", "-m", msg); err != nil {
		return "", err
	}
	// The SHA is the artifact; the commit's stdout is not.
	return runGit(ctx, dir, env, "rev-parse", "--short", "HEAD")
}

var gitIdentityKeys = []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"}

// filterOutEnv drops env entries whose key is in keys (keeps the rest, order
// preserved).
func filterOutEnv(env, keys []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		drop := false
		for _, k := range keys {
			if len(e) > len(k)+1 && e[len(k)] == '=' && e[:len(k)] == k {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, e)
		}
	}
	return out
}

// runGit runs one git command in dir with the given env and returns its
// trimmed combined output. It fails on a non-zero exit.
func runGit(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("git %v: %w: %s", args, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
