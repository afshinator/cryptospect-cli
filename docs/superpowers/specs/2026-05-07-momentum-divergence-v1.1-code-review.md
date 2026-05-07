# Code Review — momentum-divergence v1.1 implementation

## Summary

The implementation adds `market_cap` parsing, computes market-cap weighted tier averages, enriches full-detail tier entries with `market_cap` / `weight_pct`, and bumps `momentum-divergence` to `v1.1.0`. The core weighted-average path is straightforward and current tests/lint/build pass, but the documented simple-mean fallback path is not reachable through `Compute()`, which makes the `weighting_method: "simple"` contract misleading.

Verification run during review:

- `go test -race -count=1 ./...` passed: 426 tests
- `golangci-lint run ./...` passed: no issues
- `go build -o ./cryptospect-cli ./cmd/cryptospect-cli/` passed

## Issues

🔴 **Blocking — `internal/metrics/momentumdivergence/v1/compute.go`:26-32, 70-79, 186-198; `internal/metrics/momentumdivergence/v1/provider.go`:113-117**

The documented fallback path to simple means is effectively unreachable through production code. `Compute()` filters out every coin with `MarketCap <= 0` before building `tierCoin`, so `weightedMeanTier()` never receives a present tier with zero total weight. If CoinGecko returns valid ranks and 24h changes but null/zero `market_cap` values, the current behavior is tier absence and potentially `unavailable`, not `weighting_method: "simple"` as the design/spec/docs describe.

Concrete fix: choose one contract and make code/tests/docs match. If the fallback is intended, keep price-valid tier buckets first, determine floor using price-valid coins for the degenerate fallback case, compute a separate market-cap-valid subset for weighted means, and set fallback when a present tier has enough price-valid coins but no usable market caps. If the stricter double-validity behavior is intended, remove `weightedMeanTier()`'s fallback return value, remove `WeightingFallback`, remove `weighting_method: "simple"`, and update docs/tests to say missing market cap makes a coin invalid rather than falling back.

🟡 **Suggestion — `internal/metrics/momentumdivergence/v1/compute.go`:70-79; `internal/metrics/momentumdivergence/v1/provider.go`:113-117**

`WeightingFallback` is only captured from the large tier. Even if fallback semantics are fixed later, a mid-tier or small-tier fallback would be discarded and `weighting_method` would still report `"market_cap"`.

Concrete fix: aggregate fallback across all computed tiers, e.g. `meta.WeightingFallback = largeFallback || midFallback || smallFallback`, or expose a per-tier weighting method if mixed tier methods need to be distinguishable.

🟡 **Suggestion — `internal/metrics/momentumdivergence/v1/types.go`:87-91**

`WeightPct` uses `omitempty`, so a real value that rounds to `0.00` is omitted from `tier_detail`. That makes full-detail output schema unstable and can hide exactly the tiny-weight tail entries that users may want to inspect.

Concrete fix: remove `omitempty` from `WeightPct` so `0` is emitted intentionally. Consider also removing `omitempty` from `MarketCap` if the simple-mean fallback path is kept and full-detail entries need to show zero market caps explicitly.

🟡 **Suggestion — `internal/metrics/momentumdivergence/v1/types.go`:40; `docs/metrics/momentum-divergence.md`:141-142, 312**

Some comments/docs still describe the old data contract. `TierAverages` says it holds the simple mean, and the metric docs still say the required CoinGecko fields are rank and 24h price change, omitting `market_cap`.

Concrete fix: update the comment to "market-cap weighted 24h return" and update docs data-source sections to include `market_cap` as a required field for v1.1.

## Positives

- The parser change is additive and preserves nil `Change24h` behavior while translating missing `market_cap` to `0.0` for downstream handling.
- The weighted-average helper is simple and directly tested with normal, equal-weight, mixed-zero, all-zero, single-coin, and realistic cap-distribution cases.
- Existing tests were preserved by assigning equal market caps to legacy fixtures, which isolates behavior changes to the new weighted-mean tests.
- The e2e coverage now asserts that `weighting_method` is exposed at extended detail.

## Verdict

Needs significant changes before merge. The main weighted path works, but the fallback behavior and `weighting_method` contract need to be resolved so production behavior, tests, and docs agree.
