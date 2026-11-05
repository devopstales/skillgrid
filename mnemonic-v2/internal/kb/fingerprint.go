package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

const fingerprintFormat = "kb-source-fingerprint-v1"

// SourceFingerprint is the identity of a run's source inputs (003 §3 layer 4):
// the project's live memory facts + the indexed code files + the git HEAD.
// The KB's own writes (index/**, the kb_* rows, engine-owned frontmatter) are
// structurally excluded, so the run's own writes can never perturb the
// fingerprint — the same stance as OpenWiki excluding openwiki/ from its
// source fingerprint.
func SourceFingerprint(ctx context.Context, st store.Store, project, repoRoot string) (string, error) {
	h := sha256.New()
	w := func(s string) { fmt.Fprintf(h, "%d\x00%s\x00", len(s), s) }
	w(fingerprintFormat)
	w(gitHead(repoRoot))
	w(project)

	facts, err := st.Query(ctx,
		`SELECT sync_id, normalized_hash, type, topic_key FROM observations
		 WHERE project = ? AND deleted_at IS NULL ORDER BY sync_id`, project)
	if err != nil {
		return "", fmt.Errorf("kb: fingerprint facts: %w", err)
	}
	for _, r := range facts {
		w("f" + s0(r, 0) + "\x00" + s0(r, 1) + "\x00" + s0(r, 2) + "\x00" + s0(r, 3))
	}
	code, err := st.Query(ctx, `SELECT path, content_hash FROM files ORDER BY path`)
	if err != nil {
		return "", fmt.Errorf("kb: fingerprint code: %w", err)
	}
	for _, r := range code {
		w("c" + s0(r, 0) + "\x00" + s0(r, 1))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// gitHead is the repository HEAD (the fingerprint's pairing anchor). A
// lookup failure is a signal, not a failure — the detector falls to "none".
var gitHead = func(root string) string {
	if root == "" {
		return "none"
	}
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return "none"
	}
	return string(out)
}

// s0 reads one row cell as a string (int64 → decimal, []byte → string).
func s0(r []any, i int) string {
	if i >= len(r) {
		return ""
	}
	switch v := r[i].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int64:
		return fmt.Sprint(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}
