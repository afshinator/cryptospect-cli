# Go 1.25 Structured Caching Patterns — Evaluation

*Created: 2026‑04‑16*  
*Updated: 2026‑04‑17*  
*Context: Architecture tip received about using Go 1.25’s new structured caching patterns to keep the binary static and high‑performance.*

---

## 1. The Tip (Summarized)

In Go 1.25, **structured caching patterns** refer to a shift away from simple global maps toward:

- **Highly localized, CPU‑aware, type‑safe architectures**
- **Zero‑allocation keying** with `unique.Handle` (introduced in 1.23, optimized in 1.25)
- **Alignment with the “Green Tea” garbage collector** (experimental `GOEXPERIMENT=greenteagc`) for programs that manage many small objects
- **Virtualized concurrency testing** with the stabilized `testing/synctest` package
- **Static binary optimization** via DWARF v5 (default in 1.25)
- **Sharded maps + fine‑grained locking** to reduce contention
- **Faster JSON serialization** with `GOEXPERIMENT=jsonv2`

The goal: ensure the **cache‑first fetcher** remains the high‑performance backbone of `cryptospect‑cli` without bloating the static binary.

---

## 2. Current Implementation Status (as of 2026‑04‑16)

| Pattern | Implemented? | Location | Notes |
| :--- | :--- | :--- | :--- |
| **Zero‑allocation keying with `unique.Handle`** | ✅ Yes | `internal/api/fetcher.go:21‑32` | Global `sync.Map` canonicalizes endpoint strings to `unique.Handle[string]`; memory cache uses `map[unique.Handle[string]][]byte` and `map[unique.Handle[string]]FetchMeta`. |
| **“Green Tea” GC‑aligned contiguous spans** | ❌ No | `internal/api/fetcher.go` | Two separate maps create pointer‑chasing; no slice‑based contiguous storage. |
| **Virtualized concurrency (`testing/synctest`)** | ❌ No | Test files (`*_test.go`) | Uses standard `time` mocking; no `synctest.Wait()` or time‑virtualized bubbles. |
| **DWARF v5 static binary optimization** | ✅ Yes (auto) | `go.mod:3` | Go 1.25.9 → linker uses DWARF v5 by default; binary is already minimal (`CGO_ENABLED=0`). |
| **Sharded map + fine‑grained locking** | ✅ Yes | `internal/api/fetcher.go:34‑82` | 16 shards (power of two), each with its own `sync.Mutex` and maps; FNV‑1a hash distributes endpoints; concurrent test passes. |
| **JSON v2 experiment (`GOEXPERIMENT=jsonv2`)** | ❌ No | All JSON serialization | Uses standard `encoding/json`; no `GOEXPERIMENT=jsonv2` enabled. |

---

## 3. Architecture Fit Assessment

### What Works Today (Code‑Review Score: 9/10)
- Cache‑first hierarchy: **memory → file → HTTP → stale fallback**
- Atomic file writes, proper error wrapping, race‑detector safe
- Meets functional requirements for **v1**

### Missed 1.25 Performance Opportunities
1. **GC pressure** – ~~Every cache lookup allocates strings~~ **Resolved**: `unique.Handle` eliminates string allocations for memory cache lookups.
2. **Lock contention** – ~~Single mutex blocks concurrent metric fetches~~ **Resolved**: 16‑shard map with per‑shard mutex enables concurrent fetches to different endpoints.
3. **Test flakiness** – ~~Relies on real `time.Sleep` for stale‑fallback tests~~ **Resolved**: `TestStaleEntry` now uses TTL=0 (entry is immediately stale) — no sleep required. `testing/synctest` still not adopted.
4. **Serialization overhead** – JSON v2 could speed up `CLIResponse` envelope encoding by **30–50%**.

---

## 4. Importance for This Project

### High‑priority (✅ implemented before v1 release)
- **Sharded maps** – ~~Concurrent metric commands will contend on the single mutex~~ **Implemented**: 16 shards with per‑shard mutex eliminate contention.
- **`unique.Handle`** – ~~Reduces allocations for high‑frequency polling~~ **Implemented**: Zero‑allocation keying via global `sync.Map` canonicalization.

### Medium‑priority (nice‑to‑have optimizations)
- **JSON v2** – Faster output, but envelope size is small (~1‑2 KB).
- **`testing/synctest`** – Improves test reliability, not runtime performance.

### Low‑priority (deferrable)
- **Green‑Tea GC alignment** – Maps work fine at current scale (<100 endpoints).
- **Contiguous slices** – Premature optimization unless profiling shows GC pressure.

---

## 5. Recommended Action Plan

1. ✅ **Immediate (before CLI commands)** – Add **sharded locking** to `fetcher.go`:
   - ~~Replace `map[string][]byte` with `[]sync.Map` shard per endpoint hash.~~ **Implemented**: 16 shards each with `map[unique.Handle[string]]…` and `sync.Mutex`.
   - ~~Enables concurrent `liquidity‑pulse` + `stablecoin‑power` fetches.~~ **Verified** by `TestFetchConcurrentDifferentEndpoints`.

2. ✅ **Before v1** – Migrate to **`unique.Handle` for cache keys**:
   - ~~`internal/api/constants.go` → pre‑compute handles for all endpoint keys.~~ **Implemented**: `getHandle` with global `sync.Map` canonicalization.
   - ~~`fetcher.memory` → `map[unique.Handle][]byte`.~~ **Implemented**: shard‑local maps use `unique.Handle[string]` keys.
   - ~~Zero‑allocation lookups for high‑frequency use.~~ **Achieved**: lookups compare pointer‑sized integers.

3. **Testing enhancement** – Adopt **`testing/synctest`** for stale‑fallback tests:
   - Replace `time.Sleep` with virtualized bubbles.
   - Faster, deterministic integration tests.

4. **Build optimization** – Enable **`GOEXPERIMENT=jsonv2`** in `Makefile`:
   - `go build -tags=jsonv2` or environment variable.
   - Measure envelope serialization improvement.

---

## 6. Trade‑Off Consideration

**Simplicity vs. Performance**: The current cache works and passed rigorous review. These optimizations add complexity for marginal gains at **v1 scale** (6 metrics, occasional runs). However, they future‑proof the architecture for:

- Batch commands (fetch all metrics at once).
- High‑frequency polling (agent‑driven monitoring).
- Many concurrent users (CLI as library).

**Decision**: ✅ **sharded maps** and **`unique.Handle` implemented** before v1; defer others until post‑launch profiling justifies them.

---

## 7. References

- Go 1.25 Release Notes: https://go.dev/doc/go1.25
- `unique` package documentation: `go doc unique`
- `testing/synctest` proposal: [GitHub Issue](https://github.com/golang/go/issues/…)
- JSON v2 experiment: `GOEXPERIMENT=jsonv2`

---

*This document should be updated when any of the above patterns are adopted.*