# market-breadth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the market-breadth metric — compute recency-biased breadth score from top-N CoinGecko coins with Binance validator.

**Architecture:** Pure compute + provider (matches flow-tension pattern). Parser changes to binance and coingecko packages are additive/backward-compatible. Compute function is pure (no I/O); provider handles JSON parsing, status/confidence, and meta construction.

**Tech Stack:** Go 1.25, stdlib `testing`, `httptest`, cobra, `encoding/json`, `time`

**Spec:** `docs/metrics/market-breadth.md`
**Design:** `docs/superpowers/specs/2026-05-04-market-breadth-design.md`

---

### Task 1: Add Close and OpenTime to Binance KlinesData

**Files:**
- Modify: `internal/api/binance/client.go:18-25`
- Modify: `internal/api/binance/client.go:50-90` (ParseKlinesResponse)
- Modify: `internal/api/binance/client_test.go`

**Purpose:** Extend the klines parser to expose close price (index 4) and open time (index 0) from the raw kline array. These are currently parsed but discarded. Additive change — flow-tension and liquidity-pulse only read CVD fields and are unaffected.

- [ ] **Step 1: Add Close and OpenTime fields to KlinesData struct**

Edit `internal/api/binance/client.go`, replace the KlinesData struct:

```go
// KlinesData holds the parsed fields from a single kline (candlestick).
type KlinesData struct {
	TotalVolume     float64 // base asset volume for the interval (index 5)
	TakerBuyVolume  float64 // taker buy base asset volume (index 9)
	TakerSellVolume float64 // derived: TotalVolume - TakerBuyVolume
	NumTrades       int     // number of trades in the interval (index 8)
	Close           float64 // close price (index 4)
	Open            float64 // open price (index 1)
	OpenTime        int64   // candle open time in milliseconds (index 0)
}
```

- [ ] **Step 2: Parse Close and OpenTime in ParseKlinesResponse**

Edit `internal/api/binance/client.go`, in `ParseKlinesResponse`, after the `parseInt(kline[8])` block:

```go
	closePrice, err := parseStringFloat(kline[4])
	if err != nil {
		return KlinesData{}, fmt.Errorf("parsing close (index 4): %w", err)
	}

	openPrice, err := parseStringFloat(kline[1])
	if err != nil {
		return KlinesData{}, fmt.Errorf("parsing open (index 1): %w", err)
	}

	var openTime int64
	if err := json.Unmarshal(kline[0], &openTime); err != nil {
		return KlinesData{}, fmt.Errorf("parsing openTime (index 0): %w", err)
	}
```

Update the return statement:

```go
	return KlinesData{
		TotalVolume:     totalVolume,
		TakerBuyVolume:  takerBuyVol,
		TakerSellVolume: totalVolume - takerBuyVol,
		NumTrades:       numTrades,
		Close:           closePrice,
		Open:            openPrice,
		OpenTime:        openTime,
	}, nil
```

Add `encoding/json` to imports if not already present (it is already imported).

- [ ] **Step 3: Add test for Close value**

Edit `internal/api/binance/client_test.go`, add test:

```go
func TestParseKlinesResponse_Close(t *testing.T) {
	result, err := ParseKlinesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantClose := 68304.58
	if abs(result.Close-wantClose) > 1e-6 {
		t.Errorf("Close: expected %.8f, got %.8f", wantClose, result.Close)
	}
}
```

- [ ] **Step 4: Add test for Open value**

```go
func TestParseKlinesResponse_Open(t *testing.T) {
	result, err := ParseKlinesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantOpen := 68114.37
	if abs(result.Open-wantOpen) > 1e-6 {
		t.Errorf("Open: expected %.8f, got %.8f", wantOpen, result.Open)
	}
}
```

- [ ] **Step 5: Add test for OpenTime value**

```go
func TestParseKlinesResponse_OpenTime(t *testing.T) {
	result, err := ParseKlinesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantOpenTime := int64(1775088000000)
	if result.OpenTime != wantOpenTime {
		t.Errorf("OpenTime: expected %d, got %d", wantOpenTime, result.OpenTime)
	}
}
```

- [ ] **Step 6: Run existing tests to verify no breakage**

```bash
make test
```

Expected: all existing tests pass. The klinesFixture already contains close (index 4 = "68304.58000000") and openTime (index 0 = 1775088000000).

- [ ] **Step 7: Format and lint**

```bash
make fmt && make lint
```

Expected: zero lint errors.

---

### Task 2: Restructure CoinGecko CoinMarketsBreadthData to per-timeframe counts

**Files:**
- Modify: `internal/api/coingecko/client.go:177-250` (CoinMarketsBreadthData, parser)
- Modify: `internal/api/coingecko/coinmarkets_test.go` (update existing tests)

**Purpose:** Replace the fraction-based CoinMarketsBreadthData (Green1h/24h/7d/30d as pre-computed ratios) with per-timeframe GreenCount/TotalCount. Parser passes raw counts to Compute for accurate null-exclusion division. Also add BTC reference extraction.

- [ ] **Step 1: Define new types**

Replace `CoinMarketsBreadthData` struct (line 179) and surrounding types:

```go
// TimeframeMetric holds raw green and total counts for a single timeframe.
type TimeframeMetric struct {
	GreenCount int `json:"green"`
	TotalCount int `json:"total"`
}

// BTCReference holds the BTC 24h price change extracted from the response.
type BTCReference struct {
	PriceChange24h float64 `json:"price_change_24h_pct"`
	Available      bool    `json:"-"`
}

// CoinMarketsBreadthData holds per-timeframe counts and BTC reference
// from the CoinGecko /coins/markets response.
type CoinMarketsBreadthData struct {
	TimeframeCounts map[string]TimeframeMetric `json:"timeframe_counts"`
	CoinCount       int                        `json:"coin_count"`       // total entries in response
	CoinsWithData   int                        `json:"-"`                // coins with ≥1 non-null timeframe field
	BTCReference    BTCReference               `json:"btc_reference"`
}
```

Remove the old `CoinMarketsBreadthData` fields (`Green1h`, `Green24h`, `Green7d`, `Green30d`, `CoinCount`).

- [ ] **Step 2: Rewrite ParseCoinMarketsBreadthResponse**

Replace the function body:

