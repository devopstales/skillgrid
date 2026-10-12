package wrap

import "strings"

// configPathFor returns the agent's config-file path, or "" for env-only
// agents. ponytail: T07 injects via env for all agents (most testable, no
// per-agent file format to parse); config-file rewriting is an upgrade path
// for agents that do not honor base-URL env vars, and the backup/inject
// seam already supports it (backup.go captures prior file bytes when
// configPath != "").
func configPathFor(agent Agent) string {
	return ""
}

// InjectEnv returns a copy of env with the agent's base-URL var pointed at
// the proxy. It never mutates the caller's slice.
func InjectEnv(env []string, agent Agent, proxyBaseURL string) []string {
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, kv := range env {
		if strings.HasPrefix(kv, agent.EnvVar+"=") {
			out = append(out, agent.EnvVar+"="+proxyBaseURL)
			replaced = true
			continue
		}
		out = append(out, kv)
	}
	if !replaced {
		out = append(out, agent.EnvVar+"="+proxyBaseURL)
	}
	return out
}

// alreadyWrapped reports whether env already targets the CTX loopback proxy
// (a 127.0.0.1/::1 base URL in the agent's base-URL var) — the idempotency
// check. A non-loopback base URL (e.g. a real provider endpoint) is NOT
// "wrapped": the first wrap rewrites it and takes a backup first.
func alreadyWrapped(env []string, agent Agent) bool {
	for _, kv := range env {
		if !strings.HasPrefix(kv, agent.EnvVar+"=") {
			continue
		}
		val := kv[len(agent.EnvVar)+1:]
		return isLoopbackBaseURL(val)
	}
	return false
}

// isLoopbackBaseURL reports whether base looks like the CTX loopback proxy
// (a 127.0.0.1 / ::1 / localhost host, with or without a scheme or path).
// It is a heuristic marker, not a full URL grammar: the launcher only ever
// writes loopback URLs, so anything else is "not wrapped by us".
func isLoopbackBaseURL(base string) bool {
	rest := base
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	host := rest
	// An authority without a path can contain no '/' (it would be the
	// path separator), so use the first '/' as an "authority ends here"
	// marker and keep everything after it (a leading path, which we
	// discard).
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		host = rest[:i]
	}
	switch {
	case strings.HasPrefix(host, "["):
		// Bracketed IPv6 host ([::1]:8787): keep the bracketed address.
		if i := strings.IndexByte(host, ']'); i >= 0 {
			host = host[:i+1]
		}
	case strings.Contains(host, ":"):
		// host:port (IPv4/hostname form): drop the port.
		host = host[:strings.LastIndexByte(host, ':')]
	}
	host = strings.Trim(host, "[]")
	switch host {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}
