// run_test_helpers_test.go — the CLI test run helpers: the tests drive
// the dispatch in-process (no subprocess — the store is real SQLite, the
// wiring under test is the dispatch). The helpers install a fixture
// (a real store in a temp data dir, a stub detector with no git
// shell-out) and call the same run(args) the binary's main() calls,
// capturing the exit code and the stdout the command produces.

package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// installRunEnv wires the fixture: the store seam opens a fresh SQLite
// store in a temp data dir on each call (the commands close the store
// after use, so a shared store would be closed after the first command),
// the handler seam builds the test handler over it (the stub detector
// never shells out), and the git binding is a no-op. It restores the
// seams on cleanup.
func installRunEnv(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()

	prevStoreFor, prevHandlerFor, prevCodeHandlerFor, prevGit := storeFor, handlerFor, codeHandlerFor, gitToplevel

	// Each call opens a new store (acquires a new pool ref). The pool
	// keeps the DB open until the last ref is released, so the commands
	// can close their store without affecting the others.
	storeFor = func(cfg config.Config, p string) (store.Store, error) {
		return store.Open(context.Background(), dataDir, p, store.SQLite, "")
	}
	handlerFor = func(cfg config.Config, s store.Store) (*handler.Handler, error) {
		return newTestHandlerWithStore(t, s), nil
	}
	codeHandlerFor = func(cfg config.Config, s store.Store) *handler.CodeHandler {
		return newTestCodeHandlerForStore(t, s)
	}
	gitToplevel = func(string) (string, error) { return "", errors.New("no git in tests") }

	t.Cleanup(func() {
		storeFor, handlerFor, codeHandlerFor, gitToplevel = prevStoreFor, prevHandlerFor, prevCodeHandlerFor, prevGit
	})
}

// runCLI runs the command in-process with the fixture and returns the
// captured stdout. A non-zero exit code fails the test (the command is
// expected to succeed).
func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	installRunEnv(t)
	out, code := captureRun(t, args...)
	if code != exitOK {
		t.Fatalf("runCLI(%v) exited %d, want 0 (output: %s)", args, code, out)
	}
	return out
}

// runCLIError runs the command in-process and returns the combined
// output as an error when the exit code is non-zero (nil on exit 0).
// The test asserts the error text names the offending flag/subcommand.
func runCLIError(t *testing.T, args ...string) error {
	t.Helper()
	installRunEnv(t)
	out, code := captureRun(t, args...)
	if code == exitOK {
		return nil
	}
	return &runError{code: code, msg: out}
}

// runCLIExit runs the command in-process and returns its exit code (the
// doctor's strict-mode contract: the code is the assertion).
func runCLIExit(t *testing.T, args ...string) int {
	t.Helper()
	installRunEnv(t)
	_, code := captureRun(t, args...)
	return code
}

// newTestCodeHandlerForStore builds a CodeHandler over a caller-provided
// store (the test fixture for the index command group — the stub detector
// never shells out, the embedder is absent (the degraded: no_vector floor)).
func newTestCodeHandlerForStore(t *testing.T, s store.Store) *handler.CodeHandler {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return handler.NewCodeHandler(s, nil, c, t.TempDir(), &codeDetStub{resolve: func(cwd, explicit string) (string, error) {
		if explicit != "" {
			return explicit, nil
		}
		return "p", nil
	}})
}

// codeDetStub is the stub detector the CodeHandler test fixture uses (no git
// shell-out — the test's project is fixed to "p").
type codeDetStub struct {
	resolve func(cwd, explicit string) (string, error)
}

func (s *codeDetStub) Resolve(cwd, explicit string) (string, error) { return s.resolve(cwd, explicit) }

// runError is the error a run helper returns for a non-zero exit (the
// msg is the combined output the command printed).
type runError struct {
	code int
	msg  string
}

func (e *runError) Error() string { return e.msg }

// captureRun runs run(args) with stdout/stderr captured and returns
// (combined output, exit code). The capture is through the stdoutPipe
// seam (a *bytes.Buffer the commands write to — outW() is an
// *os.File and cannot be swapped for a buffer; the seam is the one
// indirection the test needs).
func captureRun(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	oldOut, oldErr := stdoutPipe, stderrPipe
	stdoutPipe, stderrPipe = &stdout, &stderr
	code := run(args)
	stdoutPipe, stderrPipe = oldOut, oldErr
	return stdout.String() + stderr.String(), code
}
