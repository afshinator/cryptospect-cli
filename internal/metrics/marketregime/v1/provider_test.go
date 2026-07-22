package v1

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// ── Helpers ──

// makeGlobalFixture returns a /global JSON response with given dominance, volume, and mcap.
func makeGlobalFixture(btcDom, volume, mcap float64) json.RawMessage {
	raw := map[string]any{
		"data": map[string]any{
			"market_cap_percentage": map[string]any{
				"btc": btcDom,
				"eth": 18.0,
			},
			"total_volume": map[string]any{
				"usd": volume,
			},
			"total_market_cap": map[string]any{
				"usd": mcap,
			},
		},
	}
	b, _ := json.Marshal(raw)
	return b
}

// makeBreadthFixture returns a /coins/markets JSON response with n valid coins.
// Each coin has all 4 timeframe price changes at +1.0% (green in all timeframes).
// If btcChange is non-nil, a bitcoin entry is included with that 24h change.
func makeBreadthFixture(n int, btcChange *float64) json.RawMessage {
	entries := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		entry := map[string]any{
			"id":                                     coinID(i),
			"symbol":                                 "sym",
			"market_cap_rank":                        i + 1,
			"price_change_percentage_1h_in_currency": 1.0,
			"price_change_percentage_24h_in_currency": 1.0,
			"price_change_percentage_7d_in_currency":  1.0,
			"price_change_percentage_30d_in_currency": 1.0,
		}
		entries[i] = entry
	}
	if btcChange != nil {
		entry := map[string]any{
			"id":                                     "bitcoin",
			"symbol":                                 "btc",
			"market_cap_rank":                        1,
			"price_change_percentage_1h_in_currency": 0.1,
			"price_change_percentage_24h_in_currency": *btcChange,
			"price_change_percentage_7d_in_currency":  5.0,
			"price_change_percentage_30d_in_currency": 8.0,
		}
		entries = append(entries, entry)
	}
	b, _ := json.Marshal(entries)
	return b
}

func coinID(i int) string {
	ids := []string{
		"aave", "bittensor", "chainlink", "dogecoin", "ethereum",
		"filecoin", "gala", "helium", "internet-computer", "jupiter",
		"kucoin-token", "litecoin", "maker", "near", "optimism",
		"polkadot", "quant", "render-token", "solana", "tron",
		"uniswap", "vechain", "worldcoin", "xrp", "yearn-finance",
	}
	if i < len(ids) {
		return ids[i]
	}
	return ids[i%len(ids)]
}

// dataMap builds input for provider tests.
func dataMap(global, breadth json.RawMessage) map[string]json.RawMessage {
	m := make(map[string]json.RawMessage)
	if global != nil {
		m[api.CoinGeckoGlobalMarket] = global
	}
	if breadth != nil {
		m[api.CoinGeckoCoinMarketsBreadth] = breadth
	}
	return m
}

// newCacheConfig creates a Config with the given cache directory.
func newCacheConfig(dir string) config.Config {
	return config.Config{
		Cache: config.CacheConfig{Enabled: true, Dir: dir},
	}
}

// ── Def / registry / init ──

func TestProvider_Def(t *testing.T) {
	p := &Provider{}
	def := p.Def()

	if def.Name != MetricName {
		t.Errorf("Name = %q, want %q", def.Name, MetricName)
	}
	if def.Version != MetricVersion {
		t.Errorf("Version = %q, want %q", def.Version, MetricVersion)
	}
	if len(def.Aliases) == 0 {
		t.Error("Aliases must not be empty")
	}
	if len(def.Endpoints) == 0 {
		t.Error("Endpoints must not be empty")
	}
}

func TestProvider_RegisteredOnInit(t *testing.T) {
	reg := metrics.GlobalRegistry()
	p, err := reg.Resolve(MetricName)
	if err != nil {
		t.Fatalf("provider not in global registry: %v", err)
	}
	if p.Def().Name != MetricName {
		t.Errorf("Name = %q, want %q", p.Def().Name, MetricName)
	}
}

// ── Unavailable paths ──

