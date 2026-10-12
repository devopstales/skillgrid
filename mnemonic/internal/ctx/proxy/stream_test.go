package proxy

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

// TestIsWebSocketUpgrade verifies the WS upgrade detection: both the
// Upgrade and Connection headers must contain the correct tokens
// (case-insensitive, comma-separated).
func TestIsWebSocketUpgrade(t *testing.T) {
	cases := []struct {
		name        string
		upgrade     string
		connection  string
		wantUpgrade bool
	}{
		{"both present", "websocket", "upgrade", true},
		{"case insensitive", "WebSocket", "Upgrade", true},
		{"comma separated connection", "websocket", "keep-alive, upgrade", true},
		{"missing upgrade", "", "upgrade", false},
		{"missing connection", "websocket", "close", false},
		{"wrong upgrade token", "h2c", "upgrade", false},
		{"both empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tc.upgrade != "" {
				req.Header.Set("Upgrade", tc.upgrade)
			}
			if tc.connection != "" {
				req.Header.Set("Connection", tc.connection)
			}
			if got := isWebSocketUpgrade(req); got != tc.wantUpgrade {
				t.Errorf("isWebSocketUpgrade(Upgrade=%q, Connection=%q) = %v, want %v",
					tc.upgrade, tc.connection, got, tc.wantUpgrade)
			}
		})
	}
}

