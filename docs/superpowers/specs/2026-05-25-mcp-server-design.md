# MCP Server Design

**Date:** 2026-05-25
**Status:** Approved — ready for implementation planning
**Reference prompt:** `docs/mcp-wrapper-prompt.md`

---

## 1. Goal

Add a native `cryptospect-cli mcp` subcommand that starts a stdio-based MCP server.
The server exposes one tool — `query_market_analytics` — that calls metric providers
in-process through the existing global registry and dispatcher. No subprocess execution,
no shell-outs, no re-implemented trading logic.

---

## 2. File Layout

No existing metric packages are touched. Changes are confined to `cmd/cryptospect-cli/` and new `internal/` packages.

```
cmd/cryptospect-cli/
  mcp.go                       # cobra "mcp" subcommand (new)

internal/
  runner/
    runner.go                  # RunMetric + unexported helpers (new)
    runner_test.go             # migrated + extended tests (new)
  mcp/
    server.go                  # MCP server init, transport, signal handling (new)
    handler.go                 # query_market_analytics handler + BuildHandler (new)
    handler_test.go            # handler unit tests (new)
```

**Deleted:** `cmd/cryptospect-cli/meta_processing.go` and
`cmd/cryptospect-cli/meta_processing_test.go` — their logic moves to `internal/runner/`.

**Modified:** `cmd/cryptospect-cli/root.go` — `buildMetricRunE` is refactored to call
`runner.RunMetric`; the fetch loop and meta post-processing are removed from it.

**Note — `cache_interface.go` from original prompt:** The original prompt (`docs/mcp-wrapper-prompt.md` §4) required `internal/mcp/cache_interface.go` with a `Cache` interface and file-adapter stub. This file is intentionally omitted. Its extensibility goal is fully covered by the `Fetcher` interface defined in `internal/runner` (see §3.1). That interface is the correct seam: callers already hold a `Fetcher`, and any future KV adapter satisfies it by implementing a single `Fetch` method. A separate `Cache` interface inside `internal/mcp` would be a redundant indirection wrapping an implementation detail of `api.Fetcher`. The `// TODO: KV adapter` comment is placed in `internal/runner/runner.go` at the `Fetcher` type definition.

---

## 3. Package: `internal/runner`

Owns the metric execution pipeline. This is the single source of truth for the
fetch → compute → post-process sequence used by both the CLI command runner and
the MCP handler.

**This is an extraction, not a rewrite.** Every line in `runner.go` already exists in
`cmd/cryptospect-cli/meta_processing.go` (`aggregateFetchMeta`, `postProcessMeta`,
`fullDetailOnlyFields`) and `root.go:buildMetricRunE` (fetch loop, `Compute` call,
detail-level switch). The only reason it cannot be shared today is that it lives in
`package main`, which is unimportable. Moving it to `internal/runner` changes no
behavior — it makes the existing CLI execution path importable by `internal/mcp`.
The "pure adapter" constraint from the original prompt is not violated.

**Scope note:** This extraction is the smallest possible change that makes the pipeline
importable. Without it, the MCP handler would duplicate the fetch→compute→post-process
sequence from `root.go`. The extraction has no independent value without the MCP consumer —
both changes belong in a single PR. A two-PR approach (extract first, MCP second) is valid
if the team prefers isolated review, but is not required. `BuildHandler` returns a plain
function, not a struct — the "no heavy handler structs" constraint from the original prompt
is satisfied.

### 3.1 Exported API

```go
// Fetcher is the minimal interface RunMetric needs from the HTTP/cache layer.
// *api.Fetcher satisfies it; a future KV adapter only needs to implement Fetch.
// TODO: KV adapter — implement Fetcher backed by Cloudflare KV or similar.
type Fetcher interface {
    Fetch(ctx context.Context, endpointKey string) ([]byte, api.FetchMeta, error)
}

func RunMetric(
    ctx    context.Context,
    p      metrics.MetricProvider,
    f      Fetcher,           // accepts *api.Fetcher or any future KV adapter
    detail string,
) (output.MetricResult, error)
```

`f` may be `nil` only when `p.Def().Endpoints` is empty — `Fetch` is never called in that case.
`RunMetric` guards this explicitly: if `f == nil` and the provider has endpoints, it returns an error rather
than panicking. This converts a latent test-misuse hazard into an early, informative failure.

