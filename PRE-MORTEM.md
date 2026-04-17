# Pre-Mortem Report

**Scope:** `internal/api/fetcher.go`, `internal/cache/cache.go`, `internal/httpclient/client.go`, `internal/metrics/registry.go`, `internal/api/constants.go`
**Date:** 2026-04-17

## Summary

The cache-first fetch pipeline is well-constructed and passed a 9/10 code review. The fragilities here are not surface bugs — they are architectural assumptions that hold today because the tool runs as a single-invocation CLI, has a fixed set of hardcoded endpoints, and is invoked by one user at a time. Five post-mortems follow. The dominant themes are: (1) in-memory state that outlives its intended scope, and (2) stringly-typed invariants between files that no compiler or test enforces. Both themes will become load-bearing as the metric count grows and new commands are added.

---

## Post-Mortems

### 1. Batch Command Serves Stale Data to Recovered API

**Severity:** High
**Component:** `internal/api/fetcher.go` — `Fetch()`, lines 135–145 and 152–161
**Fragility type:** Semantic coupling through shared mutable state

#### What happened

A `fetch-all` command was added to run all six metrics in a single invocation. During a period of intermittent CoinGecko instability, `liquidity-pulse` fetched `coingecko.global_market` first, the API call failed, and the stale file-cache entry was served as a degraded fallback. The `market-regime` metric, which also needs `coingecko.global_market`, was computed seconds later — after the API had recovered — but received the 4-hour-old stale data from the in-memory shard. The composite regime classification was wrong. No error was returned; the output showed `"status": "degraded"` which the calling agent treated as acceptable.

#### The change that caused it

A developer added a `fetch-all` subcommand to amortize HTTP overhead when all six metrics are requested together. The command creates one `Fetcher` and calls `Fetch` for each required endpoint. This is the intended use of the memory shard — avoid duplicate HTTP calls within one invocation. The change is reasonable and the code review passed cleanly.

#### Why it broke

When `Fetch` serves a stale fallback (lines 135–145 and 152–161 of `fetcher.go`), it writes the stale data into `shard.memory[h]` and `shard.memoryMeta[h]`. The memory layer has no expiry check of its own — it is a within-process cache, not a TTL cache. The guard at lines 100–104 returns whatever is in `shard.memory` unconditionally:

```go
if data, ok := shard.memory[h]; ok {
    meta := shard.memoryMeta[h]
    meta.CacheHit = true
    return data, meta, nil
}
```

Any subsequent `Fetch` call for the same endpoint key in the same process — regardless of how much time has passed or whether the API has recovered — will hit this branch and return stale data. The memory shard is intentionally "write once per process," which is correct for the current single-metric CLI model but wrong for a batch command.

#### How it was caught

Not caught by any test. The concurrency test (`TestFetchConcurrentDifferentEndpoints`) uses distinct endpoints so each goroutine hits a fresh shard entry. No test covers two sequential `Fetch` calls for the same endpoint where the first returns stale. The bug surfaces only under transient API failures during batch execution — it would appear as inexplicably stale data in `--detail extended` output, with `"stale": true` in the metadata if you know to look.

#### Hardening suggestions

1. Add a `maxAge` field to the memory shard entry and skip the memory cache hit if the entry is stale: `if time.Since(meta.FetchedAt) > time.Duration(meta.TTLRemaining)*time.Second { /* re-fetch */ }`. This makes the memory cache a true short-circuit for duplicate calls, not a permanent override.
2. Write a test: two sequential `Fetch` calls for the same endpoint where the first call's mock returns a 503 (forcing stale fallback) and the second call's mock returns 200. Assert the second call returns fresh data, not the stale entry.

---

### 2. Registry Source Strings Silently Diverge from Endpoint Constants

**Severity:** High
**Component:** `internal/metrics/registry.go` — `RegisterDefaultMetrics()`, `internal/api/constants.go`
**Fragility type:** Stringly-typed contracts

#### What happened

While refactoring endpoint naming for a new data provider, a developer renamed `CoinGeckoCoinMarketsBreadth` from `"coingecko.coin_markets_breadth"` to `"coingecko.markets_breadth"` (removing `coin_` for consistency with a new naming scheme) and updated `constants.go` and the `resolveURL` switch. The rename was complete — `go build` passed, `go vet` passed, all tests passed. In production, `market-breadth` began returning `"status": "unavailable"` for every invocation. The fetcher's `resolveURL` hit the `default:` branch and returned an error because the string `"coingecko.coin_markets_breadth"` still existed in the `Sources` map in `registry.go`, which was not updated. The metric command used the registry source to determine which endpoint to fetch, passed the old string to `Fetch`, and `Fetch` returned an error.

#### The change that caused it

A legitimate rename of an endpoint constant to follow a newly agreed naming convention. The developer searched for `CoinGeckoCoinMarketsBreadth` (the Go identifier) and updated every reference. The string literal `"coingecko.coin_markets_breadth"` in `registry.go` was a string value, not a reference to the constant, and did not appear in the grep results.