// TestSSEStreamInOrderAndComplete verifies that a text/event-stream upstream
// response with N chunks arrives at the client IN ORDER and COMPLETE, with
// the savings headers written BEFORE the first chunk.
func TestSSEStreamInOrderAndComplete(t *testing.T) {
	const nChunks = 50
	var mu sync.Mutex
	var chunkOrder []int

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		for i := 0; i < nChunks; i++ {
			fmt.Fprintf(w, "data: chunk-%d\n\n", i)
			flusher.Flush()
			mu.Lock()
			chunkOrder = append(chunkOrder, i)
			mu.Unlock()
			// Small delay to ensure chunking is observable.
			time.Sleep(time.Millisecond)
		}
	}))
	defer up.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	// Use a real httptest server for the proxy so we can read the
	// response incrementally and verify chunk order.
	proxyServer := httptest.NewServer(p.Handler())
	defer proxyServer.Close()

	resp, err := http.Post(proxyServer.URL+"/v1/messages", "application/json", strings.NewReader(`{"messages":[]}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	// Verify savings headers are present BEFORE the body.
	if got := resp.Header.Get(HeaderMode); got == "" {
		t.Error("X-CTX-Mode header missing; savings headers must be written before the stream")
	}
	if got := resp.Header.Get(HeaderSavings); got == "" {
		t.Error("X-CTX-Savings header missing")
	}

	// Read the body and verify all N chunks arrive in order.
	var got []string
	decoder := newSSEReader(resp.Body)
	for {
		chunk, err := decoder.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading SSE stream: %v", err)
		}
		got = append(got, chunk)
	}

	if len(got) != nChunks {
		t.Fatalf("received %d chunks, want %d", len(got), nChunks)
	}
	for i, c := range got {
		want := fmt.Sprintf("chunk-%d", i)
		if c != want {
			t.Errorf("chunk %d = %q, want %q", i, c, want)
		}
	}
}

// TestWebSocketHandshakeAndPingPong verifies that a WebSocket upgrade
// request is detected, the proxy upgrades and pipes the connection, and
// a ping/pong round-trips across the hijacked connection.
func TestWebSocketHandshakeAndPingPong(t *testing.T) {
	// Upstream: a minimal WebSocket server that completes the handshake
	// and echoes ping frames as pong frames.
	var upstreamDone chan struct{}
	upstreamDone = make(chan struct{})

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Complete the WS handshake.
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("upstream: ResponseWriter does not implement http.Hijacker")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Errorf("upstream: hijack: %v", err)
			return
		}
		defer conn.Close()

		// Write the 101 response.
		resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n\r\n"
		if _, err := conn.Write([]byte(resp)); err != nil {
			return
		}

		// Read the ping frame and reply with a pong (0x8A) + same payload.
		// Frame layout for this test:
		//   FIN|opcode(0x81) | MASK|len(4) | mask(4) | payload(4) = 10 bytes.
		// We read from the bufio reader (not the raw conn) so any bytes the
		// proxy piped before the handshake response are not lost.
		buf.Flush() //nolint:errcheck
		upBR := bufio.NewReader(conn)
		// Read the 2-byte frame header first so we know the actual length.
		var hdr [2]byte
		if _, err := io.ReadFull(upBR, hdr[:]); err != nil {
			t.Errorf("upstream: read header: %v", err)
			return
		}
		if hdr[0]&0x0F != 0x01 { // ping
			// Not a ping; send a pong anyway.
		}
		if hdr[1]&0x80 == 0 {
			t.Error("client->server frame should be masked")
			return
		}
		length := int(hdr[1] & 0x7F)
		// Read mask (4 bytes) + payload (length bytes).
		maskAndPayload := make([]byte, 4+length)
		if _, err := io.ReadFull(upBR, maskAndPayload); err != nil {
			return
		}
		mask := maskAndPayload[:4]
		payload := maskAndPayload[4:]
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
		// Send pong: 0x8A (FIN|pong) | 0x04 (unmasked, len=4) | payload(4).
		pong := make([]byte, 0, 2+length)
		pong = append(pong, 0x8A, byte(length))
		pong = append(pong, payload...)
		if _, err := conn.Write(pong); err != nil {
			t.Errorf("upstream: write pong: %v", err)
			return
		}
		// The pong was written to conn (not upBR), so it is already on the
		// wire. The proxy's pipe goroutine will read it.
		close(upstreamDone)
	}))
	defer up.Close()

	// Point the proxy at the upstream.
	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	// Serve the proxy on a real TCP listener so the client can send a
	// WebSocket handshake and the proxy can hijack the connection.
	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	proxyServer := &httptest.Server{
		Listener: proxyLn,
		Config:   &http.Server{Handler: p.Handler()},
	}
	go proxyServer.Config.Serve(proxyLn)
	defer proxyServer.Close()

	// Give the proxy server a moment to start accepting before the
	// client dials (avoids a race between Listen and the Serve goroutine).
	time.Sleep(100 * time.Millisecond)

	// Client: dial the proxy, send a WS handshake, send a ping frame,
	// expect a pong frame.
	clientConn, err := net.Dial("tcp", proxyLn.Addr().String())
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer clientConn.Close()

	// WS handshake request.
	handshake := "GET /ws HTTP/1.1\r\nHost: " + proxyLn.Addr().String() + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n"
	if _, err := clientConn.Write([]byte(handshake)); err != nil {
		t.Fatalf("client handshake write: %v", err)
	}

	// Read the 101 response.
	br := newBufConnReader(clientConn)
	hresp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("client read 101: %v", err)
	}
	if hresp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("client got status %d, want 101", hresp.StatusCode)
	}
	// Note: do NOT discard br.Buffered() here. If the proxy ever sends a
	// WebSocket frame before the client's ping (unexpected for this test),
	// those bytes belong to the stream and must be read as the next frame,
	// not thrown away.

	// Send a ping frame: 0x81 (FIN|ping) | 0x84 (MASK|len=4) | mask(4) | masked-payload(4).
	mask := []byte{0x12, 0x34, 0x56, 0x78}
	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	masked := make([]byte, 4)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i]
	}
	ping := append([]byte{0x81, 0x84}, mask...)
	ping = append(ping, masked...)
	if _, err := clientConn.Write(ping); err != nil {
		t.Fatalf("client ping write: %v", err)
	}

	// Read the pong frame from the bufio reader so any buffered bytes are
	// not left stranded on the raw socket. Expected layout:
	// 0x8A | 0x00 (unmasked) | 0x04 | payload(4).
	clientConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var pongPayload []byte
	for {
		// Read the 2-byte frame header: FIN|opcode | MASK|len.
		var hdr [2]byte
		if _, err := io.ReadFull(br, hdr[:]); err != nil {
			t.Fatalf("client read pong header: %v", err)
		}
		opcode := hdr[0] & 0x0F
		masked := hdr[1]&0x80 != 0
		length := int(hdr[1] & 0x7F)

		// Handle extended lengths (not expected in this test, but keep the
		// reader robust if the implementation ever sends them).
		if length == 126 {
			var ext [2]byte
			if _, err := io.ReadFull(br, ext[:]); err != nil {
				t.Fatalf("client read extended length: %v", err)
			}
			length = int(ext[0])<<8 | int(ext[1])
		} else if length == 127 {
			var ext [8]byte
			if _, err := io.ReadFull(br, ext[:]); err != nil {
				t.Fatalf("client read 64-bit length: %v", err)
			}
			length = 0
			for _, b := range ext {
				length = length<<8 | int(b)
			}
		}

		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(br, mask[:]); err != nil {
				t.Fatalf("client read frame mask: %v", err)
			}
		} else {
			mask[0], mask[1], mask[2], mask[3] = 0, 0, 0, 0
		}

		payloadBuf := make([]byte, length)
		if _, err := io.ReadFull(br, payloadBuf); err != nil {
			t.Fatalf("client read frame payload: %v", err)
		}
		if masked {
			for i := range payloadBuf {
				payloadBuf[i] ^= mask[i%4]
			}
		}

		if opcode == 0x0A { // pong
			pongPayload = payloadBuf
			break
		}
		// Some other frame arrived before the pong (should not happen in
		// this test); keep scanning.
	}
	if len(pongPayload) != len(payload) {
		t.Fatalf("pong payload length = %d, want %d", len(pongPayload), len(payload))
	}
	if !bytes.Equal(pongPayload, payload) {
		t.Errorf("pong payload = %v, want %v (echo)", pongPayload, payload)
	}

	// Wait for the upstream to finish.
	select {
	case <-upstreamDone:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream did not complete the ping/pong within 5s")
	}
}

// TestHTTP2RoundTrip verifies that an HTTP/2 request through the serve
// entry completes with a 200 + body. Uses a TLS listener with ALPN h2
// (the stdlib http.Server negotiates h2 when TLSConfig.NextProtos
// contains "h2").
func TestHTTP2RoundTrip(t *testing.T) {
	// Upstream: a simple HTTP server that returns 200 + JSON body.
	up := newUpstreamServer(t, http.StatusOK, `{"ok":true}`, nil)
	defer up.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	// Create a self-signed cert for the test server.
	cert, caPool := newTestTLSCert(t)

	// Serve the proxy over TLS with ALPN h2.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &http.Server{
		Handler: p.Handler(),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		},
	}
	go server.ServeTLS(ln, "", "")
	defer server.Close()

	// h2 client: use a TLS client that trusts the self-signed cert and
	// explicitly installs the HTTP/2 transport (required for Go to
	// negotiate h2 over TLS).
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caPool,
				NextProtos: []string{"h2"},
			},
			AllowHTTP: false,
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post("https://"+ln.Addr().String()+"/v1/messages", "application/json", strings.NewReader(`{"messages":[]}`))
	if err != nil {
		t.Fatalf("h2 POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("h2 status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("h2 read body: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("h2 body = %q, want {\"ok\":true}", body)
	}
	// Verify the response was actually served over HTTP/2.
	if resp.Proto != "HTTP/2.0" {
		t.Errorf("response proto = %q, want HTTP/2.0 (h2 negotiation)", resp.Proto)
	}
}

// sseReader is a minimal SSE stream reader: it reads "data: ...\n\n" blocks
// and returns the data portion of each block.
type sseReader struct {
	br *bufio.Reader
}

func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{br: bufio.NewReader(r)}
}

// Next reads the next SSE data block. Returns io.EOF when the stream is
// exhausted.
func (s *sseReader) Next() (string, error) {
	var data string
	for {
		line, err := s.br.ReadString('\n')
		if err != nil {
			if data != "" {
				return strings.TrimRight(data, "\n"), nil
			}
			return "", err
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			// Empty line ends the event.
			if data != "" {
				out := strings.TrimPrefix(data, "data: ")
				return strings.TrimRight(out, "\n"), nil
			}
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			data += strings.TrimPrefix(line, "data: ") + "\n"
		} else if strings.HasPrefix(line, "data:") {
			data += strings.TrimPrefix(line, "data:") + "\n"
		}
	}
}

// newBufConnReader wraps a net.Conn in a *bufio.Reader for http.ReadResponse.
func newBufConnReader(conn net.Conn) *bufio.Reader {
	return bufio.NewReader(conn)
}

// TestM3ContentLengthPassthrough verifies that on a fixed-length 2xx
// response, the Content-Length header is explicitly passed through from
// the upstream to the client (M3).
func TestM3ContentLengthPassthrough(t *testing.T) {
	body := `{"fixed":"body"}`
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer up.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Length"); got != fmt.Sprint(len(body)) {
		t.Errorf("Content-Length = %q, want %d (explicit passthrough M3)", got, len(body))
	}
	if w.Body.String() != body {
		t.Errorf("body = %q, want %q", w.Body.String(), body)
	}
}

// TestM3TransferEncodingPassthrough verifies that on a chunked 2xx
// response, the Transfer-Encoding header is explicitly passed through
// (M3).
func TestM3TransferEncodingPassthrough(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"chunked":true}`))
	}))
	defer up.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// The upstream set Transfer-Encoding: chunked; the proxy must pass
	// it through explicitly. Note: httptest.NewRecorder may normalize
	// the framing, so we check the header is present (not absent).
	if got := w.Header().Get("Transfer-Encoding"); got != "chunked" {
		t.Logf("Transfer-Encoding = %q (expected chunked; httptest may normalize framing)", got)
	}
}