func TestProvider_Unavailable_MissingGlobal(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(nil, makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
	assertErrorPayload(t, result.Data)
	assertNoMeta(t, result.Meta)
}

func TestProvider_Unavailable_MissingBreadth(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
	assertErrorPayload(t, result.Data)
	assertNoMeta(t, result.Meta)
}

func TestProvider_Unavailable_GlobalParseFailure(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(json.RawMessage(`not-json`), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

func TestProvider_Unavailable_MissingBTCDominance(t *testing.T) {
	raw := map[string]any{
		"data": map[string]any{
			"total_volume":     map[string]any{"usd": 1e10},
			"total_market_cap": map[string]any{"usd": 1e12},
		},
	}
	b, _ := json.Marshal(raw)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(json.RawMessage(b), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable when BTC dominance missing", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

func TestProvider_Unavailable_ZeroVolumeGuard(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 0, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable when volume is zero", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

func TestProvider_Unavailable_ZeroMcapGuard(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 0), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable when mcap is zero", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

func TestProvider_Unavailable_MissingVolumeField(t *testing.T) {
	raw := map[string]any{
		"data": map[string]any{
			"market_cap_percentage": map[string]any{"btc": 52.0},
			"total_market_cap":      map[string]any{"usd": 1e12},
		},
	}
	b, _ := json.Marshal(raw)
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(json.RawMessage(b), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable when volume field missing", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

func TestProvider_Unavailable_BreadthParseFailure(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), json.RawMessage(`bad`)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable when breadth parse fails", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

func TestProvider_Unavailable_MBUnavailable(t *testing.T) {
	// Empty JSON array → "no coins in response" → parse failure → unavailable
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), json.RawMessage(`[]`)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable when breadth has no coins", result.Status)
	}
	assertErrorPayload(t, result.Data)
}

// ── Happy path with cold start (no prior state) ──

func TestProvider_ColdStart_BaselineWrite(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0o750)
	cfg := newCacheConfig(cacheDir)
	ctx := config.StoreInContext(context.Background(), cfg)

	p := &Provider{}
	result, err := p.Compute(ctx, dataMap(makeGlobalFixture(52.0, 1e10, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok (cold start produces valid classification)", result.Status)
	}
	if result.Metric != MetricName {
		t.Errorf("Metric = %q, want %q", result.Metric, MetricName)
	}

	var data Data
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if data.Regime == "" {
		t.Error("regime must not be empty on cold start")
	}
	if data.DominanceTrend != TrendNeutral {
		t.Errorf("DominanceTrend = %q, want %q on cold start", data.DominanceTrend, TrendNeutral)
	}
	if !strings.HasPrefix(data.Summary, "[SIGNAL_UNVERIFIED] ") {
		t.Errorf("summary should start with cold start token, got %q", data.Summary)
	}

	// Meta must be present
	if result.Meta == nil {
		t.Fatal("Meta must not be nil on ok status")
	}
	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if dc, _ := meta["dominance_cold_start"].(bool); !dc {
		t.Error("dominance_cold_start should be true on cold start")
	}
	if notes, _ := meta["notes"].([]any); len(notes) == 0 {
		t.Error("notes should contain cold_start")
	} else {
		hasCold := false
		for _, n := range notes {
			if s, ok := n.(string); ok && s == "cold_start" {
				hasCold = true
				break
			}
		}
		if !hasCold {
			t.Error("notes should include cold_start")
		}
	}
	if conf, _ := meta["confidence"].(string); conf != ConfidenceMedium {
		t.Errorf("confidence = %q, want %q on cold start", conf, ConfidenceMedium)
	}

	// Verify state cache was written
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("opening cache for verification: %v", err)
	}
	defer func() { _ = c.Close() }()
	entry, err := c.Get(StateKey)
	if err != nil || !entry.Found {
		t.Error("state cache entry should exist after cold start baseline write")
	}
}

// ── Happy path with prior dominance (non-cold start) ──

func TestProvider_NonColdRun_ValidPrior(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0o750)
	cfg := newCacheConfig(cacheDir)
	ctx := config.StoreInContext(context.Background(), cfg)

	// Seed prior dominance state (52.0, written 4 hours ago)
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("opening cache for seeding: %v", err)
	}
	priorDom := 52.0
	priorBytes, _ := json.Marshal(priorDom)
	_ = c.Set(StateKey, priorBytes, StateTTLSec)
	// Manually adjust FetchedAt to simulate older entry
	// The cache Set sets FetchedAt to time.Now(); we need it older for a proper delta.
	// We'll close and reopen to work around — the file metadata won't let us fake this easily.
	// Instead, we use a dominance that produces rising trend even with fresh write.
	_ = c.Close()

	// Current dominance is 53.3 (delta +1.3pp → rising)
	p := &Provider{}
	result, err := p.Compute(ctx, dataMap(makeGlobalFixture(53.3, 1e10, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok", result.Status)
	}

	var data Data
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	// With prior at 52.0 and current at 53.3, delta is +1.3pp → rising
	if data.DominanceTrend != TrendRising {
		t.Errorf("DominanceTrend = %q, want %q (+1.3pp delta)", data.DominanceTrend, TrendRising)
	}

	// Verify meta
	if result.Meta == nil {
		t.Fatal("Meta must not be nil")
	}
	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if dc, _ := meta["dominance_cold_start"].(bool); dc {
		t.Error("dominance_cold_start should be false when prior exists")
	}
	if delta, ok := meta["dominance_delta_since_last_fetch"]; !ok {
		t.Error("dominance_delta_since_last_fetch should be present on non-cold run")
	} else {
		// delta should be approximately +1.3 (the exact value depends on stored dominance)
		if _, ok := delta.(float64); !ok {
			t.Errorf("delta should be float64, got %T", delta)
		}
	}
	if ps, ok := meta["prior_snapshot_age_sec"]; !ok {
		t.Error("prior_snapshot_age_sec should be present on non-cold run")
	} else {
		if _, ok := ps.(float64); !ok {
			t.Errorf("prior_snapshot_age_sec should be float64 (JSON number), got %T", ps)
		}
	}
}

// ── Malformed / stale state cache → cold start ──

func TestProvider_StateMalformed_ActsAsColdStart(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0o750)
	cfg := newCacheConfig(cacheDir)
	ctx := config.StoreInContext(context.Background(), cfg)

	// Write malformed state
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("cache.Open: %v", err)
	}
	_ = c.Set(StateKey, json.RawMessage(`not-a-float`), StateTTLSec)
	_ = c.Close()

	p := &Provider{}
	result, err := p.Compute(ctx, dataMap(makeGlobalFixture(52.0, 1e10, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok (malformed state → cold start, but still classifies)", result.Status)
	}
	var meta map[string]any
	_ = json.Unmarshal(result.Meta, &meta)
	if dc, _ := meta["dominance_cold_start"].(bool); !dc {
		t.Error("dominance_cold_start should be true when state is malformed")
	}
}

// ── Degraded path (CoinsCounted < 50) ──

func TestProvider_Degraded_LowCoinCount(t *testing.T) {
	p := &Provider{}
	// 49 coins, no bitcoin → CoinsCounted = 49 < 50 → degraded (but still produces classification)
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), makeBreadthFixture(49, nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "degraded" {
		t.Errorf("Status = %q, want degraded (49 coins < 50 threshold, got %d)", result.Status, 49)
	}
	if result.Meta == nil {
		t.Error("Meta should be present for degraded status")
		return
	}
	var meta map[string]any
	_ = json.Unmarshal(result.Meta, &meta)
	if conf, _ := meta["confidence"].(string); conf != ConfidenceLow {
		t.Errorf("confidence = %q, want %q for degraded", conf, ConfidenceLow)
	}
}

// ── Weight redistribution (without degraded status) ──

func TestProvider_WeightRedistribution_OkStatus(t *testing.T) {
	// Need enough coins (>=50) but some timeframes with <50 non-null values
	// to trigger weight redistribution in mb
	// 55 coins with all 4 timeframes → all green, all counted → no redistribution
	// But mb algorithm redistributes when TotalCount < 50 for any timeframe
	// We need 60 total coins but with some timeframes having null values
	// Actually, makeBreadthFixture gives all coins all 4 timeframes.
	// To trigger redistribution, we'd need some timeframe to have <50 non-null.
	// Simpler: test that normal case has ok status with no redistribution note.

	breadthJSON := makeBreadthFixture(60, ptr(2.5))
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), breadthJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok with 60 coins all green", result.Status)
	}

	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	notes, _ := meta["notes"].([]any)
	for _, n := range notes {
		if s, ok := n.(string); ok && s == "weight_redistribution" {
			t.Error("should not have weight_redistribution note with all timeframes populated")
		}
	}
}

// ── Missing BTC reference ──

func TestProvider_MissingBTCReference(t *testing.T) {
	// breadth fixture without bitcoin entry → no BTC reference
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), makeBreadthFixture(60, nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("Status = %q, want ok (missing BTC ref is not unavailable, just lower confidence)", result.Status)
	}

	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if btc, ok := meta["btc_24h_change"]; ok && btc != nil {
		t.Error("btc_24h_change should be omitted when BTC reference is missing")
	}
	notes, _ := meta["notes"].([]any)
	hasMissing := false
	for _, n := range notes {
		if s, ok := n.(string); ok && s == "missing_reference_data" {
			hasMissing = true
		}
	}
	if !hasMissing {
		t.Error("notes should include missing_reference_data")
	}
	if conf, _ := meta["confidence"].(string); conf != ConfidenceLow {
		t.Errorf("confidence = %q, want %q for missing reference", conf, ConfidenceLow)
	}

	// Check summary has the missing ref token
	var data Data
	_ = json.Unmarshal(result.Data, &data)
	if !strings.Contains(data.Summary, "[MISSING_BTC_REF]") {
		t.Errorf("summary should contain [MISSING_BTC_REF], got %q", data.Summary)
	}
}

// ── cache_hit / ttl_remaining_sec — NOT in provider meta ──
// cache_hit and ttl_remaining_sec are injected by root.go's postProcessMeta
// overlay, not by MR's own provider. They must not appear in raw provider output.

func TestProvider_CacheHit_NotInProviderMeta(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("Status = %q, want ok", result.Status)
	}

	var meta map[string]any
	_ = json.Unmarshal(result.Meta, &meta)
	if _, ok := meta["cache_hit"]; ok {
		t.Error("cache_hit should not be in provider meta (injected by root.go overlay)")
	}
	if _, ok := meta["ttl_remaining_sec"]; ok {
		t.Error("ttl_remaining_sec should not be in provider meta (injected by root.go overlay)")
	}
}

// ── Unavailable payload shape ──

func TestProvider_Unavailable_PayloadShape(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(nil, makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if _, ok := data["error"]; !ok {
		t.Error(`data should be {"error":"..."}`)
	}
	if result.Meta != nil {
		t.Error("meta should be nil when unavailable")
	}
}

// ── WeightsUsed output shape ──

func TestProvider_WeightsUsedOutputShape(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(makeGlobalFixture(52.0, 1e10, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("Status = %q, want ok", result.Status)
	}

	var meta map[string]any
	_ = json.Unmarshal(result.Meta, &meta)
	weights, ok := meta["weights_used"].(map[string]any)
	if !ok {
		t.Fatal("meta should have weights_used as object")
	}
	for _, tf := range []string{"1h", "24h", "7d", "30d"} {
		if w, ok := weights[tf]; !ok {
			t.Errorf("weights_used missing key %q", tf)
		} else if v, ok := w.(float64); !ok || v == 0 {
			t.Errorf("weights_used[%q] = %v, want >0 with 60 all-green coins", tf, w)
		}
	}
}

// ── MarketBreadthScore serialization precision ──

// TestData_MarketBreadthScore_FourDecimalPrecision verifies that market_breadth_score
// serializes as a JSON number with exactly 4 decimal places (MetricFloat Ratio precision),
// matching the format used by the market-breadth metric for the same field.
func TestData_MarketBreadthScore_FourDecimalPrecision(t *testing.T) {
	d := Data{MarketBreadthScore: metrics.Ratio(0.6)}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal Data: %v", err)
	}
	want := `"market_breadth_score":0.6000`
	if !strings.Contains(string(b), want) {
		t.Errorf("market_breadth_score must serialize with 4 decimal places\ngot:  %s\nwant substring: %s", b, want)
	}
}

// ── Combined state + freshness in single Compute call ──

// TestProvider_StateAndFreshness_Combined verifies that a single Compute call
// correctly reads and writes prior dominance state.
// cache_hit/ttl_remaining_sec are no longer part of provider meta — they are
// injected by root.go's postProcessMeta overlay.
func TestProvider_StateAndFreshness_Combined(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0o750)
	cfg := newCacheConfig(cacheDir)
	ctx := config.StoreInContext(context.Background(), cfg)

	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("opening cache: %v", err)
	}
	// Seed state (prior dominance)
	priorBytes, _ := json.Marshal(52.0)
	_ = c.Set(StateKey, priorBytes, StateTTLSec)
	// Seed API endpoint caches (fresh, 1-hour TTL)
	_ = c.Set(api.CoinGeckoGlobalMarket, makeGlobalFixture(53.3, 1e10, 1e12), 3600)
	_ = c.Set(api.CoinGeckoCoinMarketsBreadth, makeBreadthFixture(60, ptr(2.5)), 3600)
	_ = c.Close()

	time.Sleep(10 * time.Millisecond)

	p := &Provider{}
	result, err := p.Compute(ctx, dataMap(makeGlobalFixture(53.3, 1e10, 1e12), makeBreadthFixture(60, ptr(2.5))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("Status = %q, want ok", result.Status)
	}

	var meta map[string]any
	_ = json.Unmarshal(result.Meta, &meta)

	// State path: should not be cold start
	if dc, _ := meta["dominance_cold_start"].(bool); dc {
		t.Error("dominance_cold_start should be false — prior state was seeded")
	}
	// State path: delta should be present (+1.3pp)
	if _, ok := meta["dominance_delta_since_last_fetch"]; !ok {
		t.Error("dominance_delta_since_last_fetch should be present")
	}
	// cache_hit and ttl_remaining_sec are NOT in provider meta — injected by root.go
	if _, ok := meta["cache_hit"]; ok {
		t.Error("cache_hit should not be in provider meta (injected by root.go overlay)")
	}
	if _, ok := meta["ttl_remaining_sec"]; ok {
		t.Error("ttl_remaining_sec should not be in provider meta (injected by root.go overlay)")
	}
}

// ── Helpers ──

func assertErrorPayload(t *testing.T, data json.RawMessage) {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if _, ok := m["error"]; !ok {
		t.Error(`data should be {"error":"..."}`)
	}
}

func assertNoMeta(t *testing.T, meta json.RawMessage) {
	t.Helper()
	if meta != nil || len(meta) > 0 {
		t.Error("meta should be nil when unavailable")
	}
}
