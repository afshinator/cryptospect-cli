package defillama_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api/defillama"
)

// buildBody encodes a StablecoinsResponse to JSON for use in tests.
func buildBody(t *testing.T, resp defillama.StablecoinsResponse) []byte {
	t.Helper()
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("buildBody: %v", err)
	}
	return b
}

func makeAsset(current, prevWeek float64) defillama.PeggedAsset {
	return defillama.PeggedAsset{
		Circulating:         defillama.Circulating{PeggedUSD: current},
		CirculatingPrevWeek: defillama.Circulating{PeggedUSD: prevWeek},
	}
}

// TestParseStablecoinsResponse covers happy path and error cases.
func TestParseStablecoinsResponse(t *testing.T) {
	t.Parallel()

	t.Run("happy path two assets", func(t *testing.T) {
		t.Parallel()
		resp := defillama.StablecoinsResponse{
			PeggedAssets: []defillama.PeggedAsset{
				makeAsset(100e9, 95e9),
				makeAsset(50e9, 48e9),
			},
		}
		got, err := defillama.ParseStablecoinsResponse(buildBody(t, resp))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.PeggedAssets) != 2 {
			t.Errorf("want 2 assets, got %d", len(got.PeggedAssets))
		}
	})

	t.Run("empty body returns error", func(t *testing.T) {
		t.Parallel()
		_, err := defillama.ParseStablecoinsResponse(nil)
		if err == nil {
			t.Error("want error for nil body, got nil")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		t.Parallel()
		_, err := defillama.ParseStablecoinsResponse([]byte(`{not valid json`))
		if err == nil {
			t.Error("want error for invalid JSON, got nil")
		}
	})

	t.Run("empty peggedAssets slice is valid", func(t *testing.T) {
		t.Parallel()
		resp := defillama.StablecoinsResponse{PeggedAssets: []defillama.PeggedAsset{}}
		got, err := defillama.ParseStablecoinsResponse(buildBody(t, resp))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.PeggedAssets) != 0 {
			t.Errorf("want 0 assets, got %d", len(got.PeggedAssets))
		}
	})
}

// TestAggregateSupply covers normal summing and edge cases.
func TestAggregateSupply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		assets       []defillama.PeggedAsset
		wantCurrent  float64
		wantPrevWeek float64
	}{
		{
			name:         "two assets",
			assets:       []defillama.PeggedAsset{makeAsset(100e9, 95e9), makeAsset(50e9, 48e9)},
			wantCurrent:  150e9,
			wantPrevWeek: 143e9,
		},
		{
			name:         "single asset",
			assets:       []defillama.PeggedAsset{makeAsset(80e9, 75e9)},
			wantCurrent:  80e9,
			wantPrevWeek: 75e9,
		},
		{
			name:         "empty slice",
			assets:       []defillama.PeggedAsset{},
			wantCurrent:  0,
			wantPrevWeek: 0,
		},
		{
			name: "asset with zero prevWeek (e.g. newly listed stablecoin)",
			assets: []defillama.PeggedAsset{
				makeAsset(10e9, 0),
				makeAsset(5e9, 5e9),
			},
			wantCurrent:  15e9,
			wantPrevWeek: 5e9,
		},
		{
			name: "non-USD stablecoin contributes zero peggedUSD",
			assets: []defillama.PeggedAsset{
				makeAsset(100e9, 95e9),
				// EURC-type: peggedUSD is 0 because it uses a different peg field
				makeAsset(0, 0),
			},
			wantCurrent:  100e9,
			wantPrevWeek: 95e9,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := defillama.StablecoinsResponse{PeggedAssets: tc.assets}
			current, prevWeek := defillama.AggregateSupply(resp)
			if math.Abs(current-tc.wantCurrent) > 1 {
				t.Errorf("current: want %.0f, got %.0f", tc.wantCurrent, current)
			}
			if math.Abs(prevWeek-tc.wantPrevWeek) > 1 {
				t.Errorf("prevWeek: want %.0f, got %.0f", tc.wantPrevWeek, prevWeek)
			}
		})
	}
}

// TestClassifyTrend covers all three buckets and boundary values.
func TestClassifyTrend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		current  float64
		prevWeek float64
		want     string
	}{
		{"expanding well above threshold", 110e9, 100e9, "expanding"},      // +10%
		{"expanding at boundary", 101e9, 100e9, "expanding"},               // exactly +1%
		{"stable positive just below threshold", 1009e9, 1000e9, "stable"}, // +0.9%
		{"stable at zero change", 100e9, 100e9, "stable"},                  // 0%
		{"stable negative just above threshold", 991e9, 1000e9, "stable"},  // -0.9%
		{"contracting at boundary", 99e9, 100e9, "contracting"},            // exactly -1%
		{"contracting well below threshold", 80e9, 100e9, "contracting"},   // -20%
		{"prevWeek zero returns stable (guard)", 100e9, 0, "stable"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := defillama.ClassifyTrend(tc.current, tc.prevWeek)
			if got != tc.want {
				t.Errorf("ClassifyTrend(%.0f, %.0f) = %q, want %q",
					tc.current, tc.prevWeek, got, tc.want)
			}
		})
	}
}

// TestStablecoinsURL verifies the URL is well-formed.
func TestStablecoinsURL(t *testing.T) {
	t.Parallel()
	url := defillama.StablecoinsURL()
	if url == "" {
		t.Error("StablecoinsURL returned empty string")
	}
	// Must reference the correct host.
	const wantHost = "stablecoins.llama.fi"
	if !containsStr(url, wantHost) {
		t.Errorf("StablecoinsURL %q does not contain %q", url, wantHost)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
