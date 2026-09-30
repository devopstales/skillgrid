package skills

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ErrSoftDeleted is the sentinel for executing a soft-deleted skill (the
// tool surfaces it verbatim, naming the skill).
var ErrSoftDeleted = errors.New("skill is soft-deleted and cannot be executed")

// DefaultTimeout is the sandbox execution deadline (TICKET-04 scope: 10s).
const DefaultTimeout = 10 * time.Second

// maxOutputBytes caps each captured stream (stdout/stderr) at 1MB; larger
// output is truncated with a note appended.
const maxOutputBytes = 1 << 20

// SkillResult is the outcome of one sandboxed skill execution. UsageID is
// zero when no skill_usage row was written (failure, timeout, or reject).
type SkillResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
	UsageID  int64
}

// execAllowlist maps a registry language to its runner command. A language
// outside this set is rejected before any subprocess is spawned (unknown
// language rejects without exec — step 03 boundary).
var execAllowlist = map[string][]string{
	"sh":         {"bash", "-c"},
	"bash":       {"bash", "-c"},
	"zsh":        {"bash", "-c"},
	"shell":      {"bash", "-c"},
	"python":     {"python3", "-c"},
	"node":       {"node", "-e"},
	"javascript": {"node", "-e"},
	"go":         nil, // compiled: written to a temp .go file, run with `go run`
}

// run executes a prepared command, capturing stdout/stderr and the exit code.
// A non-zero exit (exec.ExitError) is NOT a Go error: the skill ran and its
// status is reported via the result. exec.ErrNotFound (runner binary missing)
// maps to exit code 127, also reported rather than surfaced as an error.
// The deadline channel is the sandbox timeout: when it fires the process is
// killed and the wait drained, so a non-consuming script (e.g. `sleep 15`)
// can never outlive its deadline.
func run(cmd *exec.Cmd, stdin io.Reader, deadline <-chan struct{}) (stdout, stderr []byte, exitCode int, err error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = newSysProcAttr()
	}
	var out, errs bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errs
	if stdin != nil {
		cmd.Stdin = stdin
	} else {
		devNull, openErr := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
		if openErr != nil {
			return nil, nil, 125, openErr
		}
		defer devNull.Close()
		cmd.Stdin = devNull
	}
	runErr := cmd.Start()
	if runErr != nil {
		if errors.Is(runErr, exec.ErrNotFound) {
			return nil, nil, 127, nil
		}
		return nil, nil, 125, runErr
	}

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case runErr = <-waitErr:
		// Normal completion.
	case <-deadline:
		// cmd.Wait has not reaped the process, so its children are still its
		// children — kill the process group: a `sleep 15` under `bash -c`
		// otherwise survives the kill and holds the wait open.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		runErr = <-waitErr
	}
	if runErr != nil {
		switch {
		case errors.Is(runErr, exec.ErrNotFound):
			exitCode = 127
			runErr = nil
		default:
			// cmd.Wait returns *exec.ExitError directly (not wrapped) for a
			// non-zero exit: fold the status into the reported exit code.
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
				runErr = nil
			}
		}
	}
	return out.Bytes(), errs.Bytes(), exitCode, runErr
}

// isNoRows reports the not-found sentinel from a QueryRowContext scan.
func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// Execute is ExecuteWith with no session id: the skill runs and a
// skill_usage row is logged on success, but no skill_use session trail row
// is written (the CLI default when --session-id / SKILLGRID_SESSION_ID are
// unset).
func (s *Store) Execute(ctx context.Context, name, input string) (SkillResult, error) {
	return s.ExecuteWith(ctx, name, input, "")
}

