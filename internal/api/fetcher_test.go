package api

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
)

// countingDoer implements httpclient.HTTPDoer and counts requests.
type countingDoer struct {
	count      int
	serverURL  string
	statusCode int
	body       string
}

func (c *countingDoer) Do(req *http.Request) (*http.Response, error) {
	c.count++
	// Rewrite request to test server if serverURL is set.
	if c.serverURL != "" {
		newReq := req.Clone(req.Context())
		newReq.URL.Host = c.serverURL[7:] // strip "http://"
		newReq.URL.Scheme = "http"
		return http.DefaultTransport.RoundTrip(newReq)
	}
	// Otherwise return a canned response.
	return &http.Response{
		StatusCode: c.statusCode,
		Body:       &mockReadCloser{data: []byte(c.body)},
		Header:     make(http.Header),
	}, nil
}

// mockReadCloser implements io.ReadCloser for test responses.
type mockReadCloser struct {
	data []byte
	pos  int
}

func (m *mockReadCloser) Read(p []byte) (int, error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n := copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}

// TestFileCacheFreshHit verifies that a fresh file cache entry is used on subsequent
// fetches (different fetcher instance, same cache directory).
func TestFileCacheFreshHit(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     map[string]int{},
		},
		APIs: config.APIsConfig{
			CoinGecko: config.APIKeyConfig{APIKey: "test-key"},
			Binance:   config.APIKeyConfig{APIKey: "test-key"},
		},
	}

	// Create a counting doer that will serve a response.
	body := `{"cached": true}`
	doer := &countingDoer{
		serverURL:  "", // no server, we'll fake response
		statusCode: http.StatusOK,
		body:       body,
	}

	// First fetcher: populate cache.
	f1, err := New(dir, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	f1.httpClient.SetDoer(doer)

	ctx := context.Background()
	endpoint := CoinGeckoGlobalMarket

	data1, meta1, err := f1.Fetch(ctx, endpoint)
	if err != nil {
		t.Fatalf("first fetch error: %v", err)
	}
	if string(data1) != body {
		t.Errorf("first fetch data = %q, want %q", data1, body)
	}
	if meta1.CacheHit {
		t.Error("first fetch should not be a cache hit")
	}
	if doer.count != 1 {
		t.Errorf("expected 1 HTTP call, got %d", doer.count)
	}

	// Second fetcher, same cache directory, fresh memory.
	f2, err := New(dir, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	f2.httpClient.SetDoer(doer) // same doer, will count if called

	data2, meta2, err := f2.Fetch(ctx, endpoint)
	if err != nil {
		t.Fatalf("second fetch error: %v", err)
	}
	if string(data2) != body {
		t.Errorf("second fetch data = %q, want %q", data2, body)
	}
	if !meta2.CacheHit {
		t.Error("second fetch should be a cache hit (fresh file cache)")
	}
	if doer.count != 1 {
		t.Errorf("expected no additional HTTP calls, got total %d", doer.count)
	}
}

// TestFileCacheStaleAPISuccess verifies that a stale cache entry is refreshed when the API succeeds.
func TestFileCacheStaleAPISuccess(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     map[string]int{},
		},
		APIs: config.APIsConfig{
			CoinGecko: config.APIKeyConfig{APIKey: "test-key"},
			Binance:   config.APIKeyConfig{APIKey: "test-key"},
		},
	}

	// Create a cache entry with TTL = 0 (immediately stale).
	cacheCli, err := cache.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cacheCli.Close() }()

	staleData := `{"stale": true}`
	endpoint := CoinGeckoGlobalMarket
	if err := cacheCli.Set(endpoint, []byte(staleData), 0); err != nil {
		t.Fatal(err)
	}

	// Counting doer that returns fresh data.
	freshBody := `{"fresh": true}`
	doer := &countingDoer{
		serverURL:  "",
		statusCode: http.StatusOK,
		body:       freshBody,
	}

	f, err := New(dir, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.httpClient.SetDoer(doer)

	ctx := context.Background()
	data, meta, err := f.Fetch(ctx, endpoint)
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if string(data) != freshBody {
		t.Errorf("data = %q, want fresh data %q", data, freshBody)
	}
	if meta.CacheHit {
		t.Error("should not be a cache hit (stale cache not served)")
	}
	if meta.Stale {
		t.Error("should not be stale (fresh data from API)")
	}
	if doer.count != 1 {
		t.Errorf("expected 1 HTTP call, got %d", doer.count)
	}

	// Verify cache was updated with fresh data.
	entry, err := cacheCli.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Found {
		t.Error("cache entry should exist")
	}
	if string(entry.Data) != freshBody {
		t.Errorf("cached data = %q, want %q", entry.Data, freshBody)
	}
	if entry.Stale {
		t.Error("cache entry should not be stale after refresh")
	}
}

