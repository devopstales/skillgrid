package memory

// Sensitive path matching + secret redaction for tool-call events (TICKET-02).
//
// A tool call that touches a sensitive path (env files, private keys, anything
// with "secret" in it, SSH/AWS config dirs) is flagged is_sensitive=1 and its
// content is NEVER stored raw: the event payload carries only a SHA-256 hash
// of the full content plus a masked preview (first 200 chars with KEY=value
// pairs masked). Absence-oracle property: the full secret value appears
// nowhere in the stored row.

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"strings"
)

// previewMaxChars caps the redacted preview at the first 200 chars of content.
const previewMaxChars = 200

// envValueRe matches an env-style assignment line (KEY=value) so the value
// half can be masked. Fail-closed on shape, not on key names: any line shaped
// like an assignment is masked, whether or not the key looks secret-y.
var envValueRe = regexp.MustCompile(`(?m)^(\s*[A-Za-z_][A-Za-z0-9_.\-]*\s*=).*$`)

// IsSensitivePath reports whether path touches secret-bearing material:
//   - a `.env` file (exactly, or `.env.<suffix>` variants)
//   - a `*.pem` / `*.key` private key file
//   - any path containing a `secret` segment
//   - any path under a `.ssh/` or `.aws/` directory (matched per segment, so
//     a file merely named `my.aws.backup` does not match)
//
// Matching is case-insensitive and slash-normalized so Windows-style
// separators behave the same.
func IsSensitivePath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(strings.TrimSpace(path)))
	if p == "" {
		return false
	}
	base := p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		base = p[i+1:]
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") {
		return true
	}
	if strings.Contains(p, "secret") {
		return true
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".ssh" || seg == ".aws" {
			return true
		}
	}
	return false
}

// redactPreview reduces raw content to its storable form: the hex SHA-256 of
// the FULL content (the positive control — proves what was seen without
// storing it) plus the first 200 chars with every `KEY=value` line's value
// replaced by `***`. The raw secret is in neither half.
func redactPreview(content string) (hash, preview string) {
	sum := sha256.Sum256([]byte(content))
	hash = hex.EncodeToString(sum[:])
	runes := []rune(content)
	if len(runes) > previewMaxChars {
		runes = runes[:previewMaxChars]
	}
	preview = envValueRe.ReplaceAllString(string(runes), `$1***`)
	return hash, preview
}
