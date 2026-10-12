package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// isWebSocketUpgrade reports whether r is a WebSocket handshake request:
// the Upgrade header contains "websocket" (case-insensitive) and the
// Connection header contains "upgrade" (case-insensitive, possibly
// comma-separated). Both must be present for the proxy to hijack the
// connection; a partial match falls through to the normal HTTP path.
func isWebSocketUpgrade(r *http.Request) bool {
	if !isTokenInHeader(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	return isTokenInHeader(r.Header.Get("Connection"), "upgrade")
}

// isTokenInHeader reports whether header value v contains token t as a
// case-insensitive, comma-separated token. An empty v never matches.
func isTokenInHeader(v, token string) bool {
	for _, part := range strings.Split(strings.ToLower(v), ",") {
		if strings.TrimSpace(part) == token {
			return true
		}
	}
	return false
}

// pipeWebSocket upgrades both the client and upstream connections to a
// WebSocket and pipes the raw byte streams in both directions until either
// side closes. The upstream is dialed with the client's request headers
// (auth replaced with the proxy's upstream credentials) so the handshake
// completes identically to a direct client-to-upstream connection.
//
// It returns the first error that broke either pipe; the client connection
// is always closed before returning. If w does not implement http.Hijacker
// (e.g. an httptest.ResponseRecorder), it returns 501 without touching the
// client.
func pipeWebSocket(w http.ResponseWriter, r *http.Request, u baseUpstream) error {
	clientHijacker, ok := w.(http.Hijacker)
	if !ok {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte("proxy: websocket upgrade not supported on this connection"))
		return errNoHijacker
	}

	// Build the upstream handshake request: same path, same host, the
	// client's headers minus the hop-by-hop ones, plus the proxy's auth.
	handshake := newWebSocketHandshake(r, u)

	// Open the upstream TCP connection for the handshake.
	host := handshake.URL.Host
	conn, err := net.Dial("tcp", host)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: websocket upstream unreachable: " + err.Error()))
		return err
	}
	// Close the upstream conn whenever this function returns. The pipe
	// goroutines also close it, but Close is idempotent.
	defer conn.Close()

	// Write the upstream handshake request.
	if err := handshake.Write(conn); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: websocket handshake write: " + err.Error()))
		return err
	}

	// Read the upstream handshake response. The bufio reader may buffer
	// bytes past the 101 status line (the first WebSocket frame from the
	// upstream), so it must be used as the upstream->client pipe source
	// to avoid losing those bytes.
	br := bufio.NewReaderSize(conn, 16384)
	hresp, err := http.ReadResponse(br, handshake)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: websocket handshake read: " + err.Error()))
		return err
	}
	if hresp.StatusCode != http.StatusSwitchingProtocols {
		_, _ = io.Copy(io.Discard, hresp.Body)
		_ = hresp.Body.Close()
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("proxy: websocket upstream rejected handshake (status " + strconv.Itoa(hresp.StatusCode) + ")"))
		return err
	}

	// Hijack the client connection so we own the raw bytes.
	clientConn, _, err := clientHijacker.Hijack()
	if err != nil {
		return err
	}
	defer clientConn.Close()

	// Relay the 101 response to the client over the hijacked connection.
	if err := hresp.Write(clientConn); err != nil {
		return err
	}
	_ = hresp.Body.Close()

	// Bidirectional pipe. When either direction breaks, close both
	// connections so the other pipe unblocks and the client sees a
	// clean close (no half-open socket).
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, clientConn)
		_ = conn.Close()
	}()
	go func() {
		defer wg.Done()
		// Read from br (not conn) so buffered bytes past the 101 are
		// delivered to the client, not lost.
		_, _ = io.Copy(clientConn, br)
		_ = clientConn.Close()
	}()
	wg.Wait()
	return nil
}

// errNoHijacker is returned by pipeWebSocket when the ResponseWriter does
// not implement http.Hijacker (the connection cannot be upgraded).
var errNoHijacker = http.ErrNotSupported

// newWebSocketHandshake builds the HTTP request the proxy sends to the
// upstream to complete the WebSocket handshake. It copies the client's
// path, query, and headers (minus hop-by-hop headers and the client's
// auth), then injects the proxy's upstream credentials. The request uses
// the client's context so a client-side cancellation propagates.
func newWebSocketHandshake(r *http.Request, u baseUpstream) *http.Request {
	handshake, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, websocketHandshakeURL(r, u), nil)
	for k, vs := range r.Header {
		if isHopByHopHeader(k) {
			continue
		}
		for _, v := range vs {
			handshake.Header.Add(k, v)
		}
	}
	if u.AuthHeader != "" && u.AuthValue != "" {
		handshake.Header.Set(u.AuthHeader, u.AuthValue)
	}
	return handshake
}

// websocketHandshakeURL computes the full upstream URL for the WebSocket
// handshake from the client's request path/query and the upstream base URL.
func websocketHandshakeURL(r *http.Request, u baseUpstream) string {
	base := u.BaseURL
	if base != "" && base[len(base)-1] != '/' {
		base += "/"
	}
	path := trimLeadingSlash(r.URL.Path)
	if r.URL.RawQuery != "" {
		return base + path + "?" + r.URL.RawQuery
	}
	return base + path
}

// trimLeadingSlash removes a single leading "/" from s.
func trimLeadingSlash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}

// isHopByHopHeader reports whether h is an RFC 7230 hop-by-hop header that
// must not be forwarded to the upstream on a WebSocket handshake.
func isHopByHopHeader(h string) bool {
	switch strings.ToLower(h) {
	case "connection", "upgrade", "keep-alive", "proxy-authenticate",
		"proxy-authorization", "te", "trailers", "transfer-encoding":
		return true
	}
	return false
}
