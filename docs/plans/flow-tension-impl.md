# flow-tension Implementation Plan

**Metric:** `flow-tension` (alias `ft`)
**Version:** v1.0.0
**Spec:** `docs/metrics/flow-tension.md`
**Created:** 2026-04-30
**Design Change:** All signals now sourced from keyless public APIs (CoinGecko `/derivatives` instead of CoinGecko Pro)

---

## Design Summary

### Key design decision: no API key required

The original spec required CoinGecko Pro for OI and Funding data. After investigation, the public CoinGecko `/derivatives` endpoint (free, keyless) returns both `open_interest` and `funding_rate` per exchange per ticker — 179+ BTC perpetual entries across all major exchanges. This eliminates the need for an API key entirely.

### Endpoint strategy

| Signal | Source | Endpoint | Key | Notes |
|--------|--------|----------|-----|-------|
| CVD | Binance-US (spot) | `/api/v3/klines` BTC/USDT 1h | ❌ Keyless | Already integrated via `internal/api/binance/` |
| OI (aggregated) | CoinGecko (public) | `/derivatives` | ❌ Keyless | Sum `open_interest` across all BTC perpetual entries |
| Funding | CoinGecko (public) | `/derivatives` | ❌ Keyless | Use Binance Futures entry's `funding_rate` |

**OI aggregation:** The `/derivatives` response contains one entry per exchange per ticker. Filter for `symbol` containing "BTC" and `contract_type` == "perpetual", sum `open_interest` across all matching entries. On a typical day this covers 179+ entries with a total OI in the tens-of-billions USD.

**OI 24h change:** Computed against a cached value from the previous successful fetch. The existing file cache stores `flowtension_oi_btc_current_usd` with a timestamp. On first run (no cache), `change_pct_24h` is omitted from output and the OI hook defaults to `"stable"`.

### OI 24h change via cache (cold-start)

```go
// In provider.Compute:
// 1. Read cached OI from file cache keyed "flowtension_oi_btc"
// 2. Compute change: (current - cached) / cached
// 3. Write current OI back to cache for next run
// 4. If no cache entry found → omit change_pct_24h, hook = "stable"
```

The `internal/cache/` package already supports `Get(key)` / `Set(key, data, ttl)` with atomic writes. The cache is per-endpoint-key by default, but can store arbitrary keys. Use a dedicated key like `flowtension_oi_btc` with a TTL of 86400s (24h).

### Status semantics

| Condition | Status | Data contents |
|-----------|--------|---------------|
| Both APIs succeed | `"ok"` | All 3 signals populated |
| Binance CVD fails | `"unavailable"` | Error sentinel |
| CoinGecko fails transiently | `"degraded"` | CVD + null/synthetic OI/funding |
| Thin candle (<10 trades) | `"ok"` | CVD hook = `"low_confidence"`, OI/funding normal |

### Output schema (follows spec doc)

```go
type Data struct {
    Signals Signals `json:"signals"`
    Summary string  `json:"summary"`
}

type Signals struct {
    CVD          SignalCVD `json:"cvd"`
    OpenInterest SignalOI  `json:"open_interest"`
    FundingRate  SignalFR  `json:"funding_rate"`
}

type SignalCVD struct {
    Ratio metrics.MetricFloat `json:"ratio"`
    Hook  string              `json:"hook"`  // "aggressive_buy" | "neutral" | "aggressive_sell" | "low_confidence"
}

type SignalOI struct {
    CurrentUSD  metrics.MetricFloat  `json:"current_usd"`
    ChangePct24h *metrics.MetricFloat `json:"change_pct_24h,omitempty"` // omitted on first run
    Hook        string               `json:"hook"`  // "building" | "stable" | "unwinding"
}

type SignalFR struct {
    Rate metrics.MetricFloat `json:"rate"`
    Hook string              `json:"hook"`  // "overheated" | "positive" | "neutral" | "negative"
}
```

### Thresholds (unchanged from original)

| Threshold | Value | Applies to |
|-----------|-------|------------|
| `minTrades` | 10 | CVD thin-candle guard |
| `flowAggressiveThreshold` | 0.10 | CVD hook boundary |
| `fundingNegativeThreshold` | -0.0003 | Funding hook: ≤ this = negative |
| `fundingPositiveThreshold` | 0.0003 | Funding hook: ≥ this = positive |
| `fundingOverheatedThreshold` | 0.003 | Funding hook: > this = overheated |
| `oiBuildingThreshold` | 0.05 | OI hook: > this = building |
| `oiUnwindingThreshold` | -0.05 | OI hook: < this = unwinding |

### CoinGecko derivatives parser (new function)

