package proxy

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
)

// compressPath and statsPath are the two T06 routes that short-circuit before
// the forward path (proxy.go ServeHTTP). /v1/compress is loopback-only and
// performs the compression transform with NO upstream call; /stats reports live
// metrics plus the current mode.
const (
	compressPath = "/v1/compress"
	statsPath    = "/stats"
)

// serveCompress handles POST /v1/compress: it compresses the message list in
// the request body using the SAME transform pipeline as the forward path
// (applyTransform, savings.go) but returns the compressed list directly — it
// never dials the upstream and never emits savings headers (there is no
// before/after against an upstream response).
//
// Loopback-only: a non-loopback origin gets 404 Not Found (not 403), so the
// route is invisible to a scanner — it cannot distinguish /v1/compress from a
// missing path (plan 11.18/R67). The 404 is method-agnostic: any method from a
// non-loopback origin returns 404.
//
// ADR-0016 fail-open: a transform fault (e.g. messages is a JSON object, not an
// array) degrades to the untransformed input with 200 — the caller still
// completes, never a 500.
func (p *Proxy) serveCompress(w http.ResponseWriter, r *http.Request) {
	// Loopback origin check FIRST: a non-loopback client never sees the route
	// (404, not 403), regardless of method.
	if !isLoopbackRequest(r) {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		// A read fault is a client problem, not a transform fault: 400.
		http.Error(w, "compress: reading request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	// Run the same transform the forward path uses. A nil/empty result (no
	// messages, or passthrough) or a transform fault both fail open to the
	// original body (ADR-0016) — the caller still completes.
	mode := p.GetMode()
	out := body
	if mode != ModePassthrough {
		if b, _, terr := applyTransform(mode, body); terr == nil && b != nil {
			out = b
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

// isLoopbackRequest reports whether the request's origin is a loopback address.
// It mirrors the T05 hardened isLoopbackBind check: parse the host from
// RemoteAddr (or the X-Forwarded-For first hop, absent a reverse proxy in the
// loopback-only design) and ask net.IP.IsLoopback(). A missing/unparseable
// host is treated as non-loopback (fail closed for a loopback-only route).
func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr without a port (e.g. just an IP): try parsing as-is.
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// serveStats handles GET /stats: it returns live session metrics as JSON and
// MUST include the current mode ("mode": "<mode>"). The key T06 requirement is
// that the current mode is exposed; live counters are reused from the store
// rollup when available (nil store → zeros, ADR-0016 fail-open). It is served
// on the same loopback listener as the proxy.
func (p *Proxy) serveStats(w http.ResponseWriter, r *http.Request) {
	type stats struct {
		Mode               string `json:"mode"`
		Requests           int64  `json:"requests"`
		TokensBefore       int64  `json:"tokens_before"`
		TokensAfter        int64  `json:"tokens_after"`
		CacheHits          int64  `json:"cache_hits"`
		CacheInvalidations int64  `json:"cache_invalidations"`
		Sessions           int    `json:"sessions"`
	}

	var s stats
	s.Mode = p.GetMode().String()

	// Live counters: sum across all proxied sessions (fail-open: nil store or
	// a missing table yields zeros, never a 500).
	if p.store != nil {
		sessions, err := p.store.ListProxySessions()
		if err == nil {
			s.Sessions = len(sessions)
			for _, sess := range sessions {
				s.Requests += sess.Requests
				s.TokensBefore += sess.TokensBefore
				s.TokensAfter += sess.TokensAfter
				s.CacheHits += sess.CacheHits
				s.CacheInvalidations += sess.CacheInvalidations
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s)
}
