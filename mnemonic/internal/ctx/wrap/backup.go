package wrap

import (
	"fmt"
	"os"
	"path/filepath"
)

// backupPath derives the predictable backup location for an agent:
// <config file>.ctx-backup when the agent has a config file, otherwise
// <backup dir>/<agent>.env.bak for env-only agents.
func backupPath(agent Agent, configPath string, backupDir string) string {
	if configPath != "" {
		return configPath + ".ctx-backup"
	}
	if backupDir == "" {
		backupDir = defaultBackupDir
	}
	return filepath.Join(backupDir, agent.Name+".env.bak")
}

// wrapStatePath is the persisted wrap-state marker for an agent:
// <state dir>/ctx/wrap-state/<agent>.wrapped. It is the durable idempotency
// signal across separate `ctx wrap <agent>` CLI invocations: because env
// injection is written only into the launched subprocess (not the parent
// shell), the marker is what tells a later invocation "this agent was already
// wrapped here" so it skips inject + backup without re-reading a non-persistent
// parent env.
func wrapStatePath(stateDir, agentName string) string {
	base := stateDir
	if base == "" {
		base = "."
	}
	return filepath.Join(base, ".skillgrid", "ctx", "wrap-state", agentName+".wrapped")
}

// recordWrapState writes the persisted wrap-state marker. It runs on the
// first wrap, after the backup, so a later invocation is idempotent even when
// the parent env was not itself rewritten. Fail-open (ADR-0016): the error is
// returned for the caller to surface, not swallowed.
func recordWrapState(stateDir, agentName string) error {
	path := wrapStatePath(stateDir, agentName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("wrap: creating wrap-state dir: %w", err)
	}
	return os.WriteFile(path, []byte("wrapped "+agentName+"\n"), 0o644)
}

// hasWrapState reports whether a prior wrap left a persisted wrap-state marker
// for this agent in stateDir.
func hasWrapState(stateDir, agentName string) bool {
	if stateDir == "" {
		return false
	}
	_, err := os.Stat(wrapStatePath(stateDir, agentName))
	return err == nil
}

// defaultBackupDir is the fallback backup location for env-only agents
// (the launcher's own state dir, not the agent's). ponytail: a single
// directory per repo root is enough for the core-4; per-machine scoping
// is an upgrade path if backups ever collide across projects.
var defaultBackupDir = filepath.Join(".skillgrid", "ctx", "wrap-backups")

// WriteBackup writes the pre-inject state (the config file's current bytes,
// or a snapshot of the current base-URL env for env-only agents) to
// backupPath. It runs BEFORE the injection takes effect. Idempotent: if a
// backup already exists it is left untouched (first wrap only) and
// (false, nil) is returned so the caller skips re-injection.
//
// Returns (false, err) on I/O failure; the caller treats that as
// fail-open-abort (ADR-0016): no backup, no write, clear error.
func WriteBackup(agent Agent, configPath, backupDir string, env func(string) string) (bool, error) {
	path := backupPath(agent, configPath, backupDir)

	if _, err := os.Stat(path); err == nil {
		// Backup exists: already wrapped (or a previous wrap wrote it).
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("wrap: creating backup dir: %w", err)
	}

	var content []byte
	if configPath != "" {
		// Config-file agents: capture the prior file bytes. A missing file
		// is the pre-wrap state — the backup records that explicitly so a
		// restore can recreate (not corrupt) the original.
		data, err := os.ReadFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				content = []byte("<no prior config file>\n")
			} else {
				return false, fmt.Errorf("wrap: reading prior config %s: %w", configPath, err)
			}
		} else {
			content = data
		}
	} else {
		// Env-only agents: snapshot the pre-inject base-URL value ("" when
		// unset) so a restore can put the original value back.
		content = []byte(agent.EnvVar + "=" + env(agent.EnvVar) + "\n")
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, fmt.Errorf("wrap: writing backup %s: %w", path, err)
	}
	return true, nil
}

// RestoreBackup puts the pre-wrap state back (inverse of the injection):
// config-file agents get their original bytes (or the file is removed when
// none existed); env-only agents get their original env value. It exists so
// `ctx wrap` is reversible; T14/ops surface the CLI verb.
func RestoreBackup(agent Agent, configPath, backupDir string) error {
	path := backupPath(agent, configPath, backupDir)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("wrap: reading backup %s: %w", path, err)
	}
	if configPath != "" {
		if string(content) == "<no prior config file>\n" {
			if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("wrap: removing %s: %w", configPath, err)
			}
			return nil
		}
		if err := os.WriteFile(configPath, content, 0o644); err != nil {
			return fmt.Errorf("wrap: restoring %s: %w", configPath, err)
		}
		return nil
	}
	// Env-only: the restore is a no-op for the process env (the subprocess
	// gets a fresh env per launch); nothing to undo in-process.
	return nil
}
