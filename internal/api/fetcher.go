package api

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"
	"unique"

	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/api/coindesk"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/api/coinmetrics"
	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/httpclient"
)

var (
	handleMap sync.Map // string -> unique.Handle[string]
)

func getHandle(key string) unique.Handle[string] {
	if h, ok := handleMap.Load(key); ok {
		return h.(unique.Handle[string])
	}
	h := unique.Make(key)
	handleMap.Store(key, h)
	return h
}

const shardCount = 16 // must be power of two

// FetchMeta holds metadata about how the data was obtained.
type FetchMeta struct {
	CacheHit     bool      // true when served from a fresh cache entry
	Stale        bool      // true when stale cache was used as fallback (API unavailable)
	TTLRemaining int       // remaining seconds until cache entry expires
	FetchedAt    time.Time // when the data was originally fetched (UTC)
}

type shard struct {
	mu         sync.Mutex
	memory     map[unique.Handle[string]][]byte
	memoryMeta map[unique.Handle[string]]FetchMeta
}

// Fetcher orchestrates cache‑first fetching of endpoint data.
type Fetcher struct {
	cache      *cache.Cache
	httpClient *httpclient.Client
	config     config.Config

	shards    []*shard
	shardMask uint32
}

func (f *Fetcher) shardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() & f.shardMask
}

// New creates a Fetcher with the given cache directory and configuration.
// The cache directory will be created if it does not exist.
func New(cacheDir string, cfg config.Config) (*Fetcher, error) {
	cacheCli, err := cache.Open(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("opening cache: %w", err)
	}

	httpCli := httpclient.New(3) // default 3 retries

	shards := make([]*shard, shardCount)
	for i := range shards {
		shards[i] = &shard{
			memory:     make(map[unique.Handle[string]][]byte),
			memoryMeta: make(map[unique.Handle[string]]FetchMeta),
		}
	}
	return &Fetcher{
		cache:      cacheCli,
		httpClient: httpCli,
		config:     cfg,
		shards:     shards,
		shardMask:  shardCount - 1,
	}, nil
}

// Fetch retrieves data for the given endpoint key, using cache‑first semantics.
// It returns the raw response body, metadata about the fetch, and any error.
// Errors are returned only when the API call fails and no stale cache is available.
func (f *Fetcher) Fetch(ctx context.Context, endpointKey string) ([]byte, FetchMeta, error) {
	h := getHandle(endpointKey)
	idx := f.shardIndex(endpointKey)
	shard := f.shards[idx]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// First check in‑memory cache (same session)
	if data, ok := shard.memory[h]; ok {
		meta := shard.memoryMeta[h]
		// Serving from memory cache is considered a cache hit
		meta.CacheHit = true
		return data, meta, nil
	}

	// Try file cache (if enabled)
	var fileEntry cache.CacheEntry
	if f.config.Cache.Enabled {
		entry, _ := f.cache.Get(endpointKey)
		fileEntry = entry
		if entry.Found && !entry.Stale {
			// Fresh cache hit
			ttlRemaining := int(time.Until(entry.FetchedAt.Add(time.Duration(entry.TTLSeconds) * time.Second)).Seconds())
			if ttlRemaining < 0 {
				ttlRemaining = 0
			}
			meta := FetchMeta{
				CacheHit:     true,
				Stale:        false,
				TTLRemaining: ttlRemaining,
				FetchedAt:    entry.FetchedAt,
			}
			// Store in memory for subsequent calls
			shard.memory[h] = entry.Data
			shard.memoryMeta[h] = meta
			return entry.Data, meta, nil
		}
		// If stale, keep entry as fallback for later
	}

	// No fresh cache; fetch from API
	url, err := f.resolveURL(endpointKey)
	if err != nil {
		// If we have stale cache, serve it as degraded fallback
		if fileEntry.Found && fileEntry.Stale {
			meta := FetchMeta{
				CacheHit:     false,
				Stale:        true,
				TTLRemaining: 0,
				FetchedAt:    fileEntry.FetchedAt,
			}
			shard.memory[h] = fileEntry.Data
			shard.memoryMeta[h] = meta
			return fileEntry.Data, meta, nil
		}
		return nil, FetchMeta{}, fmt.Errorf("resolving URL for %q: %w", endpointKey, err)
	}

	data, err := f.httpClient.Get(ctx, url)
	if err != nil {
		// API call failed; if stale cache exists, serve it as degraded fallback
		if fileEntry.Found && fileEntry.Stale {
			meta := FetchMeta{
				CacheHit:     false,
				Stale:        true,
				TTLRemaining: 0,
				FetchedAt:    fileEntry.FetchedAt,
			}
			shard.memory[h] = fileEntry.Data
			shard.memoryMeta[h] = meta
			return fileEntry.Data, meta, nil
		}
		return nil, FetchMeta{}, fmt.Errorf("fetching %q: %w", endpointKey, err)
	}

	// Successfully fetched from API
	fetchedAt := time.Now().UTC()
	ttl := 0
	if f.config.Cache.Enabled {
		ttl = f.resolveTTL(endpointKey)
		f.cache.Set(endpointKey, data, ttl)
	}
	meta := FetchMeta{
		CacheHit:     false,
		Stale:        false,
		TTLRemaining: ttl,
		FetchedAt:    fetchedAt,
	}
	shard.memory[h] = data
	shard.memoryMeta[h] = meta
	return data, meta, nil
}

