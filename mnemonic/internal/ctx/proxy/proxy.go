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
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// Proxy is the loopback LLM proxy. It owns the resolved config, a fail-open
// ctx store accessor for telemetry, a logger, and the effective mode (a plain
// field — switchable by T06).
type Proxy struct {
	cfg      *config.CtxConfig
	store    *store.Store
	log      *slog.Logger
	mode     Mode
	sleep    Sleep
	upstream baseUpstream
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "proxy: reading request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	upstream := p.upstream

	// Determine the effective mode for this request. T04: the startup-resolved
	// p.mode is authoritative (T06 adds per-request override). The mode used
	// for the transform is what the savings header reports.
	mode := p.mode

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
		msgs, perr := parseMessages(body)
		if perr != nil {
			// Unparseable: fail open to passthrough with the original body.
			forwardBody = body
			effectiveMode = ModePassthrough
		} else {
			tmsgs, terr := transform(mode, msgs)
			if terr != nil {
				// Compression fault: degrade to passthrough.
				forwardBody = body
				effectiveMode = ModePassthrough
			} else {
				if b, e := encodeMessages(tmsgs); e == nil {
					forwardBody = b
					cacheHit = mode == ModeCache && headSplit(msgs) > 0
				} else {
					forwardBody = body
					effectiveMode = ModePassthrough
				}
			}
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
	// or transport error.
	res := p.forward(r, forwardBody)

	// Emit telemetry (fail-open: a missing ctx table is a no-op write).
	p.recordTelemetry(r, upstream.Name, string(effectiveMode), sv)

	// Write the upstream response verbatim.
	w.Header().Set("Content-Type", res.contentType)
	w.WriteHeader(res.status)
	_, _ = w.Write(res.body)
}

// forward builds and issues the upstream request (with the configured API key
// injected, the client's Authorization/x-api-key replaced) and returns the
// upstream status/body verbatim after the retry budget.
func (p *Proxy) forward(r *http.Request, body []byte) forwardResult {
	u := p.upstream
	ctx := r.Context()

	baseURL := u.BaseURL
	path := r.URL.Path
	if baseURL == "" {
		// No upstream configured: fail open — return a 502 with a clear body
		// (the client still completes, no panic).
		return forwardResult{status: http.StatusBadGateway, contentType: "text/plain", body: []byte("proxy: no upstream configured for " + u.Name)}
	}

	fullURL := baseURL
	if !strings.HasSuffix(fullURL, "/") {
		fullURL += "/"
	}
	fullURL += strings.TrimLeft(path, "/")
	pu, err := url.Parse(fullURL)
	if err != nil {
		return forwardResult{status: http.StatusBadGateway, contentType: "text/plain", body: []byte("proxy: bad upstream URL: " + err.Error())}
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
		return forwardResult{
			status:      http.StatusBadGateway,
			contentType: "text/plain",
			body:        []byte("proxy: provider unreachable: " + res.Err.Error()),
		}
	}
	return forwardResult{
		status:      res.Status,
		contentType: "application/json",
		body:        res.Body,
	}
}

// forwardResult is the verbatim upstream outcome.
type forwardResult struct {
	status      int
	contentType string
	body        []byte
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