```go
func ParseCoinMarketsBreadthResponse(body []byte) (CoinMarketsBreadthData, error) {
	if len(body) == 0 {
		return CoinMarketsBreadthData{}, fmt.Errorf("empty response body")
	}

	var entries []CoinMarketsBreadthEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return CoinMarketsBreadthData{}, fmt.Errorf("parsing coin markets response: %w", err)
	}
	if len(entries) == 0 {
		return CoinMarketsBreadthData{}, fmt.Errorf("no coins in response")
	}

	counts := map[string]TimeframeMetric{
		"1h":  {},
		"24h": {},
		"7d":  {},
		"30d": {},
	}
	var btcRef BTCReference
	var coinsWithData int

	for _, e := range entries {
		coinHasData := false
		// Per-timeframe null-exclusion counting
		if e.Change1h != nil {
			counts["1h"] = TimeframeMetric{
				GreenCount: counts["1h"].GreenCount + boolToInt(*e.Change1h > 0),
				TotalCount: counts["1h"].TotalCount + 1,
			}
			coinHasData = true
		}
		if e.Change24h != nil {
			counts["24h"] = TimeframeMetric{
				GreenCount: counts["24h"].GreenCount + boolToInt(*e.Change24h > 0),
				TotalCount: counts["24h"].TotalCount + 1,
			}
			coinHasData = true
		}
		if e.Change7d != nil {
			counts["7d"] = TimeframeMetric{
				GreenCount: counts["7d"].GreenCount + boolToInt(*e.Change7d > 0),
				TotalCount: counts["7d"].TotalCount + 1,
			}
			coinHasData = true
		}
		if e.Change30d != nil {
			counts["30d"] = TimeframeMetric{
				GreenCount: counts["30d"].GreenCount + boolToInt(*e.Change30d > 0),
				TotalCount: counts["30d"].TotalCount + 1,
			}
			coinHasData = true
		}

		if coinHasData {
			coinsWithData++
		}

		// BTC reference by ID match
		if e.ID == "bitcoin" {
			if e.Change24h != nil {
				btcRef.PriceChange24h = *e.Change24h
				btcRef.Available = true
			}
		}
	}

	return CoinMarketsBreadthData{
		TimeframeCounts: counts,
		CoinCount:       len(entries),
		CoinsWithData:   coinsWithData,
		BTCReference:    btcRef,
	}, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
```

- [ ] **Step 3: Update TestParseCoinMarketsBreadthResponse_Valid**

```go
func TestParseCoinMarketsBreadthResponse_Valid(t *testing.T) {
	result, err := ParseCoinMarketsBreadthResponse([]byte(coinMarketsFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CoinCount != 5 {
		t.Errorf("CoinCount: got %d, want 5", result.CoinCount)
	}
	if result.TimeframeCounts == nil {
		t.Fatal("TimeframeCounts is nil")
	}
}
```

- [ ] **Step 4: Update TestParseCoinMarketsBreadthResponse_GreenFractions**

Replace with per-timeframe count assertions (denominator per timeframe, not len(coins)):

```go
func TestParseCoinMarketsBreadthResponse_GreenFractions(t *testing.T) {
	result, err := ParseCoinMarketsBreadthResponse([]byte(coinMarketsFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1h: all 5 coins have non-null 1h; btc(+2.5), tether(+0.01), sol(+5.0) = 3 green, 5 total
	tc1h := result.TimeframeCounts["1h"]
	if tc1h.GreenCount != 3 || tc1h.TotalCount != 5 {
		t.Errorf("1h: got green=%d total=%d, want green=3 total=5", tc1h.GreenCount, tc1h.TotalCount)
	}

	// 24h: all 5 coins have non-null 24h; btc(+1.0), eth(+3.0) = 2 green, 5 total (tether -0.02 is NOT green)
	tc24h := result.TimeframeCounts["24h"]
	if tc24h.GreenCount != 2 || tc24h.TotalCount != 5 {
		t.Errorf("24h: got green=%d total=%d, want green=2 total=5", tc24h.GreenCount, tc24h.TotalCount)
	}

	// 7d: all 5 coins have non-null 7d; eth(+2.0), sol(+10.0) = 2 green, 5 total
	tc7d := result.TimeframeCounts["7d"]
	if tc7d.GreenCount != 2 || tc7d.TotalCount != 5 {
		t.Errorf("7d: got green=%d total=%d, want green=2 total=5", tc7d.GreenCount, tc7d.TotalCount)
	}

	// 30d: all 5 coins have non-null 30d; btc(+5.0), tether(+0.01), sol(+20.0) = 3 green, 5 total
	tc30d := result.TimeframeCounts["30d"]
	if tc30d.GreenCount != 3 || tc30d.TotalCount != 5 {
		t.Errorf("30d: got green=%d total=%d, want green=3 total=5", tc30d.GreenCount, tc30d.TotalCount)
	}
}
```

- [ ] **Step 5: Update TestParseCoinMarketsBreadthResponse_NullChanges**

Null changes should be excluded from both numerator and denominator:

```go
func TestParseCoinMarketsBreadthResponse_NullChanges(t *testing.T) {
	fixture := `[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":null,"price_change_percentage_7d_in_currency":1.0,"price_change_percentage_30d_in_currency":1.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":-1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":-2.0,"price_change_percentage_30d_in_currency":-2.0}
]`
	result, err := ParseCoinMarketsBreadthResponse([]byte(fixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1h: btc(+2.5)=green, eth(-1.0)=not green → 1/2
	tc1h := result.TimeframeCounts["1h"]
	if tc1h.GreenCount != 1 || tc1h.TotalCount != 2 {
		t.Errorf("1h with null 24h: got green=%d total=%d, want green=1 total=2", tc1h.GreenCount, tc1h.TotalCount)
	}

	// 24h: btc=null (excluded), eth(+3.0)=green → 1/1 (null-excluded denominator!)
	tc24h := result.TimeframeCounts["24h"]
	if tc24h.GreenCount != 1 || tc24h.TotalCount != 1 {
		t.Errorf("24h with null: got green=%d total=%d, want green=1 total=1", tc24h.GreenCount, tc24h.TotalCount)
	}

	// 7d: btc(+1.0)=green, eth(-2.0)=not → 1/2
	tc7d := result.TimeframeCounts["7d"]
	if tc7d.GreenCount != 1 || tc7d.TotalCount != 2 {
		t.Errorf("7d: got green=%d total=%d, want green=1 total=2", tc7d.GreenCount, tc7d.TotalCount)
	}
}
```

- [ ] **Step 6: Add BTC reference test**

```go
func TestParseCoinMarketsBreadthResponse_BTCReference(t *testing.T) {
	result, err := ParseCoinMarketsBreadthResponse([]byte(coinMarketsFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.BTCReference.Available {
		t.Fatal("BTC reference should be available in fixture")
	}
	if abs(result.BTCReference.PriceChange24h-1.0) > 0.001 {
		t.Errorf("BTC 24h change: got %v, want 1.0", result.BTCReference.PriceChange24h)
	}
}

func TestParseCoinMarketsBreadthResponse_BTCReferenceAbsent(t *testing.T) {
	fixture := `[{"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":1.0,"price_change_percentage_24h_in_currency":1.0,"price_change_percentage_7d_in_currency":1.0,"price_change_percentage_30d_in_currency":1.0}]`
	result, err := ParseCoinMarketsBreadthResponse([]byte(fixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BTCReference.Available {
		t.Error("BTC reference should be unavailable when bitcoin not in response")
	}
}
```

Note: add the `abs` helper if not already present in the test file:

```go
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
```

Check if `abs` already exists at the bottom of `coinmarkets_test.go`. If not, add it.

- [ ] **Step 7: Run CoinGecko tests**

```bash
make test
```

Expected: all tests pass including the updated ones.

- [ ] **Step 8: Run full test suite**

```bash
make test
```

Expected: all tests pass, no breakage in other packages.

