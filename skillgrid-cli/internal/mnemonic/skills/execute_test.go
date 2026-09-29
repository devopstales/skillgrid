package skills

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExecuteBashSkillCapturesStdoutAndUsage covers @step-03 (execute
// portion): a live bash skill runs in the sandbox, stdout/stderr are captured,
// the exit code is reported, and a skill_usage row is written.
func TestExecuteBashSkillCapturesStdoutAndUsage(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	id, err := s.Write(ctx, "hello-bash", "bash", "greets the operator", "echo hello\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := s.Execute(ctx, "hello-bash", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.Stderr != "" {
		t.Errorf("Stderr = %q, want empty", res.Stderr)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.TimedOut {
		t.Error("TimedOut = true, want false")
	}
	if res.UsageID <= 0 {
		t.Errorf("UsageID = %d, want > 0 (skill_usage row written on success)", res.UsageID)
	}

	var skillID int64
	var ts string
	if err := st.DB.QueryRow(
		`SELECT skill_id, timestamp FROM skill_usage WHERE id = ?`, res.UsageID).
		Scan(&skillID, &ts); err != nil {
		t.Fatalf("read skill_usage row: %v", err)
	}
	if skillID != id {
		t.Errorf("skill_usage.skill_id = %d, want %d", skillID, id)
	}
	if ts == "" {
		t.Error("skill_usage.timestamp should be set")
	}
}

// TestExecuteBashSkillStderrAndExitCode covers the non-zero exit boundary:
// stderr is captured and the exit code reported; no skill_usage row is written
// for a failed run.
func TestExecuteBashSkillStderrAndExitCode(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	if _, err := s.Write(ctx, "fail-bash", "bash", "always fails",
		"echo oops 1>&2\nexit 3\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := s.Execute(ctx, "fail-bash", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Stderr != "oops\n" {
		t.Errorf("Stderr = %q, want %q", res.Stderr, "oops\n")
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skill_usage`).Scan(&n); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if n != 0 {
		t.Errorf("skill_usage rows = %d, want 0 for a failed run", n)
	}
}

// TestExecutePythonSkill runs an interpreted python skill (skipped when
// python3 is absent).
func TestExecutePythonSkill(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	if _, err := s.Write(ctx, "hello-py", "python", "greets in python",
		"print('py-hello')\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := s.Execute(ctx, "hello-py", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Stdout != "py-hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "py-hello\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.UsageID <= 0 {
		t.Errorf("UsageID = %d, want > 0", res.UsageID)
	}
}

// TestExecuteGoSkill runs a compiled go skill (skipped when go is absent).
func TestExecuteGoSkill(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	if _, err := s.Write(ctx, "hello-go", "go", "greets in go",
		"package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"go-hello\")\n}\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := s.Execute(ctx, "hello-go", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Stdout != "go-hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "go-hello\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.UsageID <= 0 {
		t.Errorf("UsageID = %d, want > 0", res.UsageID)
	}
}

// TestExecuteUnknownLanguageRejectsWithoutExec covers @step-03 (failure:
// unknown language rejects without exec): a live skill whose language is not
// in the exec allowlist is rejected before any subprocess is spawned, and no
// skill_usage row is written. The row is seeded with a registry-known but
// non-executable language (markdown: written via Write, not in the exec
// allowlist) to reach the dispatch boundary.
func TestExecuteUnknownLanguageRejectsWithoutExec(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	if _, err := s.Write(ctx, "doc-skill", "markdown", "documentation only", "# doc\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	_, err := s.Execute(ctx, "doc-skill", "")
	if err == nil {
		t.Fatal("Execute with a non-executable language should fail before any subprocess")
	}
	if !strings.Contains(err.Error(), "cannot be executed") {
		t.Errorf("error should name the non-executable language: %v", err)
	}

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skill_usage`).Scan(&n); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if n != 0 {
		t.Errorf("skill_usage rows = %d, want 0 (no exec happened)", n)
	}
}

// TestExecuteSoftDeletedSkillErrors covers @step-03 (failure: soft-deleted
// skill → error): executing a soft-deleted skill returns a clear error naming
// it, with no exec and no skill_usage row.
func TestExecuteSoftDeletedSkillErrors(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	id, err := s.Write(ctx, "gone-skill", "bash", "gone", "echo gone\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`UPDATE skills SET deleted_at = ? WHERE id = ?`, now, id); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	_, err = s.Execute(ctx, "gone-skill", "")
	if err == nil {
		t.Fatal("Execute on a soft-deleted skill should fail")
	}
	if !errors.Is(err, ErrSoftDeleted) {
		t.Errorf("error should wrap ErrSoftDeleted, got: %v", err)
	}

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skill_usage`).Scan(&n); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if n != 0 {
		t.Errorf("skill_usage rows = %d, want 0", n)
	}
}

// TestExecuteUnknownSkillErrors guards the not-found boundary.
func TestExecuteUnknownSkillErrors(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	if _, err := s.Execute(context.Background(), "nope", ""); err == nil {
		t.Fatal("Execute on a missing skill should fail")
	}
}

// TestExecuteSandboxTimeout covers @step-03 (failure: timeout → no hang): a
// skill that sleeps past the sandbox deadline is killed and reported as a
// timeout (result.TimedOut), with a bounded wall time.
func TestExecuteSandboxTimeout(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	if _, err := s.Write(ctx, "sleepy", "bash", "sleeps too long", "sleep 15\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Shrink the deadline so the test itself stays fast; the production
	// default stays 30s (sandbox.go).
	s.Timeout = 2 * time.Second

	start := time.Now()
	res, err := s.Execute(ctx, "sleepy", "")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.TimedOut {
		t.Error("TimedOut = false, want true for a skill past the deadline")
	}
	if res.UsageID != 0 {
		t.Errorf("UsageID = %d, want 0 (no usage row for a timeout)", res.UsageID)
	}
	if elapsed > 10*time.Second {
		t.Errorf("Execute took %v, want bounded well under 10s", elapsed)
	}
}

// TestExecuteRejectsCodePathOutsideSkillDir covers @step-03 (security: path
// escape rejects without exec): a row whose code_path escapes
// .skillgrid/files/skills/ is rejected before any subprocess is spawned.
func TestExecuteRejectsCodePathOutsideSkillDir(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	id, err := s.Write(ctx, "escaper", "bash", "escaped path", "echo esc\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	evil := filepath.Join(root, "evil.sh")
	if err := os.WriteFile(evil, []byte("echo esc\n"), 0o644); err != nil {
		t.Fatalf("write evil.sh: %v", err)
	}
	if _, err := st.DB.Exec(`UPDATE skills SET code_path = ? WHERE id = ?`, evil, id); err != nil {
		t.Fatalf("rewrite code_path: %v", err)
	}

	_, err = s.Execute(ctx, "escaper", "")
	if err == nil {
		t.Fatal("Execute with an escaping code_path should fail before any subprocess")
	}

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skill_usage`).Scan(&n); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if n != 0 {
		t.Errorf("skill_usage rows = %d, want 0 (no exec happened)", n)
	}
}

// TestExecuteTruncatesOversizedOutput guards the 1MB output cap: a skill that
// emits more than 1MB of stdout returns truncated output with a note, no error.
// The output is generated at runtime (yes, bounded by the sandbox deadline) so
// the source script stays small.
func TestExecuteTruncatesOversizedOutput(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	s.Timeout = 5 * time.Second
	if _, err := s.Write(ctx, "bigout", "bash", "emits a lot", "yes a | head -c 10000000\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := s.Execute(ctx, "bigout", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(res.Stdout) > maxOutputBytes+512 {
		t.Errorf("Stdout len = %d, want bounded near %d", len(res.Stdout), maxOutputBytes)
	}
	if !strings.Contains(res.Stdout, "truncated") {
		t.Errorf("truncated stdout should carry a truncation note, tail = %q",
			res.Stdout[len(res.Stdout)-80:])
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}
