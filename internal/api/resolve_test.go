package api

import (
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/config"
)

func TestResolveURL(t *testing.T) {
	cfg := config.Config{
		APIs: config.APIsConfig{
			CoinGecko: config.APIKeyConfig{APIKey: "cg-test-key"},
			Binance:   config.APIKeyConfig{APIKey: "binance-test-key"},
		},
	}
	// We need a Fetcher to call resolveURL, but we can create a minimal one.
	// Since resolveURL only uses config and apiKey method, we can create a dummy fetcher.
	f := &Fetcher{config: &cfg}

	tests := []struct {
		endpoint string
		wantErr  bool
	}{
		{CoinGeckoGlobalMarket, false},
		{CoinGeckoSPPStablesMarkets, false},
		{CoinGeckoDerivatives, false},
		{CoinGeckoCoinMarketsBreadth, false},
		{CoinGeckoCoinMarketsMomentum, false},
		{BinanceSpotCVD_BTC_1h, false},
		{CoinDeskAssetTopList, true},
		{CoinMetricsCommunity, true},
		{"unknown.provider", true},
		{"malformed", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got, err := f.resolveURL(tt.endpoint)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveURL(%q) expected error, got nil", tt.endpoint)
				}
				return
			}
			if err != nil {
				t.Errorf("resolveURL(%q) unexpected error: %v", tt.endpoint, err)
				return
			}
			if got == "" {
				t.Errorf("resolveURL(%q) returned empty URL", tt.endpoint)
			}
		})
	}
}

func TestResolveTTL(t *testing.T) {
	cfg := config.Config{
		Cache: config.CacheConfig{
			Enabled: true,
			TTL: map[string]int{
				"coingecko_global_market": 123,
				"binance_spot_cvd_btc_1h": 456,
			},
		},
	}
	f := &Fetcher{config: &cfg}

	tests := []struct {
		endpoint string
		want     int
	}{
		{CoinGeckoGlobalMarket, 123},        // underscore key matches config
		{CoinGeckoSPPStablesMarkets, 300},   // default
		{CoinGeckoDerivatives, 300},         // default
		{CoinGeckoCoinMarketsBreadth, 300},  // default
		{CoinGeckoCoinMarketsMomentum, 300}, // default
		{BinanceSpotCVD_BTC_1h, 456},        // underscore key matches config
		{CoinDeskAssetTopList, 300},         // default
		{CoinMetricsCommunity, 300},         // default
		{"unknown.provider", 300},           // default
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got := f.resolveTTL(tt.endpoint)
			if got != tt.want {
				t.Errorf("resolveTTL(%q) = %d, want %d", tt.endpoint, got, tt.want)
			}
		})
	}
}
