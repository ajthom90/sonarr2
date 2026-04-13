package tvdb

import (
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitOptions configures a RateLimitedTransport.
type RateLimitOptions struct {
	// RequestsPerSecond is the steady-state token refill rate. Zero or negative
	// values default to 5 req/s.
	RequestsPerSecond float64

	// Burst is the maximum number of tokens available at once for burst traffic.
	// Zero or negative values default to 10.
	Burst int

	// MaxRetries is the number of times to retry a 429 response before giving
	// up and returning the final response to the caller. Zero or negative values
	// default to 3.
	MaxRetries int
}

// RateLimitedTransport is an http.RoundTripper that:
//   - enforces a client-side token-bucket rate limit before sending each request,
//   - retries 429 (Too Many Requests) responses with exponential backoff, and
//   - honours the Retry-After header when present.
type RateLimitedTransport struct {
	inner      http.RoundTripper
	limiter    *rate.Limiter
	maxRetries int
}

// NewRateLimitedTransport wraps inner with rate limiting and 429-retry logic.
// If inner is nil, http.DefaultTransport is used.
// Defaults: 5 req/s, burst 10, 3 retries.
func NewRateLimitedTransport(inner http.RoundTripper, opts RateLimitOptions) *RateLimitedTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}

	rps := opts.RequestsPerSecond
	if rps <= 0 {
		rps = 5
	}
	burst := opts.Burst
	if burst <= 0 {
		burst = 10
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &RateLimitedTransport{
		inner:      inner,
		limiter:    rate.NewLimiter(rate.Limit(rps), burst),
		maxRetries: maxRetries,
	}
}

// RoundTrip implements http.RoundTripper.
// It waits for a token from the rate limiter, executes the request, and retries
// on 429 up to maxRetries times with back-off.
func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Wait for the rate limiter before the first attempt.
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < t.maxRetries; attempt++ {
		if resp.StatusCode != http.StatusTooManyRequests {
			// Non-429: return as-is regardless of status code.
			return resp, nil
		}

		// Drain and close the 429 response body before retrying.
		_ = resp.Body.Close()

		delay := t.backoffDelay(resp, attempt)
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}

		// Acquire another token from the rate limiter before the retry.
		if err := t.limiter.Wait(req.Context()); err != nil {
			return nil, err
		}

		resp, err = t.inner.RoundTrip(req)
		if err != nil {
			return nil, err
		}
	}

	// Exhausted retries — return whatever we got last.
	return resp, nil
}

// backoffDelay returns the duration to wait before a retry attempt.
// If the response carries a Retry-After header with a non-negative integer of
// seconds, that value is used. Otherwise an exponential schedule is applied:
// attempt 0 → 1s, attempt 1 → 2s, attempt 2 → 4s, …
func (t *RateLimitedTransport) backoffDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if raw := resp.Header.Get("Retry-After"); raw != "" {
			if secs, err := strconv.Atoi(raw); err == nil && secs >= 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}

	// Exponential: 1 << attempt seconds.
	shift := attempt
	if shift > 6 {
		shift = 6 // cap at 64s
	}
	return time.Duration(1<<shift) * time.Second
}

// Ensure RateLimitedTransport satisfies the interface at compile time.
var _ http.RoundTripper = (*RateLimitedTransport)(nil)