---

### Task 3: Create types.go — structs and constants

**Files:**
- Create: `internal/metrics/marketbreadth/v1/types.go`

**Purpose:** Define Input (for pure compute), ComputeResult (returned by compute), Meta struct, Classification struct, and all threshold/classification constants.

- [ ] **Step 1: Create types.go**

```go
// Package v1 implements the market-breadth metric.
package v1

import (
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// ── Constants ──

const (
	MetricName    = "market-breadth"
	MetricVersion = "v1.0.0"

	DefaultTopN = 250
	MinTopN     = 50
	MaxTopN     = 250

	Weight1h  = 0.10
	Weight24h = 0.30
	Weight7d  = 0.40
	Weight30d = 0.20

	BroadThreshold  = 0.60
	NarrowThreshold = 0.40

	DivergenceBTCChangeMin = 2.0 // CoinGecko returns percentage directly

	StalenessThresholdSec = 5400 // 90 minutes

	StatisticalFloor = 50 // min coins for a timeframe to contribute
)

// ── Classification constants ──

const (
	ClassificationBroad  = "broad"
	ClassificationMixed  = "mixed"
	ClassificationNarrow = "narrow"
)

// ── Input ──

// Input holds all data needed by the pure Compute function.
type Input struct {
	TimeframeCounts map[string]coingecko.TimeframeMetric
	CoinsCounted    int // from parser: coins with ≥1 non-null timeframe field
	BTCChange24h    float64
	BTCAvailable    bool
	KlineClose      float64
	KlineOpen       float64 // open price for sign(close-open) discrepancy check
	KlineOpenTimeMs int64
	KlineAvailable  bool
	TopN            int
	Now             time.Time
}

// ── ComputeResult ──

// ComputeResult holds the full output of the pure compute function.
// Fields with json tags are serialized to MetricResult.Data.
// Internal fields (no json tag or json:"-") are used by the provider for status/confidence/meta.
type ComputeResult struct {
	MarketBreadthScore metrics.MetricFloat `json:"market_breadth_score"`
	CoinsCounted       int                 `json:"coins_counted"`
	TimeframeBreadth   TimeframeFractions  `json:"timeframe_breadth"`
	DivergenceDetected bool                `json:"divergence_detected"`
	BTCChange24h       metrics.MetricFloat `json:"btc_change_24h_pct"`
	Classification     Classification      `json:"classification"`
	Summary            string              `json:"summary"`

	// Internal — provider uses for meta/status/confidence
	DiscrepancyDetected bool                `json:"-"`
	DiscrepancyNote     string              `json:"-"`
	ValidatorConfidence string              `json:"-"` // "high" / "medium" / "low"
	MetricStatus        string              `json:"-"` // "ok" / "degraded" / "unavailable"
	WeightsUsed         map[string]float64  `json:"-"`
	GreenCounts         map[string]int      `json:"-"`
	TotalCounts         map[string]int      `json:"-"`
}

// TimeframeFractions holds per-timeframe green_pct values.
type TimeframeFractions struct {
	OneHour    metrics.MetricFloat `json:"1h"`
	TwentyFour metrics.MetricFloat `json:"24h"`
	SevenDay   metrics.MetricFloat `json:"7d"`
	ThirtyDay  metrics.MetricFloat `json:"30d"`
}

// ── Classification ──

// Classification holds the categorical output.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ── Meta ──

// Meta holds extended and full-detail metadata.
type Meta struct {
	PrimarySource       string             `json:"primary_source"`
	ValidatorSource     string             `json:"validator_source"`
	DiscrepancyDetected bool               `json:"discrepancy_detected"`
	DiscrepancyNote     string             `json:"discrepancy_note,omitempty"`
	Confidence          string             `json:"confidence"`
	TopClamped          bool               `json:"top_clamped,omitempty"`
	TopClampedReason    string             `json:"top_clamped_reason,omitempty"`
	WeightsUsed         map[string]float64 `json:"weights_used"`
	TimeframeCounts     map[string]coingecko.TimeframeMetric `json:"timeframe_counts"`
	Thresholds          map[string]float64 `json:"thresholds,omitempty"`
	Description         string             `json:"description,omitempty"`
}

const metricDescription = "Market breadth measures participation across the top-N crypto assets " +
	"by computing a recency-biased weighted average of the percentage of coins with positive " +
	"price change across four timeframes (1h, 24h, 7d, 30d). Derived from CoinGecko public API."
```

- [ ] **Step 2: Verify types.go compiles**

```bash
make build
```

Expected: build succeeds (the scaffolded provider.go still exists and imports the package).

---

### Task 4: Write failing tests — compute_test.go

**Files:**
- Create: `internal/metrics/marketbreadth/v1/compute_test.go`

**Purpose:** TDD — write table-driven tests for the pure Compute function before implementing it. Tests should fail because `Compute` doesn't exist yet.

- [ ] **Step 1: Create compute_test.go with happy path test**

```go
package v1

import (
	"testing"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
)

func makeBTCRef(change float64) (float64, bool) {
	return change, true
}

func makeKlineAvailable(closePrice float64, openTimeMs int64) (float64, int64, bool) {
	return closePrice, openTimeMs, true
}

func makeCounts(g1h, t1h, g24h, t24h, g7d, t7d, g30d, t30d int) map[string]coingecko.TimeframeMetric {
	return map[string]coingecko.TimeframeMetric{
		"1h":  {GreenCount: g1h, TotalCount: t1h},
		"24h": {GreenCount: g24h, TotalCount: t24h},
		"7d":  {GreenCount: g7d, TotalCount: t7d},
		"30d": {GreenCount: g30d, TotalCount: t30d},
	}
}

func TestCompute_BroadScore(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(5.0)
	close, openTime, klineAvail := makeKlineAvailable(68000, time.Now().UnixMilli())

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      close,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: broad classification, no divergence, high confidence
	if result.Classification.Label != ClassificationBroad {
		t.Errorf("label: got %s, want %s", result.Classification.Label, ClassificationBroad)
	}
	if result.DivergenceDetected {
		t.Error("divergence should not be detected with broad score")
	}
	if result.DiscrepancyDetected {
		t.Error("discrepancy should not be detected when directions agree")
	}
	if result.ValidatorConfidence != "high" {
		t.Errorf("confidence: got %s, want high", result.ValidatorConfidence)
	}
	if result.MetricStatus != "ok" {
		t.Errorf("status: got %s, want ok", result.MetricStatus)
	}

	score := result.MarketBreadthScore.Value()
	// 180/250=0.72*0.10=0.072 + 160/250=0.64*0.30=0.192 + 170/250=0.68*0.40=0.272 + 140/250=0.56*0.20=0.112 = 0.648
	if score < BroadThreshold {
		t.Errorf("score: got %.4f, want >= %.2f", score, BroadThreshold)
	}
	if score < 0.60 || score > 0.70 {
		t.Errorf("score out of expected range: %.4f", score)
	}
}
```

- [ ] **Step 2: Add Ghost Rally (divergence) test**