#### Why it broke

`RegisterDefaultMetrics` in `registry.go` uses string literals for both endpoint keys and source map values:

```go
_ = reg.Register("market-breadth",
    []string{"mb"},
    []string{"coingecko.coin_markets_breadth"},          // string literal, not a constant
    map[string]string{"coin_markets_breadth": "coingecko.coin_markets_breadth"},
    "Measures participation across top assets.")
```

These are invisible copies of the values of `CoinGeckoCoinMarketsBreadth` and `CoinGeckoCoinMarketsMomentum` in `constants.go`. The compiler sees two independent string literals; there is no reference from one to the other. No test validates that the strings in the registry match the strings in `constants.go` — the tests only verify that `Fetch` works for each constant, and that the registry returns the expected strings, not that those strings agree with each other.

#### How it was caught

Only caught by manual testing or monitoring. `make test` passes because the unit tests for the registry verify its internal consistency, and the fetcher tests use the constants directly. The integration path — registry source string → metric command → `Fetch(ctx, sourceString)` — is not covered end-to-end with a test that validates the string round-trip.

#### Hardening suggestions

1. Use the constants directly in `RegisterDefaultMetrics` instead of string literals: `[]string{api.CoinGeckoCoinMarketsBreadth}`. This creates a compile-time dependency between `registry.go` and `constants.go`, so a rename of the constant is caught by the compiler.
2. Alternatively, add a test `TestRegistryEndpointsMatchConstants` that calls `AllEndpoints()` and verifies that every endpoint string appearing in any registered metric's `Endpoints` or `Sources` map is present in `AllEndpoints()`. This creates a test-time invariant.

---

### 3. `cache.Close()` Makes Subsequent Operations Write to the Working Directory

**Severity:** Medium
**Component:** `internal/cache/cache.go` — `Close()`, `filePath()`
**Fragility type:** Implicit resource lifecycle

#### What happened

A developer added `defer f.cache.Close()` to `Fetcher.New()` in preparation for a future graceful-shutdown handler. During integration testing, the test harness created a `Fetcher`, called a method that triggered an early return, and the deferred `Close()` ran — zeroing `c.dir`. A subsequent `Fetch` call (from a goroutine that had already started) called `cache.Get("coingecko.global_market")` on the closed cache. `filePath` returned `"coingecko.global_market.json"` (a relative path), which `os.ReadFile` resolved against the process working directory. The file did not exist, so `Get` returned `Found: false` — no crash, no error. However, a `cache.Set` call wrote a new file named `coingecko.global_market.json` into the test's working directory, which persisted across test runs and caused a later unrelated test to see unexpected cached data.

#### The change that caused it

Adding `defer f.cache.Close()` to the `Fetcher` constructor for resource hygiene. Every other `io.Closer` in the codebase errors or no-ops after close. The developer assumed `cache.Close()` was safe to defer and that subsequent operations would either error or do nothing.

#### Why it broke

`Close()` sets `c.dir = ""` but does not set a "closed" sentinel or prevent further method calls:

```go
func (c *Cache) Close() error {
    c.dir = ""
    return nil
}
```

`filePath` constructs its path as `filepath.Join(c.dir, endpoint+".json")`. On Linux, `filepath.Join("", "foo.json")` returns `"foo.json"` — a valid relative path in the CWD. Every subsequent `Get` and `Set` silently operates on the CWD. The existing tests all use `defer func() { _ = c.Close() }()` but never call any method after `Close`, so this path is untested.

#### How it was caught

Caught incidentally when a stray `.json` file appeared in the project root during CI. The root cause took time to identify because the failing test was not the one that wrote the file.

#### Hardening suggestions

1. Add a `closed bool` field and check it at the top of `Get`, `Set`, `Clear`, and `FilePath`, returning `fmt.Errorf("cache: use after Close")` if true. This makes post-Close operations fail loudly rather than silently misbehave.
2. Add a test: call `Close()` then call `Set()` and assert it returns an error. Call `Get()` and assert it returns an error. This costs four lines and permanently documents the contract.

---

### 4. `resolveURL` Parsed Variables Invite a Breaking Refactor

**Severity:** Low
**Component:** `internal/api/fetcher.go` — `resolveURL()`, lines 189–218
**Fragility type:** Coincidental correctness

#### What happened

A developer tidying up `resolveURL` noticed the `_ = suffix` line — a variable parsed but never used — and refactored the switch to use `suffix` instead of `endpointKey`, reasoning it was cleaner to dispatch on the endpoint-specific part rather than the full string. The switch worked for all CoinGecko endpoints (each has a unique suffix). It also worked for Binance. It failed silently for `CoinDeskAssetTopList` and `CoinMetricsCommunity`: their suffixes (`"asset_top_list"` and `"community"`) hit the `default` branch since they weren't in the new switch, returning `"no URL resolver for endpoint"` errors instead of `"not yet implemented"` errors. More subtly: if a future CoinDesk endpoint ever shared a suffix with a CoinGecko endpoint (e.g., `"derivatives"`), the wrong URL would be returned with no error.

