package process

import (
	"fmt"
	"hash"
	"hash/fnv"
)

// fnvNew returns a 64-bit FNV-1a hasher (pure Go, CGo-free).
func fnvNew() hash.Hash64 { return fnv.New64a() }

// fnvHex returns the hex digest of a single FNV-1a hash of s.
func fnvHex(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum64())
}

// fnvHexSum returns the hex digest of an in-progress FNV-1a hash.
func fnvHexSum(h hash.Hash64) string { return fmt.Sprintf("%x", h.Sum64()) }