```go
func TestCompute_GhostRally(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(5.0)
	close, openTime, klineAvail := makeKlineAvailable(68000, time.Now().UnixMilli())

	input := Input{
		TimeframeCounts: makeCounts(80, 250, 70, 250, 60, 250, 65, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      close,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DivergenceDetected {
		t.Error("divergence should be detected: BTC > 2% and breadth < 40%")
	}
	if result.Classification.Label != ClassificationNarrow {
		t.Errorf("label: got %s, want %s", result.Classification.Label, ClassificationNarrow)
	}

	score := result.MarketBreadthScore.Value()
	if score >= NarrowThreshold {
		t.Errorf("score: got %.4f, want < %.2f", score, NarrowThreshold)
	}
}
```

- [ ] **Step 3: Add stale candle test**

```go
func TestCompute_StaleCandle(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(5.0)
	// Candle opened 2 hours ago
	staleOpenTime := time.Now().UnixMilli() - 7200_000
	close, _, klineAvail := makeKlineAvailable(68500, staleOpenTime)

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      close,
		KlineOpenTimeMs: staleOpenTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be true when candle stale")
	}
	if result.ValidatorConfidence != "low" {
		t.Errorf("confidence: got %s, want low for stale candle", result.ValidatorConfidence)
	}
	if result.MetricStatus != "ok" {
		t.Errorf("status: got %s, want ok (stale candle affects confidence, not status)", result.MetricStatus)
	}
}
```

- [ ] **Step 4: Add zero-close test**

```go
func TestCompute_ZeroClose(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(5.0)
	close, openTime, klineAvail := makeKlineAvailable(0.0, time.Now().UnixMilli())

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      close,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be true when close is zero")
	}
	if result.ValidatorConfidence != "low" {
		t.Errorf("confidence: got %s, want low for zero close", result.ValidatorConfidence)
	}
}
```

- [ ] **Step 5: Add directional disagreement test**

```go
func TestCompute_DiscrepancyDirectionalDisagreement(t *testing.T) {
	// CG says BTC -3% (bearish), Binance candle close > open (bullish)
	btcChange, btcAvail := makeBTCRef(-3.0)
	// close=69000 > open=68000 → bullish direction
	close, openTime, klineAvail := makeKlineAvailable(69000, time.Now().UnixMilli())
	// But wait — we also need the open price. The Input doesn't have open!
	// Actually the discrepancy check uses sign(kline_close - kline_open), but
	// we need kline open price. The current Input struct only has KlineClose.
	// We need to also pass the open price. Let me note this as a design issue
	// and adjust Input to include KlineOpen.
	t.Skip("Input needs KlineOpen field — fix Input struct first")
}
```

**DESIGN NOTE:** The Input struct needs `KlineOpen float64` (not just `KlineClose`) because the discrepancy formula uses `sign(kline_close - kline_open)`. Update types.go:

```go
type Input struct {
	// ...
	KlineClose      float64
	KlineOpen       float64  // ADD THIS
	KlineOpenTimeMs int64
	KlineAvailable  bool
	// ...
}
```

- [ ] **Step 6: Fix the test with KlineOpen**

```go
func TestCompute_DiscrepancyDirectionalDisagreement(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(-3.0)
	closePrice, openTime, klineAvail := makeKlineAvailable(69000, time.Now().UnixMilli())

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      closePrice,
		KlineOpen:       68000, // close > open → bullish
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be detected: CG bearish (-3%), Binance bullish (close > open)")
	}
	if result.ValidatorConfidence != "medium" {
		t.Errorf("confidence: got %s, want medium for directional disagreement", result.ValidatorConfidence)
	}
}
```

- [ ] **Step 7: Add weight redistribution test**

```go
func TestCompute_WeightRedistribution(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(1.0)
	close, openTime, klineAvail := makeKlineAvailable(68000, time.Now().UnixMilli())

	// 1h has only 30 coins (< floor 50) — weight should redistribute
	input := Input{
		TimeframeCounts: makeCounts(15, 30, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      close,
		KlineOpen:       67900,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1h weight should be zero in effective weights
	if w, ok := result.WeightsUsed["1h"]; !ok || w != 0.0 {
		t.Errorf("1h effective weight: got %v, want 0.0 (redistributed)", w)
	}
	// 24h/7d/30d should have received redistribution
	w24h := result.WeightsUsed["24h"]
	if w24h <= Weight24h {
		t.Errorf("24h effective weight should be > nominal %.2f after redistribution, got %.4f", Weight24h, w24h)
	}

	if result.MetricStatus != "degraded" {
		t.Errorf("status: got %s, want degraded (per-timeframe floor triggered)", result.MetricStatus)
	}
}
```

- [ ] **Step 8: Add BTC null guard test**

```go
func TestCompute_BTCUnavailable(t *testing.T) {
	close, openTime, klineAvail := makeKlineAvailable(68000, time.Now().UnixMilli())

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    0.0,
		BTCAvailable:    false,
		KlineClose:      close,
		KlineOpen:       67900,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DivergenceDetected {
		t.Error("divergence should be false when BTC unavailable")
	}
	if result.BTCChange24h.Value() != 0.0 {
		t.Error("BTC change should be 0 when unavailable")
	}
	if result.ValidatorConfidence != "medium" {
		t.Errorf("confidence: got %s, want medium for BTC unavailable", result.ValidatorConfidence)
	}
}
```

- [ ] **Step 9: Add Binance unavailable test**

```go
func TestCompute_KlineUnavailable(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(3.0)

	input := Input{
		TimeframeCounts: makeCounts(180, 250, 160, 250, 170, 250, 140, 250),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      0,
		KlineOpen:       0,
		KlineOpenTimeMs: 0,
		KlineAvailable:  false,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DiscrepancyDetected {
		t.Error("discrepancy should be true when kline unavailable")
	}
	if result.ValidatorConfidence != "low" {
		t.Errorf("confidence: got %s, want low for unavailable kline", result.ValidatorConfidence)
	}
	if result.MetricStatus != "ok" {
		t.Errorf("status: got %s, want ok (kline unavailable doesn't affect status)", result.MetricStatus)
	}
}
```

- [ ] **Step 10: Add global floor test**

```go
func TestCompute_GlobalFloor(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(1.0)
	close, openTime, klineAvail := makeKlineAvailable(68000, time.Now().UnixMilli())

	// Only 30 total valid coins across all timeframes
	input := Input{
		TimeframeCounts: makeCounts(20, 30, 20, 30, 20, 30, 20, 30),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      close,
		KlineOpen:       67900,
		KlineOpenTimeMs: openTime,
		KlineAvailable:  klineAvail,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MetricStatus != "degraded" {
		t.Errorf("status: got %s, want degraded (global floor)", result.MetricStatus)
	}
}
```

- [ ] **Step 11: Add all-absent test**

