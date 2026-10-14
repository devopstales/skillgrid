// Package proxy is the loopback LLM proxy core for the Context Orchestrator
// (CTX v1.11, T04). It is a net/http + httputil.ReverseProxy bound to
// 127.0.0.1:8787 that forwards Anthropic- and OpenAI-compatible requests,
// applies a mode-selected context transform, computes an accurate
// X-CTX-Savings header, and fails open to passthrough on any compression
// fault. Provider errors are retried with exponential backoff and then
// returned verbatim.
//
// The transform (savings.go) and retry (retry.go) are pure and isolated so
// that T05 (streaming pass-through) and T06 (/v1/compress + runtime mode
// switch) can plug in without touching the transport. Mode is a plain field
// on *Proxy, switchable by a later ticket.
package proxy

import (
	"strings"
)

// Mode is the compression strategy selected for a proxied request. T04
// defines the type + startup flag + env read; T06 owns the runtime switch.
type Mode string

// The three compression modes (plan 16.2).
const (
	ModeToken       Mode = "token"
	ModeCache       Mode = "cache"
	ModePassthrough Mode = "passthrough"
)

// envModeVar is checked AFTER config resolution (per the Proxy config comment):
// a non-empty valid value overrides cfg.Proxy.Mode.
const envModeVar = "CTX_PROXY_MODE"

// valid reports whether m is one of the three modes.
func (m Mode) valid() bool {
	switch m {
	case ModeToken, ModeCache, ModePassthrough:
		return true
	}
	return false
}

// String returns the wire representation of the mode.
func (m Mode) String() string { return string(m) }

// parseMode parses and validates a raw mode string (case-insensitive).
func parseMode(s string) (Mode, bool) {
	m := Mode(strings.ToLower(strings.TrimSpace(s)))
	if !m.valid() {
		return "", false
	}
	return m, true
}

// resolveMode resolves the effective mode from, in priority order: an explicit
// override (the CLI --mode flag, validated upstream), the CTX_PROXY_MODE env
// var, then the config value. An empty/invalid override falls through to the
// next layer. A non-empty env value wins over config when valid.
func resolveMode(override, cfgMode, envValue string) Mode {
	if m, ok := parseMode(override); ok {
		return m
	}
	if envValue != "" {
		if m, ok := parseMode(envValue); ok {
			return m
		}
	}
	if m, ok := parseMode(cfgMode); ok {
		return m
	}
	return ModeCache
}

// ResolveMode is the mode-resolution entry point (exported for the CLI): it
// reads the CTX_PROXY_MODE env var via env and resolves the effective mode
// from, in priority order, an explicit override (the CLI --mode flag), the
// env var, then the config value. An empty/invalid layer falls through to the
// next.
func ResolveMode(override, cfgMode string, env func(string) string) Mode {
	return resolveMode(override, cfgMode, env(envModeVar))
}

// messages is a normalized, transport-agnostic view of the request's
// conversation: a role plus a string content blob. The proxy parses the
// upstream JSON body into this shape, applies the transform, and re-encodes it
// back to the upstream's wire format. Keeping the transform on this shape (not
// on raw JSON) makes it pure and provider-agnostic.
type messages struct {
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// headSplit returns the number of leading messages that form the "frozen head"
// for cache mode. The frozen head is the stable prefix: every message up to and
// including the last user message. Everything after that head is the mutable
// tail (assistant replies, tool results) that cache mode compresses.
//
// A simple stable-prefix detector (T04): the head boundary is the last user
// message index. If there is no user message, the head is empty (the whole
// conversation is mutable). T06/T11 refine the detector; this boundary is the
// seam they replace.
func headSplit(msgs []message) int {
	last := -1
	for i, m := range msgs {
		if strings.EqualFold(m.Role, "user") {
			last = i
		}
	}
	if last < 0 {
		return 0
	}
	return last + 1
}
