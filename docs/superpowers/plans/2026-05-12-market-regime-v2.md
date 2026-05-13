# market-regime Implementation Plan (v2)

Authoritative spec: `docs/metrics/market-regime.md`

This v2 plan replaces the previous plan and is aligned to the finalized spec behavior.

## Goal

Implement `market-regime` v1.0.0 in `internal/metrics/marketregime/v1/` using:
- Dominance trend from cached prior BTC dominance state key (`marketregime_dominance_pct`)
- Breadth via direct `marketbreadth/v1.Compute()` integration
- Conviction from `lp_ratio` (`/global` volume / mcap)
- Matrix classification + modifier + confidence/notes model
- Suite-consistent unavailable behavior (`data: {"error":"..."}`, no meta)
- Detail-level meta behavior consistent with `root.go` suppression

## Keep / Replace

- Keep existing scaffold provider registration and metric identity constants.
- Replace scaffold compute/provider logic entirely.
- Add `types.go`, `compute.go`, tests.

## Non-negotiable Spec Constraints

1. Meta is globally suppressed at `--detail basic` by `cmd/cryptospect-cli/root.go`; metric must not special-case this.
2. `status: "unavailable"` returns `data` as `{"error":"<message>"}` and omits `meta`.
3. `mr` status derivation is independent of `mbResult.MetricStatus`:
   - `degraded` iff `mbResult.CoinsCounted < 50`
   - weight redistribution affects notes/confidence, not status.
4. Dominance delta source is state key `marketregime_dominance_pct` (TTL 48h), not endpoint cache replay.
5. `cache_hit` and `ttl_remaining_sec` are derived from explicit cache reads of both endpoint entries:
   - `api.CoinGeckoGlobalMarket`
   - `api.CoinGeckoCoinMarketsBreadth`
6. Summary must emit reliability tokens when conditions apply (all detail levels):
   - `[SIGNAL_UNVERIFIED]` for cold start
   - `[MISSING_BTC_REF]` for missing BTC reference
   - `[BREADTH_PARTIAL]` for weight redistribution

## Files

1. `internal/metrics/marketregime/v1/types.go` (new)
2. `internal/metrics/marketregime/v1/compute.go` (new)
3. `internal/metrics/marketregime/v1/compute_test.go` (new)
4. `internal/metrics/marketregime/v1/provider.go` (replace scaffold)
5. `internal/metrics/marketregime/v1/provider_test.go` (replace scaffold/add full coverage)
6. `cmd/cryptospect-cli/testdata/market_regime_global.json` (new)
7. `cmd/cryptospect-cli/testdata/market_regime_coin_markets.json` (new)
8. `cmd/cryptospect-cli/market_regime_e2e_test.go` (new)

## Implementation Tasks

### Task 1: `types.go`

Define:
- Constants: regime labels, trend/modifier/conviction enums, note keys, thresholds, state key/TTL.
- `Input`:
  - `BTCDominancePct float64`
  - `PriorDominancePct *float64`
  - `PriorSnapshotAgeSec *int`
  - `BreadthScore float64`
  - `BreadthBand string`
  - `BreadthStatus string`
  - `WeightsUsed map[string]float64` (internal)
  - `WeightRedistributed bool`
  - `LPRatio float64`
  - `BTCChange24h *float64`
- `ComputeResult` (internal + data/meta construction inputs).
- `Data`, `Classification`.
- `MetaExtended` and `MetaFull` where `weights_used` is a fixed struct shape (`1h`,`24h`,`7d`,`30d`) for deterministic output.

### Task 2: `compute.go`

Implement pure compute path:
1. Dominance trend:
   - cold start if prior nil
   - rising `>= +0.5`, falling `<= -0.5`, else neutral
2. Conviction:
   - high `> 0.15`, normal `>= 0.07 && <= 0.15`, low `< 0.07`
3. Matrix lookup with Falling+Narrow disambiguation:
   - high => Capitulation, else Structural Decay