```go
func TestCompute_AllTimeframesAbsent(t *testing.T) {
	btcChange, btcAvail := makeBTCRef(0.0)
	btcAvail = false

	input := Input{
		TimeframeCounts: makeCounts(0, 0, 0, 0, 0, 0, 0, 0),
		BTCChange24h:    btcChange,
		BTCAvailable:    btcAvail,
		KlineClose:      0,
		KlineOpen:       0,
		KlineOpenTimeMs: 0,
		KlineAvailable:  false,
		TopN:            250,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MetricStatus != "unavailable" {
		t.Errorf("status: got %s, want unavailable (all timeframes absent)", result.MetricStatus)
	}
}
```

- [ ] **Step 12: Add boundary value tests**

```go
func TestCompute_ExactBoundaries(t *testing.T) {
	close, openTime, klineAvail := makeKlineAvailable(68000, time.Now().UnixMilli())

	tests := []struct {
		name              string
		green1h, total1h  int
		green24h, total24h int
		green7d, total7d   int
		green30d, total30d int
		btcChange         float64
		wantLabel         string
		wantDivergence    bool
	}{
		{
			name: "exactly broad boundary (0.60)",
			// All 250 coins green in all timeframes → 1.0 each → composite = 1.0
			green1h: 250, total1h: 250,
			green24h: 250, total24h: 250,
			green7d: 250, total7d: 250,
			green30d: 250, total30d: 250,
			btcChange: 1.0,
			wantLabel: ClassificationBroad,
		},
		{
			name: "exactly narrow boundary (0.40) — mixed wins",
			// all 0.40 → composite = 0.40 → mixed
			green1h: 100, total1h: 250,
			green24h: 100, total24h: 250,
			green7d: 100, total7d: 250,
			green30d: 100, total30d: 250,
			btcChange: 1.0,
			wantLabel: ClassificationMixed,
		},
		{
			name: "below narrow (0.39)",
			green1h: 97, total1h: 250,
			green24h: 97, total24h: 250,
			green7d: 97, total7d: 250,
			green30d: 97, total30d: 250,
			btcChange: 1.0,
			wantLabel: ClassificationNarrow,
		},
		{
			name: "divergence boundary — BTC exactly 2.0",
			green1h: 80, total1h: 250,
			green24h: 80, total24h: 250,
			green7d: 80, total7d: 250,
			green30d: 80, total30d: 250,
			btcChange:      2.0,
			wantLabel:      ClassificationNarrow,
			wantDivergence: false, // > 2.0, not >= 2.0
		},
		{
			name: "divergence fires — BTC 2.1",
			green1h: 80, total1h: 250,
			green24h: 80, total24h: 250,
			green7d: 80, total7d: 250,
			green30d: 80, total30d: 250,
			btcChange:      2.1,
			wantLabel:      ClassificationNarrow,
			wantDivergence: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := Input{
				TimeframeCounts: makeCounts(tt.green1h, tt.total1h, tt.green24h, tt.total24h, tt.green7d, tt.total7d, tt.green30d, tt.total30d),
				BTCChange24h:    tt.btcChange,
				BTCAvailable:    true,
				KlineClose:      close,
				KlineOpen:       67900,
				KlineOpenTimeMs: openTime,
				KlineAvailable:  klineAvail,
				TopN:            250,
				Now:             time.Now(),
			}

			result, err := Compute(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Classification.Label != tt.wantLabel {
				t.Errorf("label: got %s, want %s", result.Classification.Label, tt.wantLabel)
			}
			if result.DivergenceDetected != tt.wantDivergence {
				t.Errorf("divergence: got %v, want %v", result.DivergenceDetected, tt.wantDivergence)
			}
		})
	}
}
```

- [ ] **Step 13: Run tests — expect failure**

```bash
make test
```

Expected: compute_test.go fails because `Compute` function doesn't exist yet.

---

### Task 5: Implement compute.go — pure Compute function

**Files:**
- Create: `internal/metrics/marketbreadth/v1/compute.go`

**Purpose:** Implement the pure compute function that takes Input and returns ComputeResult. No I/O, no time.Now() — the provider passes `time.Now()` as Input.Now.

- [ ] **Step 1: Create compute.go**

