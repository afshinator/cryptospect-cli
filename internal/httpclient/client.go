package httpclient

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 10 * time.Second
	// BackoffBaseDelay is the base delay for exponential backoff.
	BackoffBaseDelay = 500 * time.Millisecond
)

// HTTPDoer is an interface for HTTP clients (compatible with http.Client).
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// APIError represents an HTTP API error (non‑2xx status).
type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// RateLimitError represents an HTTP 429 Too Many Requests error.
type RateLimitError struct {
	APIError
	RetryAfter time.Duration
}

// Client is an HTTP client with retry and backoff.
type Client struct {
	doer       HTTPDoer
	maxRetries int
	sleep      func(time.Duration)
}

// New creates a new HTTP client with the given max retries.
func New(maxRetries int) *Client {
	return &Client{
		doer:       &http.Client{Timeout: DefaultTimeout},
		maxRetries: maxRetries,
		sleep:      time.Sleep,
	}
}

// Get performs an HTTP GET request with retry and backoff.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	sleepFn := c.sleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			sleepFn(backoffDuration(attempt))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}

		req.Header.Set("User-Agent", "cryptospect-cli/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := c.doer.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading response: %w", readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		apiErr := APIError{
			StatusCode: resp.StatusCode,
			Body:       body,
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, &RateLimitError{
				APIError:   apiErr,
				RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
			}
		}

		if resp.StatusCode < 500 {
			return nil, &apiErr
		}

		lastErr = &apiErr
	}
	return nil, lastErr
}


func backoffDuration(attempt int) time.Duration {
	d := BackoffBaseDelay * (1 << (attempt - 1))
	jitter := time.Duration(rand.Int63n(int64(d)/2)) - d/4
	return d + jitter
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// SetDoer replaces the underlying HTTPDoer. This is intended for testing only.
func (c *Client) SetDoer(doer HTTPDoer) {
	c.doer = doer
}
