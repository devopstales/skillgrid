package syncid

import (
	"regexp"
	"testing"
)

func TestNewSyncIDFormat(t *testing.T) {
	for _, prefix := range []string{"obs", "rel", "ses", "prm"} {
		id := NewSyncID(prefix)
		if !regexp.MustCompile(`^` + prefix + `-[0-9a-f]{16}$`).MatchString(id) {
			t.Fatalf("bad sync id %q", id)
		}
		if id == NewSyncID(prefix) {
			t.Fatalf("collision in same call pair")
		}
	}
}