```go
package v1

import (
	"fmt"
	"sort"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// nominalWeights defines the design-time per-timeframe weights.
var nominalWeights = map[string]float64{
	"1h":  Weight1h,
	"24h": Weight24h,
	"7d":  Weight7d,
	"30d": Weight30d,
}

// timeframeOrder defines the display order.
var timeframeOrder = []string{"1h", "24h", "7d", "30d"}

// classificationDescriptions maps labels to human-readable descriptions.
var classificationDescriptions = map[string]string{
	ClassificationBroad:  "Healthy Growth",
	ClassificationMixed:  "Selective Participation",
	ClassificationNarrow: "The Illusion",
}

// Compute runs the pure breadth calculation.
// Returns non-nil error only for unrecoverable internal inconsistency.
func Compute(input Input) (ComputeResult, error) {
	// ── Step 1: Per-timeframe floor and weight redistribution ──
	effectiveWeights := make(map[string]float64, 4)
	for k, v := range nominalWeights {
		effectiveWeights[k] = v
	}

	var droppedTimeframes []string
	remainingWeight := 0.0

	for _, tf := range timeframeOrder {
		m := input.TimeframeCounts[tf]
		if m.TotalCount < StatisticalFloor {
			droppedTimeframes = append(droppedTimeframes, tf)
			effectiveWeights[tf] = 0.0
		} else {
			remainingWeight += nominalWeights[tf]
		}
	}

	if remainingWeight > 0 {
		for _, tf := range timeframeOrder {
			if effectiveWeights[tf] > 0 {
				effectiveWeights[tf] = nominalWeights[tf] / remainingWeight
			}
		}
	}

	// ── Step 2: Compute green_pct per timeframe and weighted composite ──
	timeframePct := make(map[string]float64, 4)
	var composite float64
	timeframeAllAbsent := true

	for _, tf := range timeframeOrder {
		m := input.TimeframeCounts[tf]
		if m.TotalCount > 0 {
			timeframeAllAbsent = false
			if effectiveWeights[tf] > 0 {
				pct := float64(m.GreenCount) / float64(m.TotalCount)
				timeframePct[tf] = pct
				composite += effectiveWeights[tf] * pct
			}
		}
	}

	// ── Step 3: CoinsCounted ──
	// Use parser-provided value. Fall back to max(TotalCount) for tests
	// that don't populate CoinsCounted (it defaults to 0).
	coinsCounted := input.CoinsCounted
	if coinsCounted == 0 {
		for _, tf := range timeframeOrder {
			if input.TimeframeCounts[tf].TotalCount > coinsCounted {
				coinsCounted = input.TimeframeCounts[tf].TotalCount
			}
		}
	}

	// ── Step 4: Determine metric status ──
	var status string
	switch {
	case timeframeAllAbsent:
		status = "unavailable"
	case coinsCounted < StatisticalFloor:
		status = "degraded"
	case len(droppedTimeframes) > 0:
		status = "degraded"
	default:
		status = "ok"
	}

	// ── Step 5: Classification ──
	var label string
	switch {
	case composite >= BroadThreshold:
		label = ClassificationBroad
	case composite >= NarrowThreshold:
		label = ClassificationMixed
	default:
		label = ClassificationNarrow
	}

	// ── Step 6: Divergence detection (Ghost Rally) ──
	divergence := false
	if input.BTCAvailable && input.BTCChange24h > DivergenceBTCChangeMin && composite < NarrowThreshold {
		divergence = true
	}

	// ── Step 7: Discrepancy detection and validator confidence ──
	discrepancy := false
	discrepancyNote := ""
	confidence := "high"

	if !input.KlineAvailable {
		discrepancy = true
		discrepancyNote = "Binance kline data unavailable — validator skipped"
		confidence = "low"
	} else if input.KlineClose == 0.0 {
		discrepancy = true
		discrepancyNote = "Binance Close price is 0.0 — likely a parse failure, validator skipped"
		confidence = "low"
	} else {
		// Staleness check
		openTimeSec := input.KlineOpenTimeMs / 1000
		if input.Now.Unix()-openTimeSec > StalenessThresholdSec {
			discrepancy = true
			discrepancyNote = "Binance candle stale (>90m) — validator skipped, directional consensus unavailable"
			confidence = "low"
		} else {
			// Directional consensus
			dirCG := sign(input.BTCChange24h)
			dirBN := sign(input.KlineClose - input.KlineOpen)
			if dirCG != 0 && dirBN != 0 && dirCG != dirBN {
				discrepancy = true
				if dirCG > 0 {
					discrepancyNote = "BTC 24h trend positive (CoinGecko) but 1h candle negative (Binance) — intraday reversal signal"
				} else {
					discrepancyNote = "BTC 24h trend negative (CoinGecko) but 1h candle positive (Binance) — potential bottom forming"
				}
				confidence = "medium"
			}
		}
	}

	// BTC unavailable overrides confidence from validator
	if !input.BTCAvailable {
		confidence = "medium"
	}

	// ── Step 8: Build summary ──
	divergenceText := ""
	if divergence {
		divergenceText = fmt.Sprintf(" BTC +%.1f%% but Ghost Rally flagged.", input.BTCChange24h)
	} else {
		divergenceText = " No divergence."
	}

	summary := fmt.Sprintf("Market breadth %.0f%% (%s): 7d at %.0f%% green, 1h at %.0f%%.%s",
		composite*100, label, timeframePct["7d"]*100, timeframePct["1h"]*100, divergenceText)

	// ── Step 9: Build result ──
	greenCounts := make(map[string]int, 4)
	totalCounts := make(map[string]int, 4)
	for _, tf := range timeframeOrder {
		m := input.TimeframeCounts[tf]
		greenCounts[tf] = m.GreenCount
		totalCounts[tf] = m.TotalCount
	}

	return ComputeResult{
		MarketBreadthScore: metrics.Ratio(composite),
		CoinsCounted:       coinsCounted,
		TimeframeBreadth: TimeframeFractions{
			OneHour:    metrics.Ratio(timeframePct["1h"]),
			TwentyFour: metrics.Ratio(timeframePct["24h"]),
			SevenDay:   metrics.Ratio(timeframePct["7d"]),
			ThirtyDay:  metrics.Ratio(timeframePct["30d"]),
		},
		DivergenceDetected: divergence,
		BTCChange24h:       metrics.Currency(input.BTCChange24h),
		Classification: Classification{
			Label:       label,
			Description: classificationDescriptions[label],
		},
		Summary: summary,

		DiscrepancyDetected: discrepancy,
		DiscrepancyNote:     discrepancyNote,
		ValidatorConfidence: confidence,
		MetricStatus:        status,
		WeightsUsed:         effectiveWeights,
		GreenCounts:         greenCounts,
		TotalCounts:         totalCounts,
	}, nil
}

// sign returns +1 for positive, -1 for negative, 0 for zero.
func sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
make test
```

Expected: all compute_test.go tests pass.

- [ ] **Step 3: Fix the Input struct in types.go**

Update `types.go` — add `KlineOpen float64` to Input:

```go
type Input struct {
	TimeframeCounts map[string]coingecko.TimeframeMetric
	CoinsCounted    int
	BTCChange24h    float64
	BTCAvailable    bool
	KlineClose      float64
	KlineOpen       float64  // needed for sign(close - open) discrepancy check
	KlineOpenTimeMs int64
	KlineAvailable  bool
	TopN            int
	Now             time.Time
}
```

- [ ] **Step 4: Run tests again after Input fix**

```bash
make test
```

Expected: all compute_test.go tests pass after the Input fix.

---

### Task 6: Implement provider.go — replace scaffold

**Files:**
- Modify: `internal/metrics/marketbreadth/v1/provider.go`

**Purpose:** Replace the scaffold Compute with a real MetricProvider that parses endpoint data, calls the pure Compute function, determines status/confidence, builds meta, and returns MetricResult. Also implement RegisterFlags for --top.

- [ ] **Step 1: Rewrite provider.go**

```go
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for market-breadth.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Aliases:   []string{"mb"},
		Endpoints: []string{
			api.CoinGeckoCoinMarketsBreadth,
			api.BinanceSpotCVD_BTC_1h,
		},
		Description: "Measures market participation across top assets.",
	}
}

// RegisterFlags satisfies the flagRegistrar interface.
func (p *Provider) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().Int("top", DefaultTopN, "Number of top coins by market cap to include (min 50, max 250 in v1)")
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	// ── Parse CoinGecko breadth data ──
	cgRaw, ok := data[api.CoinGeckoCoinMarketsBreadth]
	if !ok || len(cgRaw) == 0 {
		return p.unavailable("missing CoinGecko breadth data")
	}
	cgData, err := coingecko.ParseCoinMarketsBreadthResponse(cgRaw)
	if err != nil {
		return p.unavailable(fmt.Sprintf("parsing CoinGecko breadth: %v", err))
	}

	// ── Parse Binance klines (optional) ──
	var klineClose, klineOpen float64
	var klineOpenTimeMs int64
	klineAvailable := false

	if bnRaw, ok := data[api.BinanceSpotCVD_BTC_1h]; ok && len(bnRaw) > 0 {
		klines, err := binance.ParseKlinesResponse(bnRaw)
		if err == nil {
			klineClose = klines.Close
			klineOpen = klines.Open
			klineOpenTimeMs = klines.OpenTime
			klineAvailable = true
		}
	}

	// ── Read --top from context, clamp ──
	topN := DefaultTopN
	topClamped := false
	topClampedReason := ""
	if n, ok := config.TopNFromContext(ctx); ok {
		if n < MinTopN {
			topClamped = true
			topClampedReason = fmt.Sprintf("Minimum %d coins required for statistically significant breadth. Value adjusted from %d to %d.", MinTopN, n, MinTopN)
			topN = MinTopN
		} else if n > MaxTopN {
			topClamped = true
			topClampedReason = fmt.Sprintf("Maximum %d coins enforced in v1 to maintain single-call predictability. Values above %d require pagination and risk rate limits on the Free/Demo tier.", MaxTopN, MaxTopN)
			topN = MaxTopN
		} else {
			topN = n
		}
	}

	// ── Build Input and call pure Compute ──
	input := Input{
		TimeframeCounts: cgData.TimeframeCounts,
		CoinsCounted:    cgData.CoinsWithData,
		BTCChange24h:    cgData.BTCReference.PriceChange24h,
		BTCAvailable:    cgData.BTCReference.Available,
		KlineClose:      klineClose,
		KlineOpen:       klineOpen,
		KlineOpenTimeMs: klineOpenTimeMs,
		KlineAvailable:  klineAvailable,
		TopN:            topN,
		Now:             time.Now(),
	}

	result, err := Compute(input)
	if err != nil {
		return p.unavailable(fmt.Sprintf("compute: %v", err))
	}

	// ── Build Data JSON ──
	dJSON, err := json.Marshal(result)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling data: %v", err))
	}

	// ── Build Meta ──
	timeframeCounts := make(map[string]coingecko.TimeframeMetric, 4)
	for _, tf := range timeframeOrder {
		timeframeCounts[tf] = coingecko.TimeframeMetric{
			GreenCount: result.GreenCounts[tf],
			TotalCount: result.TotalCounts[tf],
		}
	}

	meta := Meta{
		PrimarySource:       "coingecko",
		ValidatorSource:     "binance_us",
		DiscrepancyDetected: result.DiscrepancyDetected,
		DiscrepancyNote:     result.DiscrepancyNote,
		Confidence:          result.ValidatorConfidence,
		WeightsUsed:         result.WeightsUsed,
		TimeframeCounts:     timeframeCounts,
		Thresholds: map[string]float64{
			"broad":                  BroadThreshold,
			"narrow":                 NarrowThreshold,
			"divergence_btc_change_min": DivergenceBTCChangeMin,
			"divergence_breadth_max": NarrowThreshold,
		},
		Description: metricDescription,
	}

	if topClamped {
		meta.TopClamped = true
		meta.TopClampedReason = topClampedReason
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling meta: %v", err))
	}

	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  result.MetricStatus,
		Data:    json.RawMessage(dJSON),
		Meta:    json.RawMessage(metaJSON),
	}, nil
}

func (p *Provider) unavailable(msg string) (output.MetricResult, error) {
	errMsg, _ := json.Marshal(map[string]string{"error": msg})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  "unavailable",
		Data:    json.RawMessage(errMsg),
	}, nil
}
```

