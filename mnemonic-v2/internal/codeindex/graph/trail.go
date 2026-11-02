package graph

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// RecordTrail records a retrieval trail for a walker read (M7/T15/INV-5).
// Every walker read (Impact/neighbors/Resolve/Path/Affected/Orient/Explain)
// records a trail via Store.RecordRetrieval into the shared retrieval_trails
// table. The route string becomes the `mnemonic trail show` vocabulary.
func RecordTrail(ctx context.Context, st store.Store, route, query string, degraded string) {
	t := store.RetrievalTrail{
		Route:     route,
		Query:     query,
		Project:   "p", // one DB per project (CI-O2)
		Degraded:  degraded != "",
		Timestamp: nowTimestamp(),
	}
	_ = st.RecordRetrieval(ctx, t)
}