// TestFileCacheStaleAPIFailFallback verifies that a stale cache entry is served as fallback when the API fails.
func TestFileCacheStaleAPIFailFallback(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     map[string]int{},
		},
		APIs: config.APIsConfig{
			CoinGecko: config.APIKeyConfig{APIKey: "test-key"},
			Binance:   config.APIKeyConfig{APIKey: "test-key"},
		},
	}

	// Create a stale cache entry.
	cacheCli, err := cache.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cacheCli.Close() }()

	staleData := `{"stale": true}`
	endpoint := CoinGeckoGlobalMarket
	if err := cacheCli.Set(endpoint, []byte(staleData), 0); err != nil {
		t.Fatal(err)
	}

	// Doer that returns a 500 error.
	doer := &countingDoer{
		serverURL:  "",
		statusCode: http.StatusInternalServerError,
		body:       `{"error": "server down"}`,
	}

	f, err := New(dir, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.httpClient.SetDoer(doer)

	ctx := context.Background()
	data, meta, err := f.Fetch(ctx, endpoint)
	if err != nil {
		t.Fatalf("fetch should not error (stale fallback), got %v", err)
	}
	if string(data) != staleData {
		t.Errorf("data = %q, want stale data %q", data, staleData)
	}
	if meta.CacheHit {
		t.Error("should not be a cache hit (served stale fallback)")
	}
	if !meta.Stale {
		t.Error("should be stale (served from stale cache)")
	}
	if meta.TTLRemaining != 0 {
		t.Errorf("TTLRemaining = %d, want 0", meta.TTLRemaining)
	}
	// maxRetries = 3, so attempts = 4
	expectedCalls := 4
	if doer.count != expectedCalls {
		t.Errorf("expected %d HTTP calls (including retries), got %d", expectedCalls, doer.count)
	}
}

// TestNoCacheAPISuccess verifies that when cache is disabled, data is fetched from API and not stored.
func TestNoCacheAPISuccess(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Cache: config.CacheConfig{
			Enabled: false, // cache disabled
			TTL:     map[string]int{},
		},
		APIs: config.APIsConfig{
			CoinGecko: config.APIKeyConfig{APIKey: "test-key"},
			Binance:   config.APIKeyConfig{APIKey: "test-key"},
		},
	}

	body := `{"live": true}`
	doer := &countingDoer{
		serverURL:  "",
		statusCode: http.StatusOK,
		body:       body,
	}

	f, err := New(dir, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.httpClient.SetDoer(doer)

	ctx := context.Background()
	endpoint := CoinGeckoGlobalMarket

	data, meta, err := f.Fetch(ctx, endpoint)
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if string(data) != body {
		t.Errorf("data = %q, want %q", data, body)
	}
	if meta.CacheHit {
		t.Error("should not be a cache hit (cache disabled)")
	}
	if meta.Stale {
		t.Error("should not be stale")
	}
	if doer.count != 1 {
		t.Errorf("expected 1 HTTP call, got %d", doer.count)
	}

	// Verify nothing was written to cache.
	cacheCli, err := cache.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cacheCli.Close() }()
	entry, err := cacheCli.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Found {
		t.Error("cache entry should not exist when cache disabled")
	}
}

// TestNoCacheAPIFail verifies that when cache is disabled and API fails, an error is returned.
func TestNoCacheAPIFail(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
			TTL:     map[string]int{},
		},
		APIs: config.APIsConfig{
			CoinGecko: config.APIKeyConfig{APIKey: "test-key"},
			Binance:   config.APIKeyConfig{APIKey: "test-key"},
		},
	}

	doer := &countingDoer{
		serverURL:  "",
		statusCode: http.StatusInternalServerError,
		body:       `{"error": "server down"}`,
	}

	f, err := New(dir, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.httpClient.SetDoer(doer)

	ctx := context.Background()
	endpoint := CoinGeckoGlobalMarket

	data, meta, err := f.Fetch(ctx, endpoint)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if data != nil {
		t.Error("data should be nil on error")
	}
	// meta should be zero value
	if meta.CacheHit || meta.Stale || meta.TTLRemaining != 0 || !meta.FetchedAt.IsZero() {
		t.Errorf("meta should be zero value, got %+v", meta)
	}
	// maxRetries = 3, so attempts = 4
	expectedCalls := 4
	if doer.count != expectedCalls {
		t.Errorf("expected %d HTTP calls (including retries), got %d", expectedCalls, doer.count)
	}
}