**Behaviour:**

1. Iterate `p.Def().Endpoints`. For each key call `f.Fetch(ctx, key)`.
   - On fetch error: log via `slog.Debug`, set `data[key] = nil` (matches current CLI
     behaviour — lets `Compute` decide how to handle missing data).
2. Call `p.Compute(ctx, data)`.
3. Post-process `result.Meta`:
   - `detail == "basic"`: set `result.Meta = nil`.
   - `detail == "extended"` or `"full"`: inject `cache_hit` / `ttl_remaining_sec`,
     strip full-detail-only fields when `extended`.
4. Return the post-processed `MetricResult`. Errors from `Compute` are returned
   directly and are reserved for catastrophic failures; normal unavailability is
   expressed via `result.Status = "unavailable"`.

### 3.2 Unexported helpers (moved from `meta_processing.go`)

- `aggregateFetchMeta(map[string]api.FetchMeta) (cacheHit bool, ttlRemaining int)`
- `postProcessMeta(metaJSON json.RawMessage, detail string, cacheHit bool, ttlRemaining int, fullOnlyFields []string) json.RawMessage`
- `fullDetailOnlyFields []string` — package-level var listing meta keys stripped at `extended`

These are not exported. They are implementation details of `RunMetric`.

### 3.3 CLI refactor

`buildMetricRunE` in `root.go` shrinks from ~55 lines to ~15. It retains:
- Config and flag extraction from context (including `--top`, `--segments`)
- `api.New(...)` Fetcher construction
- Call to `runner.RunMetric`
- Call to `output.WriteSuccess`

The flag propagation for `--top` / `--segments` stays in `root.go` — it is
cobra-specific and does not belong in `runner`.

---

## 4. Package: `internal/mcp`

### 4.1 `server.go`

Exports one function:

```go
func Run(ctx context.Context, cfg config.Config) error
```

Sequence on entry:

1. Stdout guard — first action inside `Run()`, before the transport starts. All four
   redirects are in `server.go`, not in `root.go` or `PersistentPreRunE`:
   - `output.SetWriter(io.Discard)` — covers `output.WriteSuccess` / `WriteError`.
   - `log.SetOutput(os.Stderr)` — explicit redirect for the standard `log` package
     (default is already stderr, but explicit is safer given the MCP stream invariant).
   - `slog` is already on stderr: `PersistentPreRunE` runs before `mcp.RunE` and calls
     `slog.SetDefault` with a stderr handler. No action needed in `server.go`.
   - `panic` traces: the Go runtime writes panic output directly to `os.Stderr` — not
     via `log` or `slog` — so no redirect is needed. This holds for panics in provider
     code, SDK transport goroutines, and Viper internals alike.
   - No internal packages in this codebase write directly to `os.Stdout` (confirmed by audit).
2. Create `*api.Fetcher` once via `api.New(cfg.CacheDir(), &cfg)`. This single
   instance is shared across all tool calls within the server's lifetime, enabling
   in-session memory cache sharing (e.g. `market-regime` and `liquidity-pulse`
   called consecutively share endpoint data).

   **Thread-safety confirmed** (source: `internal/api/fetcher.go`): `Fetcher` uses a
   16-shard mutex architecture — each endpoint key hashes to one of 16 shards, each
   protected by its own `sync.Mutex`. All map reads/writes are under the shard lock.
   HTTP calls are also made under the shard lock, serialising concurrent requests to
   the same endpoint (preventing duplicate in-flight requests). `handleMap` uses
   `sync.Map`. Long-lived single-instance reuse is the intended use pattern.

   **Shard-held-during-IO note:** the shard lock is held for the full fetch lifecycle —
   file cache read, HTTP request, retries, and cache write. This is a deliberate
   single-flight gate: concurrent callers to the same endpoint wait rather than fire
   duplicate HTTP requests. The practical throughput ceiling is 16 concurrent in-flight
   fetches (one per shard). For v1 this is not a constraint: `all` dispatches
   sequentially, and stdio transport means a single client stream — true shard
   contention cannot occur. If concurrent dispatch is added in a future version (already
   out of scope for v1), revisit lock scope or replace with `singleflight.Group`.
3. Instantiate the MCP server:
   ```go
   server := mcp.NewServer(&mcp.Implementation{Name: "cryptospect-cli", Version: version.String()}, nil)
   ```
