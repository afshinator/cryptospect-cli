  ### 1.cache_interface.go Requirement Was Dropped
  
  Option B — drop cache_interface.go, document why — but with a twist.

  The golang-pro assessment changed the framing. The original prompt asked for a Cache interface. But that's the wrong seam. A Cache interface would wrap internal/cache/, which is
  an internal detail of api.Fetcher. Callers don't hold a cache — they hold a Fetcher. If you want to swap in Cloudflare KV later, you need to swap the Fetcher, not the cache
  underneath it.

  So the decision was:
  1. Drop cache_interface.go — the specific file the prompt asked for is genuinely not the right abstraction.
  2. Define a Fetcher interface in internal/runner instead — one method, Fetch(ctx, key) ([]byte, FetchMeta, error), drawn from the existing call site. *api.Fetcher satisfies it
  today; any future KV adapter satisfies it by implementing that one method. The TODO comment moves here.
  3. Document the decision explicitly in the spec — not just drop it silently, which was the original problem you flagged.

  The Fetcher interface also fixes a real compile-time problem: RunMetric was taking *api.Fetcher (concrete), which meant the injection point for a KV backend didn't exist — you'd
  have to change RunMetric's signature. The interface fixes that for free as a byproduct of making the design honest.

  ---

  ### 2. pure adapter claim vs runner refactor

  ● Summary: The concern was a valid observation but not a real problem. The runner is a mechanical relocation of package main code into an importable package — zero new logic. The
  spec now states this explicitly so the next reader doesn't ask the same question.

  ---

  ### 3. Nil Fetcher Panic Hazard

   Decision: valid concern, add the guard. The nil-Fetcher convention is test-only — production never hits it — but "panic on future test misuse" is worse than "error with a clear
  message." The guard is one if and a format string. It enforces the documented invariant rather than just commenting it.

  ### 4. all Branch Invents New Error Semantics

  ● Decision: "error" is not a valid MetricResult.Status — only CLIResponse uses it. The correct canonical value is "unavailable", populated via metrics.UnavailableResult(name, 
  version, namespace, msg) which also writes the error message into data so it's visible to the caller. The plan and spec are now consistent with the existing status contract.

  ### 5. Dynamic Registration Contradicted by Hardcoded Enum

    The enum is generated at Run() time from the live registry. A new metric that registers itself via init() will appear in the schema with zero MCP changes. The concern is resolved.

  ### 6. MCP Ignores Metric Aliases
    Decision: intentionally exclude aliases, document it. No code change needed. One doc addition:

  ### 7. Stdout Hygiene Is Incomplete
● Summary: the concern was valid but narrower than stated. slog was already safe via PersistentPreRunE. The only genuine gap was the standard log package, which while defaulting to
   stderr is now explicitly redirected. The spec now documents what each guard covers and why, so future reviewers don't re-ask the question.



  ### 8. “Single Source of Truth” Still Has Multiple Serialization Paths

● Decision: the concern was valid. Added output.NewSuccessResponse and output.NewErrorResponse as constructors in envelope.go. Both WriteSuccess/WriteError (CLI) and
  successResult/errorResult (MCP) now delegate to them. One envelope construction path, no drift possible. The "time" import also drops out of handler.go as a byproduct since the
  handler no longer touches timestamps directly.


  ### 9. Long-Lived Fetcher Concurrency / Lifetime Assumption
Summary: the finding was worth investigating but the code resolves it entirely. The sharded mutex design, sync.Map for handles, and shard-locked HTTP calls make Fetcher safe for
  concurrent long-lived use. This is the reason the in-memory cache exists — it's a session-level optimization, not an afterthought.

### 10. SDK Version Pinning Is Weak
The concern is valid and now fixed. The spec already stated v1.2.0; the plan's install command disagreed. They're consistent now.


The rest: 


  - #11: Add a three-layer error taxonomy to spec §7 — IsError, CLIResponse.status, and MetricResult.status are distinct non-overlapping layers.
  - #12: Document BestProviders() ordering guarantees in spec §4.2 — alphabetically sorted, unique per name, deterministic.
  - #13: Close as compliant — BuildHandler returns a function, not a struct. Subsumed by #14.
  - #14: Document in spec §3 that runner extraction is inseparable from MCP feature; acknowledge two-PR option.