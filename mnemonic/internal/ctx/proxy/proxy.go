package proxy

import (
	"bytes"
	"crypto/md5" // #nosec G401 — used only to derive a stable session id, not a security boundary
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// Proxy is the loopback LLM proxy. It owns the resolved config, a fail-open
// ctx store accessor for telemetry, a logger, and the effective mode. The mode
// is guarded by mu: it is read on the hot forward path (RLock) and written by
// SetMode (Lock), so a runtime switch (T06) is safe against concurrent
// forwarding. The lock is held only for the field read/write, never across I/O.
type Proxy struct {
	cfg      *config.CtxConfig
	store    *store.Store
	log      *slog.Logger
	sleep    Sleep
	upstream baseUpstream

	// mu guards mode and lastChangeID.
	mu               sync.RWMutex
	mode             Mode
	lastChangeID     string
	SetModeEventSink func(old, new Mode, reason string)
}

// New builds a Proxy from resolved config, the shared ctx store accessor (may
// be nil — telemetry then no-ops, ADR-0016), and a logger. mode is the
// effective mode already resolved (CLI --mode > CTX_PROXY_MODE > config). The
// provider call uses the real clock/sleeper; use NewWithSleep for tests.
func New(cfg *config.CtxConfig, cs *store.Store, log *slog.Logger, mode Mode) *Proxy {
	return &Proxy{
		cfg:      cfg,
		store:    cs,
		log:      log,
		mode:     mode,
		sleep:    realSleep,
		upstream: detectUpstream(http.DefaultClient, os.Getenv),
	}
}

// NewWithSleep builds a Proxy with an injectable Sleep (test seam) and,
// optionally, a custom http.Client for the provider call.
func NewWithSleep(cfg *config.CtxConfig, cs *store.Store, log *slog.Logger, mode Mode, sleep Sleep, client *http.Client) *Proxy {
	p := New(cfg, cs, log, mode)
	p.sleep = sleep
	p.upstream.client = client
	return p
}

// Handler returns the http.Handler for the proxy. It is safe to serve on any
// listener (loopback in production; an httptest server in tests).
func (p *Proxy) Handler() http.Handler {
	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// T06 routes short-circuit BEFORE the forward path: /v1/compress is a
	// loopback-only local transform (no upstream), and /stats reports live
	// metrics + the current mode. Both are served on the same loopback
	// listener, so they share the proxy's trust boundary.
	if r.URL.Path == compressPath {
		p.serveCompress(w, r)
		return
	}
	if r.URL.Path == statsPath {
		p.serveStats(w, r)
		return
	}

	// WebSocket upgrade requests bypass the savings transform entirely: WS
	// frames are not JSON chat requests, so they pass through verbatim.
	// The proxy hijacks the connection and pipes both directions (T05).
	if isWebSocketUpgrade(r) {
		_ = p.forwardWebSocket(w, r)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "proxy: reading request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	upstream := p.upstream

	// Determine the effective mode for this request. T04: the startup-resolved
	// p.mode is authoritative (T06 adds per-request override). The mode used
	// for the transform is what the savings header reports. It is read under
	// the RWMutex so a concurrent SetMode (runtime switch) is safe.
	mode := p.GetMode()

	// Token count BEFORE the transform: count on the exact wire bytes the
	// client sent. This is the deterministic "before" for the savings delta.
	tokensBefore := EstimateTokens(body)

	// Parse + transform. A transform fault fails open to passthrough
	// (ADR-0016): forward the ORIGINAL body verbatim, report the fallback
	// mode, never 500 the client because compression failed.
	forwardBody := body
	effectiveMode := mode
	cacheHit := false
	if mode != ModePassthrough {
		if b, hit, terr := applyTransform(mode, body); terr == nil {
			if b != nil {
				forwardBody = b
				cacheHit = hit
			}
		} else {
			forwardBody = body
			effectiveMode = ModePassthrough
		}
	}
	tokensAfter := EstimateTokens(forwardBody)

	sv := savings{
		Before:   tokensBefore,
		After:    tokensAfter,
		Mode:     effectiveMode,
		CacheHit: cacheHit,
	}
	if p.cfg.Proxy.SavingsHeaders {
		sv.writeHeaders(w)
	}

	// Forward to the upstream with the correct API key, retrying on a non-2xx
	// or transport error. The response body is streamed directly into w.
	p.forward(w, r, forwardBody)

	// Emit telemetry (fail-open: a missing ctx table is a no-op write).
	p.recordTelemetry(r, upstream.Name, string(effectiveMode), sv)
}

// forward builds and issues the upstream request (with the configured API key
// injected, the client's Authorization/x-api-key replaced), and streams the
// upstream response straight into w after the retry budget. The upstream
// status/content-type is returned for the caller to inspect, and the response
// body has already been copied to w by the time this returns.
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, body []byte) (int, string) {
	u := p.upstream
	ctx := r.Context()

	baseURL := u.BaseURL
	path := r.URL.Path
	if baseURL == "" {
		// No upstream configured: fail open — return a 502 with a clear body
		// (the client still completes, no panic).
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: no upstream configured for " + u.Name))
		return http.StatusBadGateway, "text/plain"
	}

	fullURL := baseURL
	if !strings.HasSuffix(fullURL, "/") {
		fullURL += "/"
	}
	fullURL += strings.TrimLeft(path, "/")
	pu, err := url.Parse(fullURL)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: bad upstream URL: " + err.Error()))
		return http.StatusBadGateway, "text/plain"
	}

	do := func() (*http.Response, error) {
		out, err := http.NewRequestWithContext(ctx, r.Method, pu.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		// Copy the client's headers, then inject/replace the auth header.
		for k, vs := range r.Header {
			if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "x-api-key") || strings.EqualFold(k, "Content-Length") {
				continue
			}
			for _, v := range vs {
				out.Header.Add(k, v)
			}
		}
		out.Header.Set("Content-Type", "application/json")
		// Replace the client's auth with the configured upstream key.
		if u.AuthHeader != "" && u.AuthValue != "" {
			out.Header.Set(u.AuthHeader, u.AuthValue)
		}
		return u.client.Do(out)
	}

	res := callWithRetry(ctx, do, p.cfg.Proxy.ProviderRetries, p.cfg.Proxy.ProviderRetryBackoffMs, p.sleep)
	if res.Err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: provider unreachable: " + res.Err.Error()))
		return http.StatusBadGateway, "text/plain"
	}
	// Stream the upstream body into w, preserving the upstream's
	// Content-Type (JSON, SSE, ...) so streamed responses pass through as-is.
	ct := res.ContentType
	if ct == "" {
		ct = "application/json"
	}
	w.Header().Set("Content-Type", ct)
	// M3: explicitly passthrough Content-Length and Transfer-Encoding from
	// the upstream response so the client sees the exact framing the
	// upstream used, instead of relying on Go to re-encode the framing
	// (which can drop Content-Length on a chunked body, or vice versa).
	if cl := res.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	if te := res.Header.Get("Transfer-Encoding"); te != "" {
		w.Header().Set("Transfer-Encoding", te)
	}
	w.WriteHeader(res.Status)
	if res.Body != nil {
		_, _ = io.Copy(w, res.Body)
		_ = res.Body.Close()
	}
	return res.Status, ct
}

