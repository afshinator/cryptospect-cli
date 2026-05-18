# Phase 2: CLI-Level Golden Tests (Deferred)

## Goal
Test the **complete CLI output pipeline** — envelope, meta injection, detail-level filtering, compact vs pretty — using deterministic mocked API servers.

## Why Phase 1 wasn't enough
Provider-level golden tests (Phase 1, completed 2026-05-17) validate `Compute()` output at the data level. They don't cover:
- `CLIResponse` envelope (`status`, `ts`, `results` array)
- Detail-level meta filtering (`--detail basic` strips meta; `extended` strips thresholds/description/top_n_stablecoins/tier_detail)
- Cache meta injection from root.go (`cache_hit`, `ttl_remaining_sec`)
- Compact vs pretty formatting (`--pretty`)
- Flag parsing interactions (`--top`, `--segments`)
- Error mode output shapes (rate limits, API down, partial data)

## Prerequisites

### 1. Time injection into `internal/output/writer.go`
Replace `time.Now().Unix()` with a callable variable:
```go
var timeNow = time.Now
```
Add `SetTimeNow(fn func() time.Time)` test helper (same pattern as `SetWriter`).

### 2. API URL override config fields
Add config fields and env-var overrides for each API base URL:
- `CRYPTOSPECT_COINGECKO_BASE_URL` (default: `https://api.coingecko.com/api/v3`)
- `CRYPTOSPECT_BINANCE_BASE_URL` (default: `https://api.binance.us/api/v3`)
- `CRYPTOSPECT_DEFILLAMA_BASE_URL` (default: `https://coins.llama.fi`)
- `CRYPTOSPECT_COINDESK_BASE_URL` (default: `https://data-api.coindesk.com`)

This requires changes to `internal/config/` and the API clients in `internal/api/`.

### 3. httptest mock servers
Create `cmd/cryptospect-cli/testdata/mock_servers.go` with one handler per API endpoint. Fixture data already exists inline in per-metric test files.

### 4. Golden test infrastructure
Create `cmd/cryptospect-cli/testdata/golden_helpers.go` with:
```go
func RunGoldenTest(t *testing.T, args []string, goldenPath string)
func SetFixedTime(t *testing.T, ts int64)
func MockConfig(servers map[string]*httptest.Server) context.Context
```

## Golden file scope
18 files (6 metrics × 3 detail levels):
```
cmd/cryptospect-cli/testdata/golden/
  liquidity-pulse/basic.golden
  liquidity-pulse/extended.golden
  liquidity-pulse/full.golden
  stablecoin-power/basic.golden
  stablecoin-power/extended.golden
  stablecoin-power/full.golden
  flow-tension/basic.golden
  flow-tension/extended.golden
  flow-tension/full.golden
  market-breadth/basic.golden
  market-breadth/extended.golden
  market-breadth/full.golden
  momentum-divergence/basic.golden
  momentum-divergence/extended.golden
  momentum-divergence/full.golden
  market-regime/basic.golden
  market-regime/extended.golden
  market-regime/full.golden
```

## Existing E2E test patterns to convert
Every E2E test in `cmd/cryptospect-cli/*_e2e_test.go` currently calls `NewRootCommand().Execute()` which hits live APIs. These tests should be converted to use httptest mock servers. The existing `assertCacheFields` helper in `cmd/cryptospect-cli/e2e_helpers_test.go` can be reused.

## Test scenarios to add beyond happy-path
- Rate limit response (HTTP 429) → `CLIError` with `retry_after_sec`
- API unavailable (HTTP 500 / timeout) → metric status `"unavailable"`
- Partial data (one API up, another down) → metric status `"degraded"`
- `--pretty` flag → indented JSON output
- Each detail level at least one boundary case

## Timeline estimate
- Time injection: ~30 min
- API URL override config: ~1-2 hours
- httptest mock servers: ~2-3 hours
- Golden test infrastructure: ~1 hour
- Initial golden file capture: ~30 min
- **Total: ~5-8 hours**

## Dependencies
- Phase 1 golden helpers (`internal/metrics/golden.go`) can be reused as-is for normalization
- The `-update` flag mechanism works the same way