// ExecuteWith runs a live Agent Skill in the sandbox: the code file is read
// from its registry code_path (which must stay under .skillgrid/files/skills/),
// the language-specific runner is dispatched with a per-call deadline, and
// stdout/stderr are captured with a 1MB cap. On a zero-exit run a skill_usage
// row is written (stamping session_id when one is supplied) and, when
// sessionID is non-empty, a skill_use session_events trail row is appended
// (mirroring the facts store, which logs its trail inside the store so both
// the MCP tool and the CLI write it). Rejections (unknown language, path
// escape, soft-deleted, missing skill) error before any subprocess is spawned.
func (s *Store) ExecuteWith(ctx context.Context, name, input, sessionID string) (SkillResult, error) {
	if s == nil || s.db == nil {
		return SkillResult{}, errors.New("skills store not initialized")
	}
	name = strings.TrimSpace(name)
	if !validName(name) {
		return SkillResult{}, fmt.Errorf("invalid skill name %q: use alnum, dash, underscore or dot — no path separators", name)
	}

	var id int64
	var language, codePath string
	var deletedAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, language, code_path, deleted_at FROM skills WHERE name = ?`, name).
		Scan(&id, &language, &codePath, &deletedAt)
	switch {
	case err == nil:
		if deletedAt.Valid {
			return SkillResult{}, fmt.Errorf("%w: %q", ErrSoftDeleted, name)
		}
	case isNoRows(err):
		return SkillResult{}, fmt.Errorf("skill not found: %q", name)
	default:
		return SkillResult{}, fmt.Errorf("lookup skill: %w", err)
	}

	// Path escape: the stored code_path must resolve to a file inside the
	// skills directory (defense in depth — Write already constrains names).
	skillDir := s.skillDir()
	cleanPath := filepath.Clean(codePath)
	if cleanPath != filepath.Join(skillDir, filepath.Base(cleanPath)) {
		return SkillResult{}, fmt.Errorf(
			"skill %q code path escapes the skills directory: %s", name, codePath)
	}
	if _, err := os.Stat(cleanPath); err != nil {
		return SkillResult{}, fmt.Errorf("read skill code: %w", err)
	}

	runner, ok := execAllowlist[language]
	if !ok {
		return SkillResult{}, fmt.Errorf(
			"skill %q language %q cannot be executed: executable languages: %s",
			name, language, executableLanguages())
	}

	deadline := s.Timeout
	if deadline <= 0 {
		deadline = DefaultTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	cmd, codeReader := buildSkillCommand(language, runner, cleanPath, input)
	cmd = exec.Command(cmd.Path, cmd.Args[1:]...)
	if language == "go" {
		cmd.Dir = filepath.Dir(cleanPath)
	}
	stdout, stderr, exitCode, runErr := run(cmd, codeReader, execCtx.Done())
	timedOut := execCtx.Err() == context.DeadlineExceeded

	result := SkillResult{
		Stdout:   truncateOutput(stdout),
		Stderr:   truncateOutput(stderr),
		ExitCode: exitCode,
		TimedOut: timedOut,
	}
	if timedOut {
		result.Stderr = result.Stderr + fmt.Sprintf("\n[skill sandbox: killed after %s]\n", deadline)
	} else if runErr != nil {
		// Startup failure outside the ExitError class (run() folds non-zero
		// exits into the reported exit code, so an error here means the
		// process never really ran).
		return result, fmt.Errorf("run skill %q: %w", name, runErr)
	}

	if exitCode == 0 && !timedOut {
		uid, err := s.insertUsage(ctx, id, sessionID)
		if err != nil {
			return result, fmt.Errorf("skill ran (exit 0) but logging skill_usage failed: %w", err)
		}
		result.UsageID = uid
		if sessionID != "" {
			// The trail row is logged here (not in the MCP layer) so the CLI
			// `skill execute --session-id` writes the same skill_use event the
			// MCP use_skill tool does — parity with the facts trail.
			if err := s.insertSkillUseEvent(ctx, sessionID, name, result); err != nil {
				return result, fmt.Errorf("skill ran (exit 0) but logging the skill_use trail failed: %w", err)
			}
		}
	}
	return result, nil
}

// buildSkillCommand returns the runner command for one skill. The go runner
// uses cmd.Args[0] as the program (`go run <file>`); the -c/-e runners take
// the code file content after the flag.
func buildSkillCommand(language string, runner []string, codePath, input string) (*exec.Cmd, io.Reader) {
	if language == "go" {
		args := []string{"run", codePath}
		if input != "" {
			args = append(args, input)
		}
		return exec.Command("go", args...), nil
	}
	// The code travels as the -c/-e argument of the interpreter (the registry
	// file is not executable, so it cannot be run as a script path — 126),
	// and the input travels as a shell/interpreter positional ($1 /
	// sys.argv[1] / process.argv[1]). The code is shell-escaped so a multi-
	// word or special-character program survives the -c boundary.
	var args []string
	switch language {
	case "python":
		pyCode, _ := os.ReadFile(codePath)
		args = append(args, "python3", "-c", string(pyCode))
	case "node", "javascript":
		nodeCode, _ := os.ReadFile(codePath)
		args = append(args, "node", "-e", string(nodeCode))
	default: // sh, bash, zsh, shell
		shCode, _ := os.ReadFile(codePath)
		args = append(args, "bash", "-c", strings.TrimRight(string(shCode), "\n"))
	}
	if input != "" {
		// The input lands in $1 for bash and sys.argv[1]/process.argv[1] for
		// python/node. For bash -c, the first argument after the program is
		// $0, so an explicit $0 placeholder (the interpreter name) is inserted
		// before the input to shift it into $1.
		args = append(args, args[0], input)
	}
	return exec.Command(args[0], args[1:]...), nil
}

// insertUsage writes the skill_usage row for a successful run and returns its
// id. sessionID is stamped on the row when non-empty (it was previously always
// NULL, which is what this plumb fixes).
func (s *Store) insertUsage(ctx context.Context, skillID int64, sessionID string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var sid any
	if sessionID != "" {
		sid = sessionID
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO skill_usage (skill_id, session_id, timestamp)
		VALUES (?, ?, ?)`, skillID, sid, now)
	if err != nil {
		return 0, fmt.Errorf("insert skill_usage: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("skill_usage id: %w", err)
	}
	return id, nil
}