// forwardWebSocket pipes a WebSocket upgrade request through to the upstream:
// it dials the upstream, relays the client's handshake, and pipes the
// hijacked connection in both directions. It is a pure passthrough — no
// savings transform, no retry (a failed handshake is reported to the client
// as a 502, never a 500; ADR-0016 fail-open).
func (p *Proxy) forwardWebSocket(w http.ResponseWriter, r *http.Request) error {
	return pipeWebSocket(w, r, p.upstream)
}

// baseUpstream captures the resolved upstream endpoint + auth for the provider
// call.
type baseUpstream struct {
	Name       string // "anthropic" | "openai"
	BaseURL    string
	AuthHeader string
	AuthValue  string
	client     *http.Client
}

// detectUpstream picks the upstream from config/env. It is a function of the
// http client and an env lookup so tests can inject deterministic values.
func detectUpstream(client *http.Client, env func(string) string) baseUpstream {
	if b := env("ANTHROPIC_BASE_URL"); b != "" {
		return baseUpstream{Name: "anthropic", BaseURL: b, AuthHeader: "x-api-key", AuthValue: env("ANTHROPIC_API_KEY"), client: client}
	}
	if b := env("OPENAI_BASE_URL"); b != "" {
		return baseUpstream{Name: "openai", BaseURL: b, AuthHeader: "Authorization", AuthValue: "Bearer " + env("OPENAI_API_KEY"), client: client}
	}
	// Default upstream: anthropic (the loopback proxy's primary target).
	return baseUpstream{Name: "anthropic", BaseURL: "https://api.anthropic.com", AuthHeader: "x-api-key", AuthValue: env("ANTHROPIC_API_KEY"), client: client}
}

// recordTelemetry writes the per-turn savings row + session counters to the
// ctx store. A nil store or a missing table is a no-op (ADR-0016 fail-open).
func (p *Proxy) recordTelemetry(r *http.Request, upstream, mode string, sv savings) {
	if p.store == nil {
		return
	}
	now := time.Now().UnixNano()
	sessionID := sessionID(r)
	turn := int64(1) // per-turn; T06/T11 refine the turn counter

	_ = p.store.PutProxySaving(&store.ProxySaving{
		SavingID:     fmt.Sprintf("save-%d-%s", now, sessionID),
		SessionID:    sessionID,
		Turn:         turn,
		Upstream:     upstream,
		Mode:         mode,
		TokensBefore: int64(sv.Before),
		TokensAfter:  int64(sv.After),
		SavingsRatio: sv.savingsRatio(),
		CreatedAt:    now,
	})

	// Session counters: upsert with the new turn's contribution.
	if prev, err := p.store.GetProxySession(sessionID); err == nil && prev != nil {
		_ = p.store.PutProxySession(&store.ProxySession{
			SessionID:          prev.SessionID,
			Agent:              prev.Agent,
			Mode:               mode,
			StartedAt:          prev.StartedAt,
			EndedAt:            now,
			Requests:           prev.Requests + 1,
			TokensBefore:       prev.TokensBefore + int64(sv.Before),
			TokensAfter:        prev.TokensAfter + int64(sv.After),
			CacheHits:          prev.CacheHits + boolToInt(sv.CacheHit),
			CacheInvalidations: prev.CacheInvalidations,
		})
	} else {
		_ = p.store.PutProxySession(&store.ProxySession{
			SessionID:    sessionID,
			Agent:        "proxy",
			Mode:         mode,
			StartedAt:    now,
			EndedAt:      now,
			Requests:     1,
			TokensBefore: int64(sv.Before),
			TokensAfter:  int64(sv.After),
			CacheHits:    boolToInt(sv.CacheHit),
		})
	}
}

// sessionID returns the request's stable session id: the X-CTX-Session header
// if present, else a stable id derived from the client address (RemoteAddr).
func sessionID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get(SessionHeader)); id != "" {
		return id
	}
	src := r.RemoteAddr
	if src == "" {
		src = "local"
	}
	sum := md5.Sum([]byte(src)) // #nosec G401
	return "sess-" + hex.EncodeToString(sum[:8])
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
