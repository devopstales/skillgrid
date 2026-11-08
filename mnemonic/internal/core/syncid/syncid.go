package syncid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewSyncID mints a cross-machine identity: prefix + "-" + 16 hex chars
// (8 crypto/rand bytes). Prefix ∈ obs|rel|ses|prm (INV-6).
func NewSyncID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// ponytail: rand.Read failure is unrecoverable on sane platforms;
		// timestamp fallback keeps the format stable.
		return fmt.Sprintf("%s-%016x", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b)
}