Add to `internal/api/coingecko/client.go`:

```go
// ParseDerivativesResponse parses the public /derivatives endpoint response
// and extracts the data needed for flow-tension.
type DerivativesData struct {
    AggregatedOI      float64  // sum of open_interest across all BTC perpetuals
    BinanceFunding    *float64 // funding_rate from Binance Futures entry
    ExchangesCount    int      // number of exchanges contributing to OI
}

func ParseDerivativesResponse(body json.RawMessage) (DerivativesData, error)
```

**Parsing logic:**
1. Unmarshal as `[]json.RawMessage`
2. Iterate entries, find those with `contract_type == "perpetual"` and `symbol` containing "BTC"
3. For each: add `open_interest` to `AggregatedOI`
4. For the entry where `market == "Binance (Futures)"`: capture `funding_rate` as `BinanceFunding`
5. Track `ExchangesCount`

### Narrative hook combinations (unchanged from original)

| Funding | OI | Flow | Summary |
|---------|-----|------|---------|
| `negative_funding` | `oi_building` | any | "Shorts paying longs while leverage builds — early bull phase, sellers exhausted." |
| `overheated_funding` | `oi_building` | any | "Leverage building with overheated longs — elevated liquidation risk." |
| any | `oi_building` | `buyers_aggressive` | "Leverage building with aggressive buying — tension coiling, breakout likely." |
| any | `oi_stable` | `sellers_aggressive` | "Assets staged on exchanges with aggressive selling — supply shock / top warning." |
| any | `oi_unwinding` | any | "Leverage unwinding — deleveraging in progress, likely post-liquidation." |
| `neutral_funding` | `oi_stable` | `flow_neutral` | "Flow tension neutral — no directional conviction." |
| any | any | `low_confidence` | "Thin candle — insufficient trades for reliable CVD signal. Flow direction unreliable." |
| any | `oi_stable` (no cache) | any | Defaults to same text as oi_stable (caveat in meta only, not summary) |

---

## Task List

| Task | Description | Files | Status |
|------|-------------|-------|--------|
| ft-1 | Add `CoinGeckoDerivatives` endpoint constant (already exists in `constants.go`) | — | ✅ done |
| ft-1 | Add `CoinGeckoDerivatives` endpoint constant (already exists in `constants.go`) | — | ✅ done (pre-existing) |
| ft-2 | Add `ParseDerivativesResponse` to CoinGecko client + unit tests | `internal/api/coingecko/client.go`, `internal/api/coingecko/derivatives_test.go` | ✅ done (pre-existing) |
| ft-3 | Create `types.go` — `Data`, `Signals`, `SignalCVD`, `SignalOI`, `SignalFR`, `Meta` structs | `internal/metrics/flowtension/v1/types.go` | ✅ done 2026-04-30 |
| ft-4 | Create `compute.go` — pure compute functions (CVD, OI hook, funding hook, summary) reimplemented from spec | `internal/metrics/flowtension/v1/compute.go` | ✅ done 2026-04-30 |
| ft-5 | Create `compute_test.go` — 30 table-driven tests for all helpers + full Compute | `internal/metrics/flowtension/v1/compute_test.go` | ✅ done 2026-04-30 |
| ft-6 | Update `provider.go` — `Compute()` with fetch-or-use + OI cache logic | `internal/metrics/flowtension/v1/provider.go` | ✅ done 2026-04-30 |
| ft-7 | Rewrite `provider_test.go` — mock data injection + init registration + thin-candle | `internal/metrics/flowtension/v1/provider_test.go` | ✅ done 2026-04-30 |
| ft-8 | Create E2E integration test (4 tests: command, alias, extended, full) | `cmd/cryptospect-cli/flow_tension_e2e_test.go` | ✅ done 2026-04-30 |
| ft-9 | Update `CLAUDE.md` project status | `./CLAUDE.md` | ✅ done 2026-04-30 |

---

## Verification

```
go build ./...
go vet ./...
go test -race -cover ./internal/api/coingecko/...  # ParseDerivativesResponse
go test -race -cover ./internal/metrics/flowtension/...  # provider + compute
go test -race -cover ./cmd/cryptospect-cli/... -run FlowTension  # E2E
goimports -w cmd/ internal/ && gofumpt -w cmd/ internal/
golangci-lint run ./...
```

## Suggested commit sequence

- ft-1+2: "feat: add CoinGecko derivatives parser for flow-tension OI/funding"
- ft-3+4+5: "feat: implement flow-tension compute logic and thresholds"
- ft-6+7+8: "feat: implement flow-tension provider with keyless OI/funding and E2E test"
- ft-9: "docs: update CLAUDE.md with flow-tension status"