#### The change that caused it

Dead-code cleanup. The `parts` split and `suffix` variable were there — using them felt like finishing the job the original author started. The refactor passed `go vet` and `golangci-lint`. The `TestResolveURL` test caught the `CoinDeskAssetTopList` and `CoinMetricsCommunity` cases (they expect errors), but the test would not catch a future endpoint where a suffix collision returns a wrong URL silently.

#### Why it broke

The current code switches on the full `endpointKey` for correctness — uniqueness is guaranteed at the full-string level, not the suffix level. The parsed `provider` and `suffix` variables are extracted for validation (ensuring the key has a dot) and documented as "available for future logic," but the switch intentionally uses `endpointKey`. The comment `_ = suffix // available for future endpoint‑specific logic` signals intent but does not explain why the full key is used in the switch instead of `suffix`. A future reader sees two unused variables and a switch on a longer string than necessary.

#### How it was caught

Partially caught by `TestResolveURL` (the stub endpoints errored). Not caught for the suffix-collision case because no two current endpoints share a suffix. The bug would surface in production when a new provider endpoint's suffix happened to match an existing one.

#### Hardening suggestions

1. Add a comment above the switch explaining the choice: `// Switch on the full endpoint key (not just suffix) to guarantee uniqueness across providers — two providers may share a suffix (e.g., "derivatives").`
2. Remove the `parts` split entirely and just validate the key format with `strings.Contains(endpointKey, ".")`. The current split-then-discard pattern is the source of confusion; if `provider` and `suffix` are not used, don't introduce them.

---

### 5. Global Registry Singleton Cannot Be Reset Between Tests

**Severity:** Low
**Component:** `internal/metrics/registry.go` — `GlobalRegistry()`, `globalRegistryOnce`
**Fragility type:** Invisible invariants

#### What happened

A developer writing integration tests for a new metric registered it using `GlobalRegistry().Register(...)` in a `TestMain` setup function. A previously-passing test, `TestListMetrics_Output`, began failing intermittently: it expected exactly six metrics in the `list-metrics` output but now saw seven. The failure was non-deterministic because Go's test binary runs packages in parallel; whether the new metric was registered before `TestListMetrics_Output` ran depended on goroutine scheduling.

#### The change that caused it

Adding a new metric to the registry in test setup, following the natural pattern: call `GlobalRegistry().Register(...)` to make the metric available for integration tests. The developer did not realize `GlobalRegistry()` is a package-level singleton shared by all tests in the binary, not a per-test instance.

#### Why it broke

`globalRegistry` is initialized once via `sync.Once` and is never reset:

```go
var (
    globalRegistry     *Registry
    globalRegistryOnce sync.Once
)

func GlobalRegistry() *Registry {
    globalRegistryOnce.Do(func() {
        globalRegistry = NewRegistry()
        RegisterDefaultMetrics(globalRegistry)
    })
    return globalRegistry
}
```

Any test that mutates the return value of `GlobalRegistry()` — by calling `Register` on it — affects all other tests in the same binary. There is no `Reset()` or per-test factory. Tests that assert on the count or contents of the global registry are fragile against any test that adds metrics. The `Register` method also returns `ErrDuplicateMetric` if the name already exists, so a test that calls `GlobalRegistry().Register("liquidity-pulse", ...)` will silently swallow the error if it runs after `RegisterDefaultMetrics`.

#### How it was caught

Caught during CI when test ordering changed after an unrelated file was added (changing the order of test package compilation). Local runs passed because the new test always happened to run after `TestListMetrics_Output` on the developer's machine.

#### Hardening suggestions

1. Add a `Reset()` method to `Registry` that clears `metrics` and `aliasToName`, and expose a `ResetGlobalRegistry()` function for test use. Call it in `TestMain` teardown. Guard it with a build tag (`//go:build test`) so it's not available in production binaries.
2. Any test that cares about the exact metric count should use `NewRegistry()` + `RegisterDefaultMetrics()` rather than `GlobalRegistry()`, keeping the singleton for production command wiring only. Document this convention with a comment on `GlobalRegistry`.

---

## Themes and Recommendations

**Theme 1: In-memory state with no expiry.** Post-mortems 1 and 3 both involve state that is written once and never invalidated. The memory shard in `fetcher.go` stores data for the lifetime of the process with no TTL check on reads; `cache.Close()` zeros a field but leaves the struct callable. As the tool grows toward batch commands or library use, these assumptions will be violated. A single structural fix — adding a `closed` guard to `Cache` and an age check to the memory shard — addresses both.

**Theme 2: Stringly-typed invariants across files.** Post-mortems 2 and 4 both arise from string values that are copies of, or derived from, constants defined elsewhere, with no compile-time link between them. The fix for both is the same: replace string literals in `registry.go` with references to the constants in `constants.go` (requires importing the `api` package, which is a minor but worthwhile coupling). This converts a runtime-invisible invariant into a compile-time dependency.
