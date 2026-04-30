# stablecoin-power Implementation Plan

**Metric:** `stablecoin-power` (alias `sp`)
**Version:** v1.0.0
**Spec:** `docs/metrics/stablecoin-power.md`
**Created:** 2026-04-29

---

## Design Summary

### Endpoint strategy
- **Primary:** `coingecko.global_market` — total_market_cap_usd
- **Primary:** `coingecko.spp_stables_markets` — per-stablecoin market caps (top N, from hardcoded list)
- **Validator:** `defillama.stablecoins` — aggregate supply cross-check + 7d trend

### DefiLlama API
- URL: `https://stablecoins.llama.fi/stablecoins`
- Response: `{"peggedAssets": [{..., "circulating": {"peggedUSD": float64}, "circulatingPrevWeek": {"peggedUSD": float64}, ...}]}`
- ~488KB response; free, no rate limiting
- Sum `circulating.peggedUSD` across all assets → current total
- Sum `circulatingPrevWeek.peggedUSD` across all assets → 7d-ago total
- `supply_trend_7d`: expanding (>+1%), stable (±1%), contracting (<-1%)

### `--top` flag wiring
- Private `flagRegistrar` interface in `root.go` (type assertion, no cobra dep in metrics)
- Provider implements `RegisterFlags(*cobra.Command)`; dispatcher calls it if available
- Value read from `cmd.Flags()` in `buildMetricRunE`, stored in context via `StoreTopNInContext`
- Provider reads via `TopNFromContext`; clamps to minimum 8

### Stablecoin ID list
- Default: `coingecko.SPPStableIDs` (hardcoded 18-entry list in coingecko package)
- Overridable via config: `metrics.stablecoin_power.stablecoin_ids []string`
- Provider reads from config first, falls back to hardcoded list

### Cross-source discrepancy
- Compare CoinGecko top-N mcap sum vs. DefiLlama aggregate total
- < 15%: confidence `"high"`
- 15–25%: confidence `"medium"`
- ≥ 25%: confidence `"low"`

### Classification descriptions (Low splits on supply_trend_7d)
| Label | supply_trend_7d | Description |
|-------|-----------------|-------------|
| high | any | "Dry Powder Alert" |
| normal | any | "Healthy Balance" |
| low | stable/expanding | "Overextended" |
| low | contracting | "Capital Flight" |

### Meta fields by detail level
- **extended:** primary_source, validator_source, discrepancy_detected, discrepancy_note, confidence, supply_trend_source, stablecoin_scope, top_clamped, top_clamped_reason
- **full:** + thresholds, description, top_n_stablecoins

### Known gap (consistent with liquidity-pulse)
`cache_hit` and `ttl_remaining_sec` are in the spec schema but not yet populated — FetchMeta doesn't flow to Compute. Deferred as a cross-metric infrastructure concern.

---

## Task List

| Task | Description | Status |
|------|-------------|--------|
| sp-1 | Add `DefiLlamaStablecoins` constant to `internal/api/constants.go` | ✅ done 2026-04-29 |
| sp-2 | Create `internal/api/defillama/client.go` + tests (TDD) | ✅ done 2026-04-29 |
| sp-3 | Wire DefiLlama into fetcher (URL resolver + apiKey) | ✅ done 2026-04-29 |
| sp-4 | Config: add `StablecoinPower.StablecoinIDs`; add `StoreTopNInContext`/`TopNFromContext` | ✅ done 2026-04-29 |
| sp-5 | root.go: `flagRegistrar` dispatch + read `--top` in `buildMetricRunE` | ✅ done 2026-04-29 |
| sp-6 | Implement `Compute()` + `RegisterFlags()` in stablecoinpower provider | ✅ done 2026-04-29 |
| sp-7 | E2E integration test (`stablecoin-power_e2e_test.go`) | ✅ done 2026-04-29 |

---

## Verification (per task group)
```
go build ./...
go test -race -cover ./...
golangci-lint run ./...
goimports -w . && gofumpt -w .
```

## Suggested commit sequence
- sp-1+2+3: "feat: add DefiLlama client and wire stablecoins endpoint"
- sp-4+5: "feat: add --top flag dispatch and context helpers for stablecoin-power"
- sp-6+7: "feat: implement stablecoin-power compute and E2E test"