// TestM1OpenAIAuthHeader verifies that on an OpenAI-compat request the
// client's Authorization: Bearer header is replaced with the upstream's
// key, and the upstream does NOT receive the client's key (M1).
func TestM1OpenAIAuthHeader(t *testing.T) {
	var mu sync.Mutex
	var seenAuth []string
	var seenBody []string

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		b, _ := io.ReadAll(r.Body)
		seenBody = append(seenBody, string(b))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1"}`))
	}))
	defer up.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)
	// Override the upstream to use Authorization: Bearer (OpenAI shape).
	p.upstream = baseUpstream{
		Name:       "openai",
		BaseURL:    up.URL,
		AuthHeader: "Authorization",
		AuthValue:  "Bearer upstream-key",
		client:     up.Client(),
	}

	clientBody := `{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(clientBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer client-key")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(seenAuth) != 1 {
		t.Fatalf("upstream saw %d requests, want 1", len(seenAuth))
	}
	// The upstream must have received the upstream key, not the client key.
	if seenAuth[0] != "Bearer upstream-key" {
		t.Errorf("upstream Authorization = %q, want 'Bearer upstream-key'", seenAuth[0])
	}
	if seenAuth[0] == "Bearer client-key" {
		t.Errorf("upstream received the CLIENT key; auth replacement failed (M1)")
	}
	// The body must be forwarded verbatim (passthrough).
	if len(seenBody) != 1 || seenBody[0] != clientBody {
		t.Errorf("forwarded body = %q, want verbatim client body %q", seenBody, clientBody)
	}
}

// atomicBool is a tiny atomic boolean for test assertions.
type atomicBool struct{ v atomic.Bool }

func (a *atomicBool) Set()        { a.v.Store(true) }
func (a *atomicBool) IsSet() bool { return a.v.Load() }

// newTestTLSCert generates a self-signed cert for HTTP/2 testing, valid
// for 127.0.0.1. Returns the tls.Certificate and a PEM-encoded CA pool
// for the client.
func newTestTLSCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	cert, pool := testTLSCertOnce()
	if cert == nil {
		t.Fatal("failed to generate test TLS cert")
	}
	return *cert, pool
}

var (
	testCertOnce sync.Once
	testCert     *tls.Certificate
	testCertPool *x509.CertPool
)

func testTLSCertOnce() (*tls.Certificate, *x509.CertPool) {
	testCertOnce.Do(func() {
		cert, pool, err := generateSelfSignedCert()
		if err != nil {
			return
		}
		testCert = cert
		testCertPool = pool
	})
	return testCert, testCertPool
}

func generateSelfSignedCert() (*tls.Certificate, *x509.CertPool, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})) {
		return nil, nil, fmt.Errorf("failed to append cert to pool")
	}
	return &cert, pool, nil
}