4. Modifier from BTC 24h change dead-band ±0.5, with nil => neutral + missing reference condition.
5. Notes:
   - `cold_start`, `weight_redistribution`, `missing_reference_data`
   - capitulation sub-state notes for neutral/positive modifier
6. Confidence (minimum-applicable rule):
   - start high
   - lower for cold start / redistribution / capitulation sub-states / missing reference / degraded
7. Summary generation:
   - conviction-aware branching for Consolidation/Stagnation
   - modifier-aware Capitulation messaging
   - inject reliability tokens exactly as specified.

### Task 3: `provider.go`

Implement provider orchestration:
1. Parse raw inputs:
   - missing/empty/parse failures for either required endpoint => unavailable
2. Parse global:
   - BTC dominance required (nil => unavailable)
   - `total_volume["usd"]` and `total_market_cap["usd"]` required and non-zero (zero-guard => unavailable)
3. Parse coin markets breadth response; parse failure => unavailable.
4. Build `mbv1.Input` with:
   - `CoinsCounted: cgData.CoinsWithData`
   - BTC reference from parsed breadth data
   - `KlineAvailable: false`, zero Binance fields, `TopN: 250`
5. Run `mbv1.Compute()`:
   - if mb unavailable => unavailable
   - derive `BreadthStatus` by `CoinsCounted` threshold only (`<50` => degraded else ok)
   - derive `WeightRedistributed` via `WeightsUsed` vs nominal keys.
6. State cache handling:
   - open cache from config (if unavailable, continue with cold-start-safe behavior)
   - read `marketregime_dominance_pct`
   - compute age from `Entry.FetchedAt`; if negative or >86400 => cold start path
   - parse prior dominance float; malformed => cold start path
   - write current dominance back to state key with TTL 172800
7. Compute cache freshness meta:
   - explicit `Get` on both endpoint keys for `cache_hit` and TTL min logic
8. Call `Compute(input)`.
9. Build output:
   - `status`: ok/degraded from provider-derived breadth status unless unavailable path already taken
   - data JSON from compute result
   - meta JSON only for non-unavailable paths
   - map `mbResult.WeightsUsed` (map) to fixed output `weights_used` struct fields.

### Task 4: Unit tests (`compute_test.go`)

Cover:
- Dominance trend boundaries (`-0.5`, `+0.5`)
- Conviction boundaries (`0.07`, `0.15`)
- Matrix mapping for all cells + capitulation disambiguation
- Modifier boundaries and missing BTC reference
- Confidence precedence with simultaneous conditions
- Summary token injection for cold/missing-ref/redistribution
- Capitulation sub-state notes and confidence behavior

### Task 5: Provider tests (`provider_test.go`)

Cover:
- Unavailable paths:
  - missing endpoint bytes
  - parse failures
  - missing BTC dominance
  - zero-guard triggers
  - mb unavailable
- Cold start (missing state key) + baseline write
- Stale/malformed state key => cold start
- Non-cold run with valid prior and age
- Degraded path from `CoinsCounted < 50`
- Weight redistribution note/confidence without degraded status
- Missing BTC reference => neutral fallback + low confidence + omitted btc_24h_change
- `cache_hit` false/true and `ttl_remaining_sec=min(...)`
- `status: unavailable` payload shape exactly `{"error":"..."}`

### Task 6: E2E test

Add live-style E2E test following existing metric E2E pattern (not httptest server scaffolding):
- basic: data fields present, meta suppressed
- extended/full: meta present for ok/degraded
- unavailable: error payload in data, meta absent

## Verification Checklist

1. `go test ./internal/metrics/marketregime/v1 -v`
2. `go test ./cmd/cryptospect-cli -run MarketRegime -v`
3. `go test ./...`
4. `gofumpt -w ./...`
5. `goimports -w ./...`
6. `golangci-lint run ./...`

## Completion Criteria

- All tests pass.
- Lint/format passes.
- Output behavior matches spec for:
  - unavailable payload shape
  - basic/extended/full detail semantics
  - status independence from `mbResult.MetricStatus`
  - deterministic `weights_used` output shape
  - reliability summary tokens.
