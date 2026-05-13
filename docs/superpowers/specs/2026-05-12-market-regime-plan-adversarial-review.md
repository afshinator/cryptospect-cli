# Adversarial Review — market-regime Implementation Plan (2026-05-12)

Target plan: `docs/superpowers/plans/2026-05-12-market-regime.md`
Target spec: `docs/metrics/market-regime.md`

## Findings (highest risk first)

1. **[Critical] Basic-detail meta contract in plan/spec is impossible with current CLI pipeline**
   - Plan expects `meta` fields at basic detail (`docs/superpowers/plans/2026-05-12-market-regime.md:339-347`, `:800-807`).
   - Spec also says meta is "Always present (basic and above)" (`docs/metrics/market-regime.md:291-313`).
   - Current CLI hard-drops `result.Meta` for `--detail basic` (`cmd/cryptospect-cli/root.go:187-192`).
   - Impact: implementation can pass provider tests but still never emit basic meta from CLI, violating spec and E2E expectations.
   - Fix: either (a) change global detail behavior in `root.go`, or (b) update plan/spec to align with current global contract (basic hides meta for all metrics).

2. **[Critical] Plan’s status/confidence logic conflicts with the shared `marketbreadth` implementation**
   - Plan maps MR status directly from `mbResult.MetricStatus` (`docs/superpowers/plans/2026-05-12-market-regime.md:392`, `:763`) and also says weight redistribution should cap confidence at medium (`:590`).
   - In `marketbreadth`, any dropped timeframe (`len(droppedTimeframes)>0`) sets status to `"degraded"` (`internal/metrics/marketbreadth/v1/compute.go:87-89`).
   - Plan’s confidence logic forces `"low"` whenever degraded (`docs/superpowers/plans/2026-05-12-market-regime.md:540-542`).
   - Impact: weight-redistribution scenarios become low-confidence/degraded, contradicting the plan/spec expectation of medium+ok behavior.
   - Fix: define explicit MR status policy independent of `mbResult.MetricStatus` or accept `mb` semantics and update tests/spec accordingly.

3. **[High] Fixture schema in plan will silently break BTC modifier extraction**
   - Plan fixture requires BTC field `price_change_percentage_24h` (`docs/superpowers/plans/2026-05-12-market-regime.md:919`).
   - Parser used by MR reads `price_change_percentage_24h_in_currency` (`internal/api/coingecko/client.go:206`).
   - Impact: BTC reference appears missing in tests, forcing fallback modifier `neutral`, adding `missing_reference_data`, and collapsing confidence to low.
   - Fix: update plan fixture keys to `*_in_currency` everywhere.

4. **[High] E2E test strategy references non-existent "httptest pattern" and mismatches current architecture**
   - Plan says to follow a hardened pattern with `httptest servers` (`docs/superpowers/plans/2026-05-12-market-regime.md:931`).
   - Referenced file does not use `httptest`; it executes real CLI calls with tolerant assertions (`cmd/cryptospect-cli/market_breadth_e2e_test.go`).
   - Endpoint resolution is hardcoded in fetcher and not injected per-test (`internal/api/fetcher.go:185-216`).
   - Impact: test instructions are misleading; implementer may attempt infra that current code path does not support.
   - Fix: either document live-API-tolerant E2E pattern (current standard) or add explicit fetcher URL override hooks before demanding `httptest`.

5. **[Medium] Provider test "happy path with prior dominance cache" is underspecified and likely non-functional**
   - Plan asks provider tests to assert non-cold-start by using prior state cache (`docs/superpowers/plans/2026-05-12-market-regime.md:889`).
   - Cache access is gated by config in context (`config.FromContext`) and skipped otherwise (`docs/superpowers/plans/2026-05-12-market-regime.md:685-686` pattern; see existing providers).
   - Impact: without explicit test setup (`config.StoreInContext` + temp cache dir + seeded cache entry), tests will always run as cold-start.
   - Fix: add explicit test harness steps for config context and cache seeding.

6. **[Medium] `cache_hit` semantics are ambiguous and likely misleading**
   - Plan derives `cache_hit` from `cache.Get(api.CoinGeckoGlobalMarket)` freshness (`docs/superpowers/plans/2026-05-12-market-regime.md:710-721`).
   - This does not indicate whether this run fetched from cache vs network; it only indicates current file cache state after fetch pipeline executed.
   - Impact: consumers may misread `cache_hit` as request-level provenance.
   - Fix: either rename to `cache_fresh`/`cache_entry_fresh` or pipe fetch metadata from `api.Fetcher.Fetch` into provider output.

7. **[Low] Plan creates large fixture files in `cmd/testdata` without clear consumption path**
   - Task 6 creates two fixture files (`docs/superpowers/plans/2026-05-12-market-regime.md:903-923`).
   - E2E tasks as written execute live command path and do not wire these fixtures by default (`:933-947`).
   - Impact: repo bloat and maintenance overhead without deterministic-test benefit.
   - Fix: either consume fixtures in provider-level tests only, or add an explicit mocked fetch path for command tests.

## Open Questions / Assumptions

- Should MR be allowed to diverge from the global detail contract and keep basic `meta`, or must it follow the existing `root.go` rule used by all metrics?
- Should MR status reuse `mbResult.MetricStatus` mechanically, or should MR own a separate status model where redistribution is informational (`ok`) and only sample-floor failure is degraded?
- Do we want deterministic command-level tests for MR (requiring injectable endpoint URLs), or continue the project’s current live-API tolerant E2E style?

## Recommended Plan Patches

1. Add an explicit "global contract alignment" section covering `--detail basic` meta behavior.
2. Resolve status policy conflict with `marketbreadth` before implementation (single source of truth in plan + spec + tests).
3. Correct fixture field names to `price_change_percentage_24h_in_currency`.
4. Replace "httptest" claim with the current accepted E2E pattern or add endpoint injection work as a prerequisite task.
5. Expand provider-test setup steps for cache-backed non-cold-start scenarios.
