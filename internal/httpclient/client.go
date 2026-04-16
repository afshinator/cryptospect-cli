package httpclient

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	DefaultTimeout   = 10 * time.Second
	BackoffBaseDelay = 500 * time.Millisecond
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type APIError struct {
	StatusCode int
	Endpoint   string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: HTTP %d: %s", e.Endpoint, e.StatusCode, e.Body)
}

type RateLimitError struct {
	APIError
	RetryAfter time.Duration
}

type Client struct {
	doer       HTTPDoer
	maxRetries int
	sleep      func(time.Duration)
}

func New(maxRetries int) *Client {
	return &Client{
		doer:       &http.Client{Timeout: DefaultTimeout},
		maxRetries: maxRetries,
		sleep:      time.Sleep,
	}
}

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
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading response: %w", readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		apiErr := APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   url,
			Body:       string(body),
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

func (c *Client) GetWithKey(ctx context.Context, urlStr, key string) ([]byte, error) {
	if key != "" {
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, fmt.Errorf("parsing URL: %w", err)
		}
		q := u.Query()
		q.Set("x_cg_demo_api_key", key)
		u.RawQuery = q.Encode()
		urlStr = u.String()
	}
	return c.Get(ctx, urlStr)
}

func init() {
	// Seed the random number generator for jitter in backoff.
	// Using nanosecond time ensures different seeds across invocations.
	rand.Seed(time.Now().UnixNano())
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
