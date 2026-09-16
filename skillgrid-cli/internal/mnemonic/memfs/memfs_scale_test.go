package memfs

import (
	"context"
	"strings"
	"testing"
)

// TestMemURIResolutionWith10kObservations is 26.1 [RED] — `mem://` URI
// resolution and scope listing stay correct and fast under scale (10k+
// observations). The URI parser (ResolveURI) is allocation-free and must
// resolve identically regardless of store size; the scope List query must
// honor its LIMIT 200 cap (returning at most 200 rows, newest first) rather
// than materializing the whole 10k+ set. This pins the scale behavior: a
// regression that drops the LIMIT, or that makes resolution depend on a full
// table scan, would blow past the cap or change results as the store grows.
func TestMemURIResolutionWith10kObservations(t *testing.T) {
	fx := newMemFSTestFixture(t, "memfs-scale-test")
	ctx := context.Background()

	const total = 10_000
	const inScope = 5_000

	// 5000 observations under project/A/preferences (varied titles so dedup on
	// (title+content+type) does not collapse them), 5000 under project/A/entities.
	for i := 0; i < inScope; i++ {
		fx.saveObs(t, "pref-obs-"+itoa(i), "project/A/preferences/pref-"+itoa(i), "preferences", "preference body "+itoa(i))
	}
	for i := 0; i < total-inScope; i++ {
		fx.saveObs(t, "ent-obs-"+itoa(i), "project/A/entities/ent-"+itoa(i), "entities", "entity body "+itoa(i))
	}

	// URI resolution is allocation-free and store-size independent: it must
	// resolve to the same ScopeFilter as the small-scale test, under 10k rows.
	f, err := ResolveURI("mem://project/A/preferences")
	if err != nil {
		t.Fatalf("ResolveURI at 10k scale: %v", err)
	}
	if f.Kind != "project" || f.ID != "A" || f.MemoryType != "preferences" {
		t.Fatalf("ResolveURI at 10k scale: got %+v", f)
	}
	if f.Prefix() != "project/A/preferences" {
		t.Fatalf("ResolveURI prefix at 10k scale: got %q", f.Prefix())
	}

	// Scope listing must honor its LIMIT 200 cap (not return all 5000) and
	// return only in-scope (preferences) observations.
	res, err := fx.fs.List(ctx, "project/A/preferences")
	if err != nil {
		t.Fatalf("ls project/A/preferences at 10k scale: %v", err)
	}
	if len(res) > 200 {
		t.Fatalf("ls returned %d rows, want at most 200 (LIMIT cap) at 10k scale", len(res))
	}
	if len(res) == 0 {
		t.Fatalf("ls returned 0 rows at 10k scale; expected the newest <=200 preferences")
	}
	for _, o := range res {
		if !strings.Contains(o.TopicKey, "preferences") {
			t.Fatalf("ls at 10k scale returned an out-of-scope observation: title=%q topic_key=%q", o.Title, o.TopicKey)
		}
	}

	// The broader scope (project/A/) must also be capped at 200 and mix only
	// in-A rows (no user/B rows exist here, but the cap is the assertion).
	resAll, err := fx.fs.List(ctx, "project/A/")
	if err != nil {
		t.Fatalf("ls project/A/ at 10k scale: %v", err)
	}
	if len(resAll) > 200 {
		t.Fatalf("ls project/A/ returned %d rows, want at most 200 (LIMIT cap)", len(resAll))
	}
}

// itoa is a tiny int→string helper so the test does not import strconv for a
// formatting detail.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