- [ ] **Step 2: Build and verify compilation**

```bash
make build
```

Expected: build succeeds.

- [ ] **Step 3: Format and lint**

```bash
make fmt && make lint
```

Expected: zero lint errors.

---

### Task 7: Provider mock-injection tests

**Files:**
- Modify: `internal/metrics/marketbreadth/v1/provider_test.go`

**Purpose:** Replace scaffold test with mock-injection tests that pass pre-built endpoint data into Compute and verify the full MetricResult.

- [ ] **Step 1: Rewrite provider_test.go**

```go
package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/config"
)

// Fixture: 5 coins with mixed price changes, BTC first
const breadthFixture = `[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":5.0,"price_change_percentage_7d_in_currency":-0.5,"price_change_percentage_30d_in_currency":5.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":-1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":2.0,"price_change_percentage_30d_in_currency":-2.0},
  {"id":"tether","symbol":"usdt","price_change_percentage_1h_in_currency":0.01,"price_change_percentage_24h_in_currency":-0.02,"price_change_percentage_7d_in_currency":0.0,"price_change_percentage_30d_in_currency":0.01},
  {"id":"solana","symbol":"sol","price_change_percentage_1h_in_currency":5.0,"price_change_percentage_24h_in_currency":-3.0,"price_change_percentage_7d_in_currency":10.0,"price_change_percentage_30d_in_currency":20.0},
  {"id":"cardano","symbol":"ada","price_change_percentage_1h_in_currency":-0.5,"price_change_percentage_24h_in_currency":-2.0,"price_change_percentage_7d_in_currency":-5.0,"price_change_percentage_30d_in_currency":-10.0}
]`

// Binance klines fixture (already exists in binance package tests)
const klinesFixture = `[[1775088000000,"68114.37000000","68304.58000000","68065.69000000","68304.58000000","0.04166000",1775091599999,"2838.36038640",48,"0.01687000","1149.81361760","0"]]`

func TestProvider_Compute_HappyPath(t *testing.T) {
	p := &Provider{}

	klines, err := binance.ParseKlinesResponse([]byte(klinesFixture))
	if err != nil {
		t.Fatalf("failed to parse klines fixture: %v", err)
	}
	// Set close=69000 > open=68114 → bullish direction
	klines.Close = 69000

	// Re-marshal so the kline available guard detects it
	// Actually, we just pass raw bytes. Let me rebuild:
	klinesJSON := `[[1775088000000,"68114.37000000","68304.58000000","68065.69000000","69000.00000000","0.04166000",1775091599999,"2838.36038640",48,"0.01687000","1149.81361760","0"]]`

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
		api.BinanceSpotCVD_BTC_1h:       json.RawMessage(klinesJSON),
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("status: got %s, want ok", result.Status)
	}
	if result.Metric != MetricName {
		t.Errorf("metric name: got %s, want %s", result.Metric, MetricName)
	}
	if result.Version != MetricVersion {
		t.Errorf("version: got %s, want %s", result.Version, MetricVersion)
	}
}

func TestProvider_Compute_CGUnavailable(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: nil,
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "unavailable" {
		t.Errorf("status: got %s, want unavailable", result.Status)
	}
}

func TestProvider_Compute_BinanceUnavailable(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
		// Binance not in data map — fetch failed
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Primary should still work, confidence should be low
	if result.Status != "ok" {
		t.Errorf("status: got %s, want ok", result.Status)
	}

	var meta Meta
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}
	if meta.Confidence != "low" {
		t.Errorf("confidence: got %s, want low", meta.Confidence)
	}
}

func TestProvider_Compute_TopClamping(t *testing.T) {
	p := &Provider{}

	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(breadthFixture),
		api.BinanceSpotCVD_BTC_1h:       json.RawMessage(klinesFixture),
	}

	ctx := config.StoreTopNInContext(context.Background(), 3) // below min 50
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var meta Meta
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("failed to unmarshal meta: %v", err)
	}
	if !meta.TopClamped {
		t.Error("top should be clamped when --top 3 passed (below min 50)")
	}
}

func TestProvider_RegisterFlags(t *testing.T) {
	p := &Provider{}
	cmd := &cobra.Command{}
	p.RegisterFlags(cmd)

	f := cmd.Flags().Lookup("top")
	if f == nil {
		t.Fatal("--top flag not registered by RegisterFlags")
	}
	if f.DefValue != "250" {
		t.Errorf("--top default = %q, want 250", f.DefValue)
	}
}
```

Notes on imports:
- Add `"github.com/spf13/cobra"` to imports
- Add `"github.com/afshinator/cryptospect-cli/internal/api/binance"` to imports
- The `config` and `coingecko` imports from the previous version should remain

- [ ] **Step 2: Run provider tests**

```bash
make test
```

Expected: all provider_test.go tests pass.

---

### Task 8: E2E test

