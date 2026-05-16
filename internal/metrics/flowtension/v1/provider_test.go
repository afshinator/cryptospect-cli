package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// ── JSON fixtures ──

// makeKlinesFixture returns a valid Binance klines JSON response for BTC/USDT 1h.
// takerBuyVol and totalVol are string-encoded floats to match Binance API format.
// numTrades controls the thin-candle guard.
func makeKlinesFixture(takerBuyVol, totalVol string, numTrades int) json.RawMessage {
	raw := [][]any{{
		float64(1777500000000), // openTime
		"70000.0",              // open
		"71000.0",              // high
		"69000.0",              // low
		"70500.0",              // close
		totalVol,               // volume (string float)
		float64(1777503600000), // closeTime
		"1000000.0",            // quoteAssetVolume
		float64(numTrades),     // numberOfTrades
		takerBuyVol,            // takerBuyBaseVol (string float)
		"500000.0",             // takerBuyQuoteVol
		"0",                    // ignore
	}}
	b, _ := json.Marshal(raw)
	return b
}

// cgDerivativesFixture returns mock CoinGecko /derivatives data with the given
// total OI, a funding rate (float64, in decimal form), and exchangeCount entries.
func cgDerivativesFixture(totalOI float64, fundingRate float64, exchangeCount int) json.RawMessage {
	entries := make([]any, exchangeCount)
	for i := range entries {
		oi := totalOI / float64(exchangeCount)
		entries[i] = map[string]any{
			"market":        "Exchange",
			"symbol":        "BTCUSDT",
			"index_id":      "BTC",
			"contract_type": "perpetual",
			"funding_rate":  fundingRate,
			"open_interest": oi,
		}
	}
	b, _ := json.Marshal(entries)
	return b
}

// dataMap builds the input data map for provider tests.
func dataMap(binance, cg json.RawMessage) map[string]json.RawMessage {
	m := make(map[string]json.RawMessage)
	if binance != nil {
		m[api.BinanceSpotCVD_BTC_1h] = binance
	}
	if cg != nil {
		m[api.CoinGeckoDerivatives] = cg
	}
	return m
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
	if len(def.Endpoints) != 2 {
		t.Errorf("Endpoints len = %d, want 2 (binance + coingecko)", len(def.Endpoints))
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

// ── Compute: failure paths ──

func TestCompute_MissingBinanceData_Unavailable(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(nil, cgDerivativesFixture(18e9, 0.001, 10)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestCompute_NilData_Unavailable(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

func TestCompute_EmptyData_Unavailable(t *testing.T) {
	p := &Provider{}
	result, err := p.Compute(context.Background(), map[string]json.RawMessage{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable", result.Status)
	}
}

// ── Compute: happy paths ──

func TestCompute_FullSignals_Ok(t *testing.T) {
	// Binance: strong buyer aggression (70% taker buy)
	// CG: $18B OI across 10 exchanges, 0.23% funding (positive)
	binance := makeKlinesFixture("70.0", "100.0", 50)
	cg := cgDerivativesFixture(18e9, 0.0023, 10)

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(binance, cg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok", result.Status)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Signals.CVD.Hook != HookAggressiveBuy {
		t.Errorf("CVD hook = %q, want %q", d.Signals.CVD.Hook, HookAggressiveBuy)
	}
	if d.Signals.FundingRate.Hook != HookPositive {
		t.Errorf("Funding hook = %q, want %q", d.Signals.FundingRate.Hook, HookPositive)
	}
	if d.Signals.OpenInterest.CurrentUSD.Value() == 0 {
		t.Error("OI CurrentUSD must be non-zero")
	}
	if d.Summary == "" {
		t.Error("Summary must not be empty")
	}

	if result.Meta == nil {
		t.Fatal("Meta must not be nil")
	}
	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if m.CvdSampleTrades != 50 {
		t.Errorf("CvdSampleTrades = %d, want 50", m.CvdSampleTrades)
	}
	if m.OIExchangeCount != 10 {
		t.Errorf("OIExchangeCount = %d, want 10", m.OIExchangeCount)
	}
	if m.PrimarySource != "binance_us+coingecko" {
		t.Errorf("PrimarySource = %q, want %q", m.PrimarySource, "binance_us+coingecko")
	}
}

func TestCompute_CgDataMissing_Degraded(t *testing.T) {
	// Only Binance data, no CoinGecko — should be degraded with CVD only
	binance := makeKlinesFixture("30.0", "100.0", 50)

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(binance, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "degraded" {
		t.Errorf("Status = %q, want degraded (CG missing)", result.Status)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Signals.CVD.Hook != HookAggressiveSell {
		t.Errorf("CVD hook = %q, want %q", d.Signals.CVD.Hook, HookAggressiveSell)
	}
	// OI and funding should be zero (no CG data)
	if d.Signals.OpenInterest.CurrentUSD.Value() != 0 {
		t.Errorf("OI current = %g, want 0 (no CG data)", d.Signals.OpenInterest.CurrentUSD.Value())
	}
	if d.Signals.FundingRate.Rate.Value() != 0 {
		t.Errorf("Funding rate = %g, want 0 (no CG data)", d.Signals.FundingRate.Rate.Value())
	}
}

func TestCompute_ThinCandle_StatusStillOk(t *testing.T) {
	// Fewer than 10 trades — CVD hook should be low_confidence, but
	// overall status stays "ok" because CG data is present
	binance := makeKlinesFixture("70.0", "100.0", 3)
	cg := cgDerivativesFixture(18e9, 0.001, 5)

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(binance, cg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok (thin candle is data, not status)", result.Status)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Signals.CVD.Hook != HookLowConfidence {
		t.Errorf("CVD hook = %q, want %q", d.Signals.CVD.Hook, HookLowConfidence)
	}
}

func TestCompute_NeutralFlow(t *testing.T) {
	// Balanced taker flow (50/50), neutral funding
	binance := makeKlinesFixture("50.0", "100.0", 50)
	cg := cgDerivativesFixture(18e9, 0.0001, 5)

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(binance, cg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Status = %q, want ok", result.Status)
	}

	var d Data
	if err := json.Unmarshal(result.Data, &d); err != nil {
		t.Fatalf("unmarshal Data: %v", err)
	}
	if d.Signals.CVD.Hook != HookNeutral {
		t.Errorf("CVD hook = %q, want %q", d.Signals.CVD.Hook, HookNeutral)
	}
	if d.Signals.FundingRate.Hook != HookNeutralFR {
		t.Errorf("Funding hook = %q, want %q", d.Signals.FundingRate.Hook, HookNeutralFR)
	}
	if d.Signals.OpenInterest.Hook != HookOIStable {
		t.Errorf("OI hook = %q, want %q (no cache)", d.Signals.OpenInterest.Hook, HookOIStable)
	}
}

func TestCompute_MetaFieldsPresent(t *testing.T) {
	binance := makeKlinesFixture("55.0", "100.0", 25)
	cg := cgDerivativesFixture(20e9, 0.0015, 8)

	p := &Provider{}
	result, err := p.Compute(context.Background(), dataMap(binance, cg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta == nil {
		t.Fatal("Meta must not be nil")
	}

	var m Meta
	if err := json.Unmarshal(result.Meta, &m); err != nil {
		t.Fatalf("unmarshal Meta: %v", err)
	}
	if m.Instrument != "btc" {
		t.Errorf("Instrument = %q, want btc", m.Instrument)
	}
	if m.Thresholds == nil {
		t.Error("Thresholds must not be nil")
	}
	if m.Description == "" {
		t.Error("Description must not be empty")
	}
}
