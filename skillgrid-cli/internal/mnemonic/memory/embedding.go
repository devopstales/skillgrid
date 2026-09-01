package memory

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
)

// EmbeddingEnabled reports whether embedding recall is active for this
// process. It is opt-in (MNEMONIC_EMBED=1 / true) so the default path stays
// FTS5-only and has no embedder dependency.
func EmbeddingEnabled() bool {
	switch os.Getenv("MNEMONIC_EMBED") {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Vector is an in-memory float32 vector.
type Vector struct{ Data []float32 }

// EncodeVector serialises a vector to the little-endian float32 blob layout
// used by the `embedding` BLOB column.
func EncodeVector(v Vector) []byte {
	buf := make([]byte, len(v.Data)*4)
	for i, f := range v.Data {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// DecodeVector is the inverse of EncodeVector.
func DecodeVector(blob []byte) (Vector, error) {
	if len(blob)%4 != 0 {
		return Vector{}, fmt.Errorf("embedding length %d is not a multiple of 4", len(blob))
	}
	v := Vector{Data: make([]float32, len(blob)/4)}
	for i := range v.Data {
		v.Data[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return v, nil
}

// CosineSimilarity is the standard cosine distance in [-1, 1].
func CosineSimilarity(a, b Vector) float64 {
	if len(a.Data) != len(b.Data) || len(a.Data) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a.Data {
		d := float64(a.Data[i])
		e := float64(b.Data[i])
		dot += d * e
		na += d * d
		nb += e * e
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// ReciprocalRankFusion blends two ranked lists (FTS and vector) over the same
// item set using the RRF formula  score(x) = 1/(k+rankF(x)) + 1/(k+rankV(x)),
// with k=60 (the standard constant). Items ranked by only one list still
// appear with that single term. This is the standard way to combine lexical
// and semantic retrieval. FTS5 stays the floor: when the vector list is
// empty, the score reduces to 1/(k+rankF), which is monotonic in rankF, so
// the ordering is identical to FTS alone.
func ReciprocalRankFusion(ids []string, ftsRanks map[string]int, vecRanks map[string]int, k int) []string {
	if k <= 0 {
		k = 60
	}
	type scored struct {
		id    string
		score float64
	}
	out := make([]scored, 0, len(ids))
	for _, id := range ids {
		score := 0.0
		if r, ok := ftsRanks[id]; ok {
			score += 1.0 / float64(k+r+1)
		}
		if r, ok := vecRanks[id]; ok {
			score += 1.0 / float64(k+r+1)
		}
		if score > 0 {
			out = append(out, scored{id: id, score: score})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].id < out[j].id
	})
	ids2 := make([]string, 0, len(out))
	for _, o := range out {
		ids2 = append(ids2, o.id)
	}
	return ids2
}
