package skills

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// seedUsage inserts skill_usage rows for one skill so the loop tests can drive
// Inspect/Evaluate with deterministic before/after outcome shapes without
// spawning sandbox processes. Timestamps are RFC3339 strings (monotonic order
// is preserved lexicographically within one year).
func seedUsage(t *testing.T, db *sql.DB, skillID int64, ts string, outcome, stderr string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO skill_usage (skill_id, session_id, timestamp, outcome, stderr) VALUES (?, NULL, ?, ?, ?)`,
		skillID, ts, outcome, stderr); err != nil {
		t.Fatalf("seed skill_usage: %v", err)
	}
}

// TestObserveLogsFailureWithStderr covers @happy path — skill failure is
// logged with outcome and stderr (C2 fix): a skill that exits 1 logs a
// skill_usage row with outcome='failure' and the captured stderr, and the
// result carries the row id (pre-C2, failures wrote no row).
func TestObserveLogsFailureWithStderr(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	id, err := s.Write(ctx, "connfail", "bash", "connects and fails",
		"echo 'connection timeout' 1>&2\nexit 1\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := s.Execute(ctx, "connfail", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", res.ExitCode)
	}
	if res.UsageID <= 0 {
		t.Fatalf("UsageID = %d, want > 0 (C2: failure rows are logged)", res.UsageID)
	}
	var outcome string
	var stderr sql.NullString
	if err := st.DB.QueryRow(
		`SELECT outcome, stderr FROM skill_usage WHERE id = ?`, res.UsageID).
		Scan(&outcome, &stderr); err != nil {
		t.Fatalf("read skill_usage outcome: %v", err)
	}
	if outcome != "failure" {
		t.Errorf("outcome = %q, want failure", outcome)
	}
	if !strings.Contains(stderr.String, "connection timeout") {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String, "connection timeout")
	}
	_ = id
}

// TestObserveLogsTimeoutAsError covers @failure — skill timeout is logged as
// error outcome (C2 fix): a skill past the sandbox deadline logs outcome='error'
// with the timeout marker captured in stderr.
func TestObserveLogsTimeoutAsError(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	if _, err := s.Write(ctx, "sleepy-loop", "bash", "sleeps too long", "sleep 15\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	s.Timeout = 2 * time.Second

	res, err := s.Execute(ctx, "sleepy-loop", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.TimedOut {
		t.Fatal("TimedOut = false, want true")
	}
	if res.UsageID <= 0 {
		t.Fatalf("UsageID = %d, want > 0 (C2: timeout rows are logged)", res.UsageID)
	}
	var outcome string
	var stderr sql.NullString
	if err := st.DB.QueryRow(
		`SELECT outcome, stderr FROM skill_usage WHERE id = ?`, res.UsageID).
		Scan(&outcome, &stderr); err != nil {
		t.Fatalf("read skill_usage outcome: %v", err)
	}
	if outcome != "error" {
		t.Errorf("outcome = %q, want error", outcome)
	}
	if !strings.Contains(stderr.String, "killed after") {
		t.Errorf("stderr = %q, want the timeout marker", stderr.String)
	}
}

// TestObserveLogsSuccessUnchanged covers @G3: the success leg keeps logging,
// now with an explicit outcome='success' (the pre-073 default value).
func TestObserveLogsSuccessUnchanged(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	if _, err := s.Write(ctx, "ok-loop", "bash", "works", "echo ok\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	res, err := s.Execute(ctx, "ok-loop", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.UsageID <= 0 {
		t.Fatalf("UsageID = %d, want > 0", res.UsageID)
	}
	var outcome string
	if err := st.DB.QueryRow(
		`SELECT outcome FROM skill_usage WHERE id = ?`, res.UsageID).
		Scan(&outcome); err != nil {
		t.Fatalf("read outcome: %v", err)
	}
	if outcome != "success" {
		t.Errorf("outcome = %q, want success", outcome)
	}
}

// TestInspectDetectsRecurringFailurePattern covers @happy path — inspect
// detects recurring failure pattern: 5 runs, 3 failing with the identical
// stderr → 40% success rate and that exact stderr as the top failure pattern.
func TestInspectDetectsRecurringFailurePattern(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	id, err := s.Write(ctx, "flaky", "bash", "fails sometimes", "echo flaky\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// 2 successes (t00-t01) then 3 failures with the identical stderr (t02-t04).
	seedUsage(t, st.DB, id, "2026-10-24T00:00:00Z", "success", "")
	seedUsage(t, st.DB, id, "2026-10-24T00:01:00Z", "success", "")
	seedUsage(t, st.DB, id, "2026-10-24T00:02:00Z", "failure", "connection timeout")
	seedUsage(t, st.DB, id, "2026-10-24T00:03:00Z", "failure", "connection timeout")
	seedUsage(t, st.DB, id, "2026-10-24T00:04:00Z", "failure", "connection timeout")

	ev, err := s.Inspect(ctx, "flaky")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if ev.TotalRuns != 5 {
		t.Errorf("TotalRuns = %d, want 5", ev.TotalRuns)
	}
	if ev.SuccessRuns != 2 || ev.FailureRuns != 3 {
		t.Errorf("SuccessRuns/FailureRuns = %d/%d, want 2/3", ev.SuccessRuns, ev.FailureRuns)
	}
	if ev.SuccessRate != 40 {
		t.Errorf("SuccessRate = %d, want 40 (percent, rounded)", ev.SuccessRate)
	}
	if len(ev.TopPatterns) != 1 {
		t.Fatalf("TopPatterns = %+v, want exactly 1 (one distinct stderr)", ev.TopPatterns)
	}
	if ev.TopPatterns[0].Stderr != "connection timeout" || ev.TopPatterns[0].Count != 3 {
		t.Errorf("top pattern = %+v, want {connection timeout 3}", ev.TopPatterns[0])
	}
}

// TestInspectTriangulatesMixedPatterns drives Inspect with a second,
// different input shape: 4 runs where two distinct stderr strings compete for
// the top slot — the most frequent one must win, and the second must be
// ranked below it with its own count.
func TestInspectTriangulatesMixedPatterns(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	id, err := s.Write(ctx, "mixed", "bash", "two failure kinds", "echo m\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	seedUsage(t, st.DB, id, "2026-10-24T00:00:00Z", "success", "")
	seedUsage(t, st.DB, id, "2026-10-24T00:01:00Z", "failure", "boom")
	seedUsage(t, st.DB, id, "2026-10-24T00:02:00Z", "failure", "boom")
	seedUsage(t, st.DB, id, "2026-10-24T00:03:00Z", "failure", "other error")

	ev, err := s.Inspect(ctx, "mixed")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if ev.TotalRuns != 4 || ev.SuccessRuns != 1 || ev.FailureRuns != 3 {
		t.Errorf("counts = %+v, want 4 total / 1 success / 3 failure", ev)
	}
	if ev.SuccessRate != 25 {
		t.Errorf("SuccessRate = %d, want 25", ev.SuccessRate)
	}
	if len(ev.TopPatterns) != 2 {
		t.Fatalf("TopPatterns = %+v, want 2 distinct stderrs", ev.TopPatterns)
	}
	top := ev.TopPatterns[0]
	if top.Stderr != "boom" || top.Count != 2 {
		t.Errorf("top pattern = %+v, want {boom 2}", top)
	}
	if ev.TopPatterns[1].Stderr != "other error" || ev.TopPatterns[1].Count != 1 {
		t.Errorf("second pattern = %+v, want {other error 1}", ev.TopPatterns[1])
	}
}

// TestInspectUnknownSkillErrors guards the not-found boundary of Inspect.
func TestInspectUnknownSkillErrors(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	if _, err := s.Inspect(context.Background(), "nope"); err == nil {
		t.Fatal("Inspect on a missing skill should fail")
	}
}

// TestAmendWritesNewFileWithoutModifyingOriginal covers @happy path — amend
// writes a new file without modifying the original: the LLM's proposed patch
// lands at <code_path>.amended and the original file is byte-identical.
func TestAmendWritesNewFileWithoutModifyingOriginal(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	const code = "echo broken\n"
	id, err := s.Write(ctx, "amendable", "bash", "gets amended", code, false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// "A skill with a recorded failure pattern": seed the evidence Amend
	// grounds the patch in.
	seedUsage(t, st.DB, id, "2026-10-24T00:00:00Z", "failure", "connection timeout")
	seedUsage(t, st.DB, id, "2026-10-24T00:01:00Z", "failure", "connection timeout")
	origPath := filepath.Join(root, ".skillgrid", "files", "skills", "amendable.sh")
	origBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	s.LLM = func(_ context.Context, system, user string) (string, error) {
		if !strings.Contains(user, "connection timeout") {
			t.Errorf("amend prompt should carry the failure evidence, got: %q", user)
		}
		return "echo fixed\n", nil
	}

	amendedPath, err := s.Amend(ctx, "amendable")
	if err != nil {
		t.Fatalf("Amend: %v", err)
	}
	if amendedPath != origPath+".amended" {
		t.Errorf("amended path = %q, want %q", amendedPath, origPath+".amended")
	}
	amendedBytes, err := os.ReadFile(amendedPath)
	if err != nil {
		t.Fatalf("read amended file: %v", err)
	}
	if string(amendedBytes) != "echo fixed\n" {
		t.Errorf("amended content = %q, want the LLM patch", string(amendedBytes))
	}
	afterBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("re-read original: %v", err)
	}
	if !bytes.Equal(origBytes, afterBytes) {
		t.Error("original skill file changed; Amend must not modify it in place")
	}
	_ = id
}

// TestAmendNoFailuresErrors covers the empty-evidence boundary: no logged
// failure to ground a patch in → a clear error, no .amended file.
func TestAmendNoFailuresErrors(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	if _, err := s.Write(ctx, "healthy", "bash", "no failures", "echo fine\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	s.LLM = func(context.Context, string, string) (string, error) { return "x", nil }

	if _, err := s.Amend(ctx, "healthy"); err == nil {
		t.Fatal("Amend with no failure evidence should fail")
	}
	if _, err := os.Stat(filepath.Join(root, ".skillgrid", "files", "skills", "healthy.sh.amended")); !os.IsNotExist(err) {
		t.Errorf("no .amended file should exist, stat err = %v", err)
	}
}

// TestAmendWithoutLLMErrors covers the LLM-missing boundary (auto_amend is off
// by default): Amend without an attached LLM fails rather than writing a
// guess.
func TestAmendWithoutLLMErrors(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	if _, err := s.Write(ctx, "nollm", "bash", "no llm", "echo x\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := s.Amend(ctx, "nollm"); err == nil {
		t.Fatal("Amend without an LLM attached should fail")
	}
}

// TestEvaluateRegressionRollsBack covers @edge — evaluate detects regression
// and rolls back: 80% before the amendment window, 50% after → the verdict is
// regression, the .amended file is discarded, and the original stays intact.
func TestEvaluateRegressionRollsBack(t *testing.T) {
	st, root := openTestStore(t)

	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	const code = "echo original\n"
	id, err := s.Write(ctx, "eval-regress", "bash", "regresses", code, false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	origPath := filepath.Join(root, ".skillgrid", "files", "skills", "eval-regress.sh")
	origBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	amendedPath := origPath + ".amended"
	if err := os.WriteFile(amendedPath, []byte("echo amended\n"), 0o644); err != nil {
		t.Fatalf("write .amended: %v", err)
	}

	// Before T: 4/5 success (80%). After T: 1/2 success (50%).
	for i, outcome := range []string{"success", "success", "success", "success", "failure"} {
		seedUsage(t, st.DB, id, "2026-10-24T00:0"+itoa(i)+":00Z", outcome, "x")
	}
	seedUsage(t, st.DB, id, "2026-10-24T01:00:00Z", "success", "")
	seedUsage(t, st.DB, id, "2026-10-24T01:01:00Z", "failure", "x")

	res, err := s.Evaluate(ctx, "eval-regress", "2026-10-24T00:30:00Z")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Verdict != VerdictRegression {
		t.Errorf("Verdict = %q, want regression", res.Verdict)
	}
	if res.BeforeRate != 80 {
		t.Errorf("BeforeRate = %d, want 80", res.BeforeRate)
	}
	if res.AfterRate != 50 {
		t.Errorf("AfterRate = %d, want 50", res.AfterRate)
	}
	if _, err := os.Stat(amendedPath); !os.IsNotExist(err) {
		t.Errorf(".amended should be discarded after a regression, stat err = %v", err)
	}
	afterBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("re-read original: %v", err)
	}
	if !bytes.Equal(origBytes, afterBytes) {
		t.Error("original skill file changed after a regression rollback")
	}
}

// TestEvaluateImprovementAppliesAndCommits covers @edge — evaluate detects
// improvement and applies: 50% before, 80% after → the verdict is
// improvement, the .amended file replaces the original, and a git commit is
// created. The test workspace is a git repo so the commit is observable.
func TestEvaluateImprovementAppliesAndCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	// git init the workspace: the Evaluate commit lands in a local repo.
	testRunGit(t, root, "init", "-q", "-b", "main")
	testRunGit(t, root, "add", ".")
	testRunGit(t, root, "commit", "-q", "-m", "baseline", "--allow-empty")
	testRunGit(t, root, "add", ".")
	testRunGit(t, root, "commit", "-q", "-m", "seed")

	const code = "echo original\n"
	id, err := s.Write(ctx, "eval-improve", "bash", "improves", code, false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	origPath := filepath.Join(root, ".skillgrid", "files", "skills", "eval-improve.sh")
	origBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	amendedPath := origPath + ".amended"
	if err := os.WriteFile(amendedPath, []byte("echo amended\n"), 0o644); err != nil {
		t.Fatalf("write .amended: %v", err)
	}

	// Before T: 1/2 success (50%). After T: 4/5 success (80%).
	seedUsage(t, st.DB, id, "2026-10-24T00:00:00Z", "success", "")
	seedUsage(t, st.DB, id, "2026-10-24T00:01:00Z", "failure", "x")
	for i, outcome := range []string{"success", "success", "success", "success", "failure"} {
		seedUsage(t, st.DB, id, "2026-10-24T01:0"+itoa(i)+":00Z", outcome, "x")
	}

	res, err := s.Evaluate(ctx, "eval-improve", "2026-10-24T00:30:00Z")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Verdict != VerdictImprovement {
		t.Fatalf("Verdict = %q, want improvement", res.Verdict)
	}
	if res.BeforeRate != 50 || res.AfterRate != 80 {
		t.Errorf("BeforeRate/AfterRate = %d/%d, want 50/80", res.BeforeRate, res.AfterRate)
	}
	// The .amended content now sits at the original path; the .amended file is
	// gone (consumed, not copied-and-kept).
	afterBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("re-read original: %v", err)
	}
	if !bytes.Equal(afterBytes, []byte("echo amended\n")) {
		t.Errorf("original = %q, want the amended content applied", string(afterBytes))
	}
	if _, err := os.Stat(amendedPath); !os.IsNotExist(err) {
		t.Errorf(".amended should be consumed on apply, stat err = %v", err)
	}
	if _, err := os.Stat(origPath + ".orig"); err != nil {
		t.Errorf("the discarded pre-apply backup should exist, stat err = %v", err)
	}
	if !bytes.Equal(origBytes, []byte("echo original\n")) {
		t.Errorf("baseline original = %q, sanity", string(origBytes))
	}

	// A git commit naming the skill must exist now.
	subj := testRunGit(t, root, "log", "-1", "--format=%s")
	if !strings.Contains(subj, "eval-improve") {
		t.Errorf("latest commit subject = %q, want it to name the skill", subj)
	}
	msg := testRunGit(t, root, "log", "-1", "--format=%B")
	for _, want := range []string{"before 50%", "after 80%"} {
		if !strings.Contains(msg, want) {
			t.Errorf("commit body missing %q: %s", want, msg)
		}
	}
}

// TestEvaluateNoAmendedFileErrors guards the missing-.amended boundary.
func TestEvaluateNoAmendedFileErrors(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root, "skilltest")
	ctx := context.Background()

	if _, err := s.Write(ctx, "no-amend", "bash", "nothing amended", "echo x\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := s.Evaluate(ctx, "no-amend", "2026-10-24T00:30:00Z"); err == nil {
		t.Fatal("Evaluate without a .amended file should fail")
	}
}

// TestStripCodeFenceCoversPlainFencedAndPlain guards the fence-stripping
// floor: a bare fenced answer loses its fence (and optional language tag),
// and an unfenced answer passes through byte-identical.
func TestStripCodeFenceCoversPlainFencedAndPlain(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"```bash\necho fixed\n```", "echo fixed"},
		{"```\necho fixed\n```", "echo fixed"},
		{"```bash\necho a\necho b\n```", "echo a\necho b"},
		{"echo plain", "echo plain"},
	}
	for i, c := range cases {
		if got := stripCodeFence(c.in); got != c.want {
			t.Errorf("case %d: stripCodeFence(%q) = %q, want %q", i, c.in, got, c.want)
		}
	}
}

// itoa is a tiny int→string for the seeded timestamps (strconv would be fine
// too; this keeps the test file's imports lean).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// runGit runs a git command in dir and returns its trimmed stdout. It fails
// the test on a non-zero exit.
func testRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