// insertSkillUseEvent appends the skill_use trail row to the session event
// stream, mirroring the facts trail convention (next sequence in-session). It
// is the store-owned counterpart of what the MCP layer used to do, so the CLI
// and the MCP tool both write the same trail.
func (s *Store) insertSkillUseEvent(ctx context.Context, sessionID, name string, res SkillResult) error {
	now := time.Now().UTC().Format(time.RFC3339)
	payload, err := json.Marshal(map[string]any{
		"skill":     name,
		"exit_code": res.ExitCode,
		"timed_out": res.TimedOut,
		"usage_id":  res.UsageID,
	})
	if err != nil {
		return fmt.Errorf("marshal skill_use payload: %w", err)
	}
	var seq int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id = ?`,
		sessionID,
	).Scan(&seq); err != nil {
		return fmt.Errorf("next event sequence: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO session_events (session_id, project, sequence, action_type, result_status, tool_name, payload, timestamp)
		VALUES (?, ?, ?, 'skill_use', ?, 'use_skill', ?, ?)`,
		sessionID, s.project, seq, statusFor(res), string(payload), now,
	); err != nil {
		return fmt.Errorf("insert skill_use event: %w", err)
	}
	return nil
}

// statusFor maps a sandbox result to the session_events result_status
// column value (success on a zero-exit run, failure otherwise).
func statusFor(res SkillResult) string {
	if res.ExitCode != 0 || res.TimedOut {
		return "failure"
	}
	return "success"
}

// truncateOutput caps a captured stream at maxOutputBytes and appends a
// truncation note when it was cut.
func truncateOutput(b []byte) string {
	const note = "\n[skill sandbox: output truncated at 1MB]\n"
	if len(b) <= maxOutputBytes {
		return string(b)
	}
	return string(b[:maxOutputBytes]) + note
}

// executableLanguages renders the execAllowlist as a sorted, deduped list for
// the unknown-language error message.
func executableLanguages() string {
	set := map[string]struct{}{}
	for lang := range execAllowlist {
		set[lang] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for lang := range set {
		out = append(out, lang)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return strings.Join(out, ", ")
}