4. Register `query_market_analytics` tool:
   ```go
   mcp.AddTool[QueryInput, any](server, &mcp.Tool{
       Name:        "query_market_analytics",
       Description: "...",
       InputSchema: buildToolSchema(),
   }, BuildHandler(cfg, fetcher))
   ```
   `Tool.InputSchema` is `*jsonschema.Schema` (from `github.com/modelcontextprotocol/go-sdk/mcp/jsonschema`),
   not `json.RawMessage`. `buildToolSchema()` constructs it programmatically from
   `metrics.GlobalRegistry().BestProviders()` so the metric enum is always in sync
   with registered providers — no hardcoded list. Import alias for jsonschema:
   `"github.com/modelcontextprotocol/go-sdk/mcp/jsonschema"`.

   **Blank-import dependency:** `internal/mcp/server.go` does not import metric packages
   directly. The global registry is populated by the blank imports in `cmd/cryptospect-cli/root.go`
   (same `package main` binary), which run before `Run()` is called. If `internal/mcp`
   were compiled into a binary that omits those blank imports, `BestProviders()` would
   return empty and the schema enum would contain only `["all"]`. This is not a concern
   for the current binary but is worth noting for any future reuse of `internal/mcp`.
5. Wire SIGINT/SIGTERM to context cancellation for graceful shutdown.
6. Start stdio transport and block until the context is cancelled or transport exits:
   ```go
   server.Run(ctx, &mcp.StdioTransport{})
   ```

### 4.2 `handler.go`

Defines the input struct and exports one constructor:

```go
// QueryInput is the typed input for the query_market_analytics tool.
// The SDK unmarshals and validates tool call arguments into this struct.
type QueryInput struct {
    Metric string `json:"metric" jsonschema:"which metric to compute"`
    Detail string `json:"detail,omitempty" jsonschema:"output verbosity: basic, extended, or full"`
}

// BuildHandler returns the typed tool handler closed over cfg and f.
// Separating construction from registration makes the handler testable
// without instantiating a full MCP server.
//
// f accepts runner.Fetcher (satisfied by *api.Fetcher or any future adapter).
// Pass nil only in tests where all providers have no endpoints.
func BuildHandler(cfg config.Config, f runner.Fetcher) func(context.Context, *mcp.CallToolRequest, QueryInput) (*mcp.CallToolResult, any, error)
```

The explicit function type is used instead of `mcp.ToolHandlerFor[QueryInput, any]` to avoid
ambiguity between the SDK's internal low-level type and the convenience form accepted by `mcp.AddTool[In, Out]`.
The `Out` type parameter is `any` — no structured output schema; the CLIResponse JSON
is returned as unstructured text content.

**Handler logic:**

1. `detail` defaults to `"basic"` if the field is empty.
2. Build a per-call context: `config.StoreInContext` + `config.StoreDetailInContext`.
3. Dispatch:
   - **Single metric:** `GlobalRegistry().Resolve(in.Metric)` →
     `runner.RunMetric(ctx, p, f, detail)` → wrap in `CLIResponse{status:"ok"}`.
     If `RunMetric` returns a non-nil error (catastrophic `Compute` failure), return
     `errorResult(500, "compute_error")` with `IsError: true`. This is correct: when
     the one thing you asked for fails completely, the tool call itself fails.
   - **`all`:** `GlobalRegistry().BestProviders()` → call `runner.RunMetric` for
     each sequentially → collect all `MetricResult` values (including any with
     `status:"unavailable"`) → one `CLIResponse{status:"ok", results:[...all 10...]}`.
     Do not abort on partial failure; the existing unavailability contract handles it.
     If `RunMetric` returns a non-nil error for one provider, wrap it via
     `metrics.UnavailableResult(...)` and continue — partial success is still success.

     **Error asymmetry is intentional:** single-metric catastrophic failure → tool fails
     (`IsError: true`); `all` partial failure → tool succeeds, one entry is `"unavailable"`.
     Do not "fix" the `all` branch to match the single-metric error path.

     **Registry stability** (confirmed from `internal/metrics/registry.go`):
     `BestProviders()` sorts names with `slices.Sort`, uses a `seen` map to deduplicate,
     and selects deterministically per name (core namespace preferred, highest semver wins).
     The `all` response order is alphabetical by canonical name and stable across calls.
