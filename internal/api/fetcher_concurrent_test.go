package api

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/config"
)

// delayedDoer implements httpclient.HTTPDoer with a fixed delay before responding.
type delayedDoer struct {
	delay      time.Duration
	statusCode int
	body       string
	mu         sync.Mutex
	calls      map[string]int // endpoint -> call count
}

func (d *delayedDoer) Do(req *http.Request) (*http.Response, error) {
	time.Sleep(d.delay)
	d.mu.Lock()
	if d.calls == nil {
		d.calls = make(map[string]int)
	}
	d.calls[req.URL.String()]++
	d.mu.Unlock()
	return &http.Response{
		StatusCode: d.statusCode,
		Body:       &mockReadCloser{data: []byte(d.body)},
		Header:     make(http.Header),
	}, nil
}

func TestFetchConcurrentDifferentEndpoints(t *testing.T) {
	// This test verifies that concurrent fetches to different endpoints
	// do not block each other (sharded locking).
	// If sharding works, total time should be ~delay (not delay * N).
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

	// Delay each HTTP response by 100ms.
	delay := 100 * time.Millisecond
	doer := &delayedDoer{
		delay:      delay,
		statusCode: http.StatusOK,
		body:       `{"test": "data"}`,
	}

	f, err := New(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.httpClient.SetDoer(doer)

	ctx := context.Background()
	endpoints := []string{
		CoinGeckoGlobalMarket,
		CoinGeckoSPPStablesMarkets,
		CoinGeckoDerivatives,
		CoinGeckoCoinMarketsBreadth,
		CoinGeckoCoinMarketsMomentum,
		BinanceSpotCVD_BTC_1h,
	}

	var wg sync.WaitGroup
	start := time.Now()
	for _, ep := range endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			_, _, err := f.Fetch(ctx, endpoint)
			if err != nil {
				t.Errorf("fetch %q: %v", endpoint, err)
			}
		}(ep)
	}
	wg.Wait()
	elapsed := time.Since(start)

	// With sharding, elapsed should be close to delay (some overhead).
	// Without sharding, elapsed would be ~delay * len(endpoints) (6 * 100ms = 600ms).
	// We'll check that elapsed is significantly less than that bound to detect serialization.
	// Allow for hash collisions (some endpoints may land in same shard).
	// Conservative threshold: delay * (len(endpoints) - 1) = 500ms
	maxExpected := delay * time.Duration(len(endpoints)-1)
	if elapsed > maxExpected {
		t.Errorf("concurrent fetches took %v, expected ≤ %v (indicating serialization)", elapsed, maxExpected)
	}

	// Each endpoint should have been called exactly once (no duplicate HTTP calls).
	doer.mu.Lock()
	defer doer.mu.Unlock()

	totalCalls := 0
	for _, c := range doer.calls {
		totalCalls += c
	}
	if totalCalls != len(endpoints) {
		t.Errorf("expected %d HTTP calls, got %d", len(endpoints), totalCalls)
	}
}
