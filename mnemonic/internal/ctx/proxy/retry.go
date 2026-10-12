package proxy

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Sleep is the injectable delay used between provider retries. Production
// uses time.Sleep; tests use a no-op (or recording) function so backoff is
// exercised deterministically without real waiting.
type Sleep func(time.Duration)

// realSleep is the production Sleep.
func realSleep(d time.Duration) { time.Sleep(d) }

// retryResult carries the provider's outcome. On final failure, Status is the
// upstream status code and Body is the upstream body verbatim; on an
// unrecoverable transport error (no upstream status at all) Body may be the
// rendered error and Status is 0.
type retryResult struct {
	Status int
	Body   []byte
	// Attempts is how many times the provider call was actually issued.
	Attempts int
	// Err is non-nil only for a transport-level failure with no usable
	// upstream status (the caller renders a 502).
	Err error
}

// callWithRetry issues do, retrying on a non-2xx upstream response with
// exponential backoff. The first backoff is baseMs; each subsequent backoff is
// doubled. It issues at most 1 + maxRetries attempts. After the final retry it
// returns the provider's last status/body verbatim (no 500 — ADR-0016).
//
// A transport-level error (do returns a non-nil err with no response, e.g. a
// connection refused or a client-side timeout) is also retried; if the final
// attempt still yields no status, retryResult.Err is set so the caller renders
// a 502 (the provider is unreachable — there is no verbatim body to pass).
func callWithRetry(ctx context.Context, do func() (*http.Response, error), maxRetries int, baseMs int, sleep Sleep) retryResult {
	if maxRetries < 0 {
		maxRetries = 0
	}
	attempt := 0
	var last retryResult
	for {
		resp, err := do()
		attempt++

		if err != nil {
			// Transport-level failure: no upstream status.
			last = retryResult{Attempts: attempt, Err: err}
			if attempt > maxRetries {
				return last
			}
			sleep(backoffMs(attempt, baseMs))
			continue
		}

		// Read the body fully so we can replay it verbatim on the final
		// attempt (and so the connection is returned to the pool).
		body := readAll(resp)
		code := resp.StatusCode

		// A 2xx success is returned immediately.
		if code >= 200 && code < 300 {
			return retryResult{Status: code, Body: body, Attempts: attempt}
		}

		// Non-2xx: remember the last status/body (verbatim for passthrough)
		// and retry until we exhaust the budget.
		last = retryResult{Status: code, Body: body, Attempts: attempt}
		if attempt > maxRetries {
			return last
		}
		sleep(backoffMs(attempt, baseMs))
	}
}

// backoffMs returns the backoff to apply after the given (1-based) attempt has
// failed: baseMs * 2^(attempt-1). This yields the sequence base, 2*base,
// 4*base, ... starting from the first failure.
func backoffMs(attempt, baseMs int) time.Duration {
	if baseMs < 0 {
		baseMs = 0
	}
	if attempt < 1 {
		attempt = 1
	}
	ms := baseMs
	for i := 1; i < attempt; i++ {
		ms *= 2
	}
	return time.Duration(ms) * time.Millisecond
}

// errNoUpstream marks a provider failure with no upstream status to pass.
var errNoUpstream = errors.New("proxy: no upstream status (provider unreachable)")

// readAll consumes and closes resp.Body, returning its bytes. A read error is
// swallowed: the bytes read so far are still usable for the verbatim passthrough.
func readAll(resp *http.Response) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()
	var buf []byte
	tmp := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			return buf
		}
	}
}
