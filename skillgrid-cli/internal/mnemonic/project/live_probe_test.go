package project

import (
	"os"
	"testing"
)

// TestResolveLiveLayout is a diagnostic that prints how the resolver now
// treats this machine's real directories. It never fails — it is here to give
// the operator a visible before/after record when running ./project tests.
func TestResolveLiveLayout(t *testing.T) {
	cases := []string{"/data/git/AI", "/data/git/AI/skillgrid", "/tmp"}
	orig, _ := os.Getwd()
	for _, dir := range cases {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		os.Chdir(dir)
		res, err := ResolveDetailed(".")
		var avail int
		_ = avail
		if err != nil {
			t.Logf("%-26s id=%-22s src=%-12s err=%v", dir, res.ID, res.Source, err)
		} else {
			t.Logf("%-26s id=%-22s src=%-12s avail=%d warn=%q", dir, res.ID, res.Source, len(res.Available), res.Warning)
		}
	}
	os.Chdir(orig)
}