// resolveURL returns the API URL for the given endpoint key.
// It uses the provider's API key from configuration where needed.
func (f *Fetcher) resolveURL(endpointKey string) (string, error) {
	// Split provider from key (format "provider.endpoint")
	parts := strings.SplitN(endpointKey, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid endpoint key: %q", endpointKey)
	}
	provider, suffix := parts[0], parts[1]
	_ = suffix // available for future endpoint‑specific logic

	// Retrieve API key for this provider
	apiKey := f.apiKey(provider)

	switch endpointKey {
	case CoinGeckoGlobalMarket:
		return coingecko.GlobalURL(), nil
	case CoinGeckoSPPStablesMarkets:
		return coingecko.StablesMarketsURL(), nil
	case CoinGeckoDerivatives:
		return coingecko.DerivativesURL(apiKey), nil
	case CoinGeckoCoinMarketsBreadth:
		return coingecko.CoinMarketsBreadthURL(250), nil
	case CoinGeckoCoinMarketsMomentum:
		return coingecko.CoinMarketsMomentumURL(250), nil
	case BinanceSpotCVD_BTC_1h:
		return binance.KlinesURL("BTCUSDT", "1h", 1), nil
	case CoinDeskAssetTopList:
		// Placeholder – not yet implemented
		return coindesk.AssetTopListURL(), nil
	case CoinMetricsCommunity:
		// Placeholder – not yet implemented
		return coinmetrics.CommunityURL(), nil
	default:
		return "", fmt.Errorf("no URL resolver for endpoint: %q", endpointKey)
	}
}

// resolveTTL returns the cache TTL (seconds) for the given endpoint key.
// It looks up the TTL map in config.Cache.TTL (keys use underscores instead of dots).
// If not found, returns a default TTL (300 seconds).
func (f *Fetcher) resolveTTL(endpointKey string) int {
	// Convert dots to underscores as viper treats dots as nesting
	key := strings.ReplaceAll(endpointKey, ".", "_")
	if ttl, ok := f.config.Cache.TTL[key]; ok {
		// Validate TTL bounds
		if ttl < 0 {
			return 300 // default
		}
		if ttl == 0 {
			return 60 // minimum reasonable TTL
		}
		if ttl > 86400 {
			return 86400 // cap at 1 day
		}
		return ttl
	}
	// Default TTL: 5 minutes
	return 300
}

// apiKey returns the API key for the given provider (e.g., "coingecko").
func (f *Fetcher) apiKey(provider string) string {
	switch provider {
	case "coingecko":
		return f.config.APIs.CoinGecko.APIKey
	case "binance":
		return f.config.APIs.Binance.APIKey
	default:
		return ""
	}
}