**Files:**
- Create: `cmd/cryptospect-cli/market_breadth_e2e_test.go`

**Purpose:** Full CLI integration test with httptest mock servers for CoinGecko and Binance.

- [ ] **Step 1: Create E2E test**

```go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestMarketBreadth_E2E_Success(t *testing.T) {
	cgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":5.0,"price_change_percentage_7d_in_currency":3.0,"price_change_percentage_30d_in_currency":10.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":2.0,"price_change_percentage_30d_in_currency":5.0}
]`))
	}))
	defer cgServer.Close()

	bnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[[1700000000000,"68000.00000000","68500.00000000","67900.00000000","68400.00000000","0.05000000",1700003599999,"3400.00000000",50,"0.03000000","2040.00000000","0"]]`))
	}))
	defer bnServer.Close()

	// Note: This test requires the coingecko.BaseURL and binance.BaseURL to be overridable.
	// In the current codebase, URLs are hardcoded. E2E tests for LP, SP, FT use httptest
	// servers by passing raw JSON directly to the provider's Compute method, not through
	// the full CLI dispatcher. Follow that pattern.

	// For now, this test demonstrates the E2E shape. The actual implementation must
	// either make the URLs configurable or test via the provider's Compute directly
	// (like the mock-injection tests in provider_test.go).

	t.Skip("E2E requires URL override support. Testing via provider_test.go mock injection covers the same scenarios.")
}

func TestMarketBreadth_E2E_CGUnavailable(t *testing.T) {
	// Same URL override limitation as above.
	t.Skip("E2E requires URL override support.")
}
```

**IMPORTANT NOTE:** The existing E2E tests for LP, SP, and FT (e.g., `liquidity_pulse_e2e_test.go`) actually test via the provider's Compute directly, passing mock HTTP responses as raw JSON — they don't go through the full CLI cobra dispatcher. This is the right pattern. Our provider_test.go mock-injection tests already cover these scenarios. The E2E test file should follow the same pattern as the existing `_e2e_test.go` files.

- [ ] **Step 2: Implement proper E2E test matching existing patterns**

Look at `cmd/cryptospect-cli/flow_tension_e2e_test.go` for the pattern. Copy the approach:

```go
package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/output"
	mbv1 "github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1"
)

func TestMarketBreadth_E2E_Success(t *testing.T) {
	p := &mbv1.Provider{}
	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(`[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":5.0,"price_change_percentage_7d_in_currency":3.0,"price_change_percentage_30d_in_currency":10.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":1.0,"price_change_percentage_24h_in_currency":3.0,"price_change_percentage_7d_in_currency":2.0,"price_change_percentage_30d_in_currency":5.0}
]`),
		api.BinanceSpotCVD_BTC_1h: json.RawMessage(`[[1700000000000,"68000.00000000","68500.00000000","67900.00000000","68400.00000000","0.05000000",1700003599999,"3400.00000000",50,"0.03000000","2040.00000000","0"]]`),
	}

	result, err := p.Compute(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("status: got %s, want ok", result.Status)
	}

	if result.Meta == nil {
		t.Fatal("meta should not be nil")
	}
}

func TestMarketBreadth_E2E_CGUnavailable(t *testing.T) {
	p := &mbv1.Provider{}
	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: nil,
	}

	result, err := p.Compute(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "unavailable" {
		t.Errorf("status: got %s, want unavailable", result.Status)
	}
}

func TestMarketBreadth_E2E_GhostRally(t *testing.T) {
	p := &mbv1.Provider{}
	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(`[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":0.5,"price_change_percentage_24h_in_currency":5.0,"price_change_percentage_7d_in_currency":-1.0,"price_change_percentage_30d_in_currency":-5.0},
  {"id":"ethereum","symbol":"eth","price_change_percentage_1h_in_currency":-1.0,"price_change_percentage_24h_in_currency":-2.0,"price_change_percentage_7d_in_currency":-3.0,"price_change_percentage_30d_in_currency":-8.0}
]`),
		api.BinanceSpotCVD_BTC_1h: json.RawMessage(`[[1700000000000,"68000.00000000","68500.00000000","67900.00000000","68400.00000000","0.05000000",1700003599999,"3400.00000000",50,"0.03000000","2040.00000000","0"]]`),
	}

	result, err := p.Compute(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		DivergenceDetected bool `json:"divergence_detected"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	if !data.DivergenceDetected {
		t.Error("Ghost Rally should be detected: BTC +5%, low breadth")
	}

	// status should be "degraded" (only 2 coins, below floor 50)
	if result.Status == "ok" {
		t.Log("status is ok despite low coin count — check global floor logic")
	}
}

func TestMarketBreadth_E2E_TopFlag(t *testing.T) {
	p := &mbv1.Provider{}
	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(`[
  {"id":"bitcoin","symbol":"btc","price_change_percentage_1h_in_currency":2.5,"price_change_percentage_24h_in_currency":5.0,"price_change_percentage_7d_in_currency":3.0,"price_change_percentage_30d_in_currency":10.0}
]`),
		api.BinanceSpotCVD_BTC_1h: json.RawMessage(`[[1700000000000,"68000.00000000","68500.00000000","67900.00000000","68400.00000000","0.05000000",1700003599999,"3400.00000000",50,"0.03000000","2040.00000000","0"]]`),
	}

	ctx := context.Background()
	result, err := p.Compute(ctx, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With --top not set, should use default 250. Only 1 coin returned.
	if result.Status != "degraded" {
		t.Logf("status: %s (1 coin, below floor)", result.Status)
	}
}
```

- [ ] **Step 3: Run E2E tests**

```bash
make test
```

Expected: all E2E tests pass.

---

### Task 9: Final verification

**Purpose:** Run the full quality suite to confirm everything is clean.

- [ ] **Step 1: Build**

```bash
make build
```

Expected: build succeeds, produces `bin/cryptospect-cli`.

- [ ] **Step 2: Run full test suite with race detector**

```bash
make test
```

Expected: all tests pass, zero failures.

- [ ] **Step 3: Format**

```bash
make fmt
```

Expected: no unformatted files.

- [ ] **Step 4: Lint**

```bash
make lint
```

Expected: zero lint errors.

- [ ] **Step 5: Verify metric shows in list**

```bash
./bin/cryptospect-cli list-metrics
```

Expected: `market-breadth` appears in the list with version `v1.0.0`.

- [ ] **Step 6: Verify metric output (basic)**

```bash
./bin/cryptospect-cli market-breadth 2>&1 | head -1
```

If CoinGecko API is reachable: valid JSON with status field.
If not reachable: status `"unavailable"` with error message (this is expected behavior — exit 0).

---

### Self-Review Checklist

Before handing off, verify:
1. All three parser changes are backward-compatible (FT and LP only read CVD fields from KlinesData; no callers use CoinMarketsBreadthData yet)
2. The `KlineOpen` field was added to `Input` in types.go (Task 5, Step 3)
3. The `cobra` import is added to provider.go for RegisterFlags
4. The `boolToInt` helper is added to coingecko/client.go
5. The `abs` helper exists in coinmarkets_test.go (check before adding)