4. Marshal `CLIResponse` to JSON. Return as:
   - **Success:** `json.Marshal(output.NewSuccessResponse(results))` → `&mcp.CallToolResult{Content: [TextContent{...}]}`
   - **Error:** `json.Marshal(output.NewErrorResponse(CLIError{...}))` → same with `IsError: true`

   `output.NewSuccessResponse` and `output.NewErrorResponse` are new constructors added to `internal/output/envelope.go`. `WriteSuccess` and `WriteError` in `writer.go` delegate to the same constructors, ensuring a single construction path for both CLI and MCP output.

   > Per the go-sdk docs: tool errors must be reported inside `Content` with `IsError: true`,
   > not as MCP protocol-level errors — otherwise the LLM cannot see the error and
   > self-correct. Protocol-level errors are reserved for "tool not found" and other
   > exceptional conditions outside the handler.

---

## 5. Tool Schema

```json
{
  "name": "query_market_analytics",
  "description": "Fetches computed crypto market regime signals. Returns agent-optimized JSON. Use detail=full when feeding output to an LLM — it includes metric descriptions and thresholds. Use detail=basic for lightweight loops.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "metric": {
        "type": "string",
        "enum": [
          "liquidity-pulse", "stablecoin-power", "flow-tension",
          "market-breadth", "momentum-divergence", "market-regime",
          "dominance", "volatility", "fear-greed-index", "china-m2",
          "all"
        ],
        "description": "Which metric to compute. 'all' runs every metric and returns all results in a single CLIResponse envelope."
      },
      "detail": {
        "type": "string",
        "enum": ["basic", "extended", "full"],
        "default": "basic",
        "description": "Controls output verbosity. Mirrors the CLI --detail flag."
      }
    },
    "required": ["metric"]
  }
}
```

---

## 6. Cobra Subcommand: `cmd/cryptospect-cli/mcp.go`

```go
func newMCPCommand() *cobra.Command {
    return &cobra.Command{
        Use:     "mcp",
        Short:   "Start an MCP server over stdio",
        GroupID: "utility",
        Long: `Start a Model Context Protocol server over stdio.

Exposes the query_market_analytics tool to any MCP-compatible client.
The --detail flag is inherited but unused; detail is specified per tool call.`,
        RunE: func(cmd *cobra.Command, _ []string) error {
            cfg, ok := config.FromContext(cmd.Context())
            if !ok {
                return fmt.Errorf("config not found in context")
            }
            return mcpserver.Run(cmd.Context(), cfg)
        },
    }
}
```

Added to `NewRootCommand()` alongside the existing `listCmd` and `cacheCmd`.

**Import alias:** `internal/mcp` declares `package mcp`. In `cmd/cryptospect-cli/mcp.go`
it is imported with the alias `mcpserver` to avoid visual confusion with the go-sdk's
`mcp` package (which appears in `internal/mcp/server.go` but not in the cobra file):

```go
import mcpserver "github.com/afshinator/cryptospect-cli/internal/mcp"
```

**Note on `PersistentPreRunE`:** The root command's persistent pre-run executes for
the `mcp` subcommand. It validates `--detail` (harmless — defaults to `"basic"`),
configures slog to write to stderr (correct), and loads config into context (required).
The boot-time `--detail` value is ignored at runtime; per-call detail arrives via
tool input.

---

## 7. Error Handling

### Error layer taxonomy

Three layers represent failure independently — they are non-overlapping:

| Layer | Owner | Field | Values | Retry semantics |
|---|---|---|---|---|
| MCP transport | MCP SDK | `CallToolResult.IsError` | `true` / `false` | `true` → LLM client should consider retrying the tool call, possibly with different parameters |
| Envelope | `internal/output` | `CLIResponse.status` | `"ok"` / `"error"` | `"error"` always co-occurs with `IsError: true`; retry decisions are at tool-call level |
| Per-metric | metric provider | `MetricResult.status` | `"ok"` / `"degraded"` / `"unavailable"` | Data quality signal, not a tool failure; LLM may proceed with degraded data or retry the call later |

`IsError: true` always accompanies `CLIResponse.status:"error"` — both are set together
for tool-level failures. `CLIResponse.status:"ok"` with `IsError: false` can still contain
`MetricResult.status:"unavailable"` entries — this is the normal degraded-data path, not a
tool failure. The per-metric `"unavailable"` status never escapes to the envelope level.

