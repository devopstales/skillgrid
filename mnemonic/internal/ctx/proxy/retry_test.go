package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newStatusServer builds an upstream that always returns the given status.
func newStatusServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// noSleep records backoff durations so tests can assert the exponential
// sequence without waiting.
type noSleep struct{ calls []time.Duration }

func (n *noSleep) sleep(d time.Duration) { n.calls = append(n.calls, d) }

// TestCallWithRetrySuccessFirstTry proves a 2xx on the first attempt returns
// immediately with the body verbatim and no retries.
func TestCallWithRetrySuccessFirstTry(t *testing.T) {
	up := newStatusServer(t, http.StatusOK, `{"ok":true}`)
	defer up.Close()
	attempts := 0
	res := callWithRetry(context.Background(), func() (*http.Response, error) {
		attempts++
		req, _ := http.NewRequest(http.MethodPost, up.URL+"/v1/messages", nil)
		return http.DefaultClient.Do(req)
	}, 3, 200, realSleep)
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Status)
	}
	if string(res.Body) != `{"ok":true}` {
		t.Fatalf("body = %q, want verbatim", res.Body)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if res.Attempts != 1 {
		t.Errorf("res.Attempts = %d, want 1", res.Attempts)
	}
}

// TestCallWithRetryExhaustsBudget proves that a persistent non-2xx is retried
// up to maxRetries (1 + maxRetries total attempts), backs off exponentially
// (base, 2*base), then returns the provider's last status/body verbatim.
func TestCallWithRetryExhaustsBudget(t *testing.T) {
	attempts := 0
	base := 100
	ns := &noSleep{}
	res := callWithRetry(context.Background(), func() (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: http.StatusTooManyRequests, Body: http.NoBody}, nil
	}, 3, base, ns.sleep)

	// 1 initial + 3 retries = 4 attempts.
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4 (1 initial + 3 retries)", attempts)
	}
	if res.Attempts != 4 {
		t.Errorf("res.Attempts = %d, want 4", res.Attempts)
	}
	if res.Status != http.StatusTooManyRequests {
		t.Errorf("final status = %d, want 429 (verbatim)", res.Status)
	}
	// Backoff: base, 2*base, 4*base (after attempts 1, 2, 3).
	want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	if len(ns.calls) != len(want) {
		t.Fatalf("backoff calls = %d, want %d", len(ns.calls), len(want))
	}
	for i, w := range want {
		if ns.calls[i] != w {
			t.Errorf("backoff[%d] = %v, want %v", i, ns.calls[i], w)
		}
	}
}

// TestCallWithRetryVerbatimBody proves the final retry returns the provider's
// body verbatim (AC d: upstream error passes through after retries).
func TestCallWithRetryVerbatimBody(t *testing.T) {
	up := newStatusServer(t, http.StatusBadGateway, `{"error":"provider blew up"}`)
	defer up.Close()
	ns := &noSleep{}
	res := callWithRetry(context.Background(), func() (*http.Response, error) {
		req, _ := http.NewRequest(http.MethodPost, up.URL, nil)
		return http.DefaultClient.Do(req)
	}, 2, 50, ns.sleep)
	if res.Status != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", res.Status)
	}
	if string(res.Body) != `{"error":"provider blew up"}` {
		t.Fatalf("body = %q, want verbatim", res.Body)
	}
	// 1 + 2 retries = 3 attempts, 2 backoffs.
	if res.Attempts != 3 {
		t.Errorf("attempts = %d, want 3", res.Attempts)
	}
	if len(ns.calls) != 2 {
		t.Errorf("backoff calls = %d, want 2", len(ns.calls))
	}
}

// TestCallWithRetryTransportError proves a transport-level error (no upstream
// status) is retried and, on final failure, yields res.Err so the caller can
// render a 502.
func TestCallWithRetryTransportError(t *testing.T) {
	errs := 0
	ns := &noSleep{}
	res := callWithRetry(context.Background(), func() (*http.Response, error) {
		errs++
		return nil, errTestDial
	}, 2, 10, ns.sleep)
	if res.Err == nil {
		t.Fatalf("expected res.Err set on transport failure")
	}
	if res.Err != errTestDial {
		t.Errorf("res.Err = %v, want %v", res.Err, errTestDial)
	}
	if errs != 3 {
		t.Errorf("attempts = %d, want 3", errs)
	}
}

// TestBackoffMsSequence proves the exponential sequence base, 2*base, 4*base.
func TestBackoffMsSequence(t *testing.T) {
	if got := backoffMs(1, 200); got != 200*time.Millisecond {
		t.Errorf("backoff(1,200) = %v, want 200ms", got)
	}
	if got := backoffMs(2, 200); got != 400*time.Millisecond {
		t.Errorf("backoff(2,200) = %v, want 400ms", got)
	}
	if got := backoffMs(3, 200); got != 800*time.Millisecond {
		t.Errorf("backoff(3,200) = %v, want 800ms", got)
	}
	// base 0 → no backoff.
	if got := backoffMs(2, 0); got != 0 {
		t.Errorf("backoff(2,0) = %v, want 0", got)
	}
}
