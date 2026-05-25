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

---

## 3. Package: `internal/runner`

Owns the metric execution pipeline. This is the single source of truth for the
fetch → compute → post-process sequence used by both the CLI command runner and
the MCP handler.

### 3.1 Exported API

```go
func RunMetric(
    ctx    context.Context,
    p      metrics.MetricProvider,
    f      *api.Fetcher,
    detail string,
) (output.MetricResult, error)
```

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

1. Call `output.SetWriter(io.Discard)` — stdout guard. The MCP transport owns
   `os.Stdout` exclusively. Any accidental call to `output.WriteSuccess` must not
   corrupt the JSON-RPC stream. This call is made before the transport starts.
2. Create `*api.Fetcher` once via `api.New(cfg.CacheDir(), &cfg)`. This single
   instance is shared across all tool calls within the server's lifetime, enabling
   in-session memory cache sharing (e.g. `market-regime` and `liquidity-pulse`
   called consecutively share endpoint data).
3. Instantiate the MCP server:
   ```go
   server := mcp.NewServer(&mcp.Implementation{Name: "cryptospect-cli", Version: version.String()}, nil)
   ```
4. Register `query_market_analytics` tool:
   ```go
   mcp.AddTool[QueryInput, any](server, &mcp.Tool{
       Name:        "query_market_analytics",
       Description: "...",
       InputSchema: json.RawMessage(`{...enum schema from §5...}`),
   }, BuildHandler(cfg, fetcher))
   ```
   `InputSchema` is set explicitly as `json.RawMessage` so the `metric` and `detail`
   enum constraints are enforced by the SDK's jsonschema validator rather than inferred
   from the Go struct (which cannot express enums via struct tags alone).
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
func BuildHandler(cfg config.Config, f *api.Fetcher) mcp.ToolHandlerFor[QueryInput, any]
```

The handler type is `func(context.Context, *mcp.CallToolRequest, QueryInput) (*mcp.CallToolResult, any, error)`.
The `Out` type parameter is `any` — no structured output schema; the CLIResponse JSON
is returned as unstructured text content.

**Handler logic:**

1. `detail` defaults to `"basic"` if the field is empty.
2. Build a per-call context: `config.StoreInContext` + `config.StoreDetailInContext`.
3. Dispatch:
   - **Single metric:** `GlobalRegistry().Resolve(in.Metric)` →
     `runner.RunMetric(ctx, p, f, detail)` → wrap in `CLIResponse{status:"ok"}`.
   - **`all`:** `GlobalRegistry().BestProviders()` → call `runner.RunMetric` for
     each sequentially → collect all `MetricResult` values (including any with
     `status:"unavailable"`) → one `CLIResponse{status:"ok", results:[...all 10...]}`.
     Do not abort on partial failure; the existing unavailability contract handles it.
4. Marshal `CLIResponse` to JSON. Return as:
   - **Success:** `&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: jsonStr}}}`
   - **Error:** same structure with `IsError: true`

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

The handler returns a `CLIResponse` JSON payload as the tool result in all cases
except unparseable tool input (MCP protocol error).

| Situation | `IsError` | Response |
|---|---|---|
| Unknown metric name | `true` | `CLIResponse{status:"error", error:{code:400, msg:"unknown_metric", source:"mcp"}}` as text content |
| `Compute` returns non-nil error | `true` | `CLIResponse{status:"error", error:{code:500, msg:"compute_error"}}` as text content |
| API unavailable / thin data | `false` | `result.Status == "unavailable"` from `Compute` — passes through untouched |
| `all` with one metric failing | `false` | Mixed `results` array; top-level `status:"ok"` |
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
- `mcp.AddTool[In, Out any](s *Server, t *Tool, h ToolHandlerFor[In, Out])`
- `ToolHandlerFor[In, Out]` = `func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)`
- `server.Run(ctx, &mcp.StdioTransport{}) error`
- `CallToolResult.IsError bool` — set `true` for tool errors so LLM can self-correct

---

## 9. Behavioural Differences from the CLI

These are intentional, not bugs.

| Behaviour | CLI | MCP server |
|---|---|---|
| Fetcher lifetime | One per process invocation | One for server lifetime — enables in-session cache sharing |
| `--top` / `--segments` flags | Exposed per metric | Not exposed in v1; providers use compiled-in defaults |
| Output path | `output.WriteSuccess` → stdout | `json.Marshal(CLIResponse)` → MCP tool result string |
| Multi-metric | One metric per invocation | `all` returns all 10 in a single response |

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