### Error table

The handler returns a `CLIResponse` JSON payload as the tool result in all cases
except unparseable tool input (MCP protocol error).

| Situation | `IsError` | Response |
|---|---|---|
| Unknown metric name | `true` | `CLIResponse{status:"error", error:{code:400, msg:"unknown_metric", source:"mcp"}}` as text content |
| `Compute` returns non-nil error | `true` | `CLIResponse{status:"error", error:{code:500, msg:"compute_error"}}` as text content |
| API unavailable / thin data | `false` | `result.Status == "unavailable"` from `Compute` — passes through untouched |
| `all` with one metric failing | `false` | Mixed `results` array; failing entry gets `status:"unavailable"` via `metrics.UnavailableResult`; top-level `status:"ok"` |
| Unparseable tool input | n/a | MCP SDK protocol error — only case outside the CLIResponse envelope |

`IsError: true` on the `CallToolResult` signals to MCP clients that the tool call failed
while still delivering structured JSON the LLM can inspect for retry decisions.

---

## 8. MCP SDK Dependency

**SDK:** `github.com/modelcontextprotocol/go-sdk` — official Go MCP SDK.

Added to `go.mod` / `go.sum` as a direct dependency. The MCP binary is the same
artifact as the existing CLI (`make build` → `bin/cryptospect-cli`). No new build
target. No WASM target.

**Verified API surface** (context7, v1.2.0):
- `mcp.NewServer(&mcp.Implementation{Name, Version}, nil) *mcp.Server`
- `mcp.AddTool[In, Out any](s *Server, t *Tool, h func(context.Context, *CallToolRequest, In) (*CallToolResult, Out, error))`
- `Tool.InputSchema` is `*jsonschema.Schema` (pkg `github.com/modelcontextprotocol/go-sdk/mcp/jsonschema`) — **not** `json.RawMessage`
- `server.Run(ctx, &mcp.StdioTransport{}) error`
- `CallToolResult.IsError bool` — set `true` for tool errors so LLM can self-correct
- `CallToolResult.Content []Content` where `Content` is an interface; `*TextContent{Text: "..."}` implements it

---

## 9. Behavioural Differences from the CLI

These are intentional, not bugs.

| Behaviour | CLI | MCP server |
|---|---|---|
| Fetcher lifetime | One per process invocation | One for server lifetime — enables in-session cache sharing |
| `--top` / `--segments` flags | Exposed per metric | Not exposed in v1; providers use compiled-in defaults |
| Output path | `output.WriteSuccess` → stdout | `json.Marshal(CLIResponse)` → MCP tool result string |
| Multi-metric | One metric per invocation | `all` returns all 10 in a single response |
| Metric aliases (`lp`, `sp`, `ft`, `cnm2`…) | Accepted by all commands | **Intentionally excluded.** The schema enum lists canonical names only. Aliases are CLI brevity shortcuts; canonical names are more useful for AI-agent discoverability. SDK input validation rejects aliases before the handler is reached. |

---

## 10. Testing

### `internal/runner/runner_test.go`
- Migrate all cases from `meta_processing_test.go` verbatim.
- Add table-driven tests for `RunMetric` using `httptest` servers:
  happy path, stale cache fallback, endpoint fetch failure (data becomes nil),
  nil meta passthrough, `detail=basic` strips meta, `detail=full` injects cache fields.

### `internal/mcp/handler_test.go`
- Call `BuildHandler(cfg, fetcher)` directly with an httptest-backed Fetcher.
- Cases: valid single metric → correct CLIResponse shape; unknown metric → error
  envelope with code 400; `all` → 10 results in one envelope; `detail=full` → meta
  present; `detail=basic` → meta nil.

### No MCP E2E test for v1
The existing 10 per-metric E2E tests cover metric correctness. The MCP handler is a
thin adapter over `runner.RunMetric`; handler unit tests + runner unit tests provide
sufficient coverage without the complexity of spawning a binary and connecting over
stdio.

---

## 11. Out of Scope (v1)

- `--top` / `--segments` as MCP tool input fields
- Concurrent dispatch for the `all` branch
- KV / cloud cache backend
- WASM build target
- Separate MCP E2E test suite
