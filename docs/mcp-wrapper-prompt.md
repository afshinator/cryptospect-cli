We are extending an existing, fully operational Go CLI repository: https://github.com/afshinator/cryptospect-cli

The codebase is a single static binary (CGO_ENABLED=0) built with Cobra+Viper. It computes 10 crypto market regime metrics from public APIs and outputs agent-optimized JSON. Every invocation writes exactly one JSON object to stdout — a CLIResponse envelope with a `results` array of MetricResult entries. Stderr is diagnostics only.

We want to add a native MCP server capability directly into this Go repository as a new subcommand: `cryptospect-cli mcp`. This starts an stdio-based MCP server loop. No subprocess execution, no shell-outs — the MCP handler calls metric providers in-process through the existing global registry and dispatcher.

Please absorb the architecture requirements and constraints below and produce, in order: (1) a spec, (2) an implementation plan, (3) the implementation.

---

## 1. Verified Interface Contract

The following is ground truth extracted from the actual codebase. Build everything against this — do not infer a different interface.

### MetricProvider interface (`internal/metrics/provider.go`)

```go
type MetricProvider interface {
    Def() MetricDef
    Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error)
}
```

`Compute` takes a plain `map[string]json.RawMessage` of pre-fetched endpoint data. No `*cobra.Command`, no Viper reads, no flag globals. All metrics implement this identically — there is no signature variance across metrics.

### MetricDef (`internal/metrics/provider.go`)

```go
type MetricDef struct {
    Name        string
    Namespace   string
    Version     string
    Aliases     []string
    Endpoints   []string   // endpoint keys this metric needs fetched
    Description string
}
```

`Def().Endpoints` is the list of endpoint keys the dispatcher must fetch before calling `Compute`. This is how the MCP handler knows what data to fetch for each metric.

### Global Registry (`internal/metrics/registry.go`)

```go
metrics.GlobalRegistry().Resolve("liquidity-pulse") // → MetricProvider, error
metrics.GlobalRegistry().BestProviders()            // → []MetricProvider (all metrics, one per name)
```

All metrics self-register via `init()` using `metrics.MustRegister(&Provider{})`. The global registry is fully populated by the time `main()` runs. The MCP handler does not need a routing table — it calls `Resolve(metricName)` to get the provider, then calls `Def().Endpoints` to know what to fetch, then calls `Compute`.

### Registration pattern (from `liquidity-pulse/v1/provider.go`)

```go
func init() { metrics.MustRegister(&Provider{}) }
```

Every metric package registers itself. The MCP handler imports all metric packages (the same blank imports the CLI already has) and gets the full registry for free.

### output.MetricResult

`Compute` returns `output.MetricResult` — the same struct the CLI marshals into the `results` array. The MCP handler wraps one or more of these into the standard `CLIResponse` envelope and returns the JSON as the tool result.

---

## 2. Architecture Constraints

**No re-implementations.** The MCP layer is a pure adapter. Zero trading logic, zero threshold calculations, zero financial math. It resolves providers from the registry, triggers the existing dispatcher to fetch endpoint data, calls `Compute`, and serializes the result.

**Single tool, not ten.** Expose exactly ONE tool: `query_market_analytics`. It accepts a `metric` string enum and a `detail` string enum. The handler resolves the provider via `GlobalRegistry().Resolve(metric)`, fetches the endpoints declared in `Def().Endpoints` through the existing cache-aware dispatcher, calls `Compute`, and wraps the result in a `CLIResponse` envelope. Adding a new metric in future requires zero MCP changes — it self-registers via `init()` and appears automatically.

**No heavy handler structs.** Register the tool using `mcp.NewTool(...)` with a clean anonymous handler function. No tracking structs, middleware chains, or custom registries beyond what the global registry already provides.

**Cache reuse.** The existing dispatcher handles in-memory + file cache (`~/.cryptospect-cli/cache/`). The MCP handler must fetch endpoint data through this same dispatcher, not directly via HTTP. Running `query_market_analytics` for `market-regime` and then `liquidity-pulse` in the same session must hit the cache for shared endpoints, exactly as the CLI does today.

**No WASM target.** Stdio-based MCP servers require OS process semantics. Do not add a WASM build target.

**Defer cloud/KV persistence.** The existing file cache is the target for this session. Define a `Cache` interface that a future KV adapter could satisfy, leave a clearly marked `// TODO: KV adapter` stub, and move on. Do not implement Cloudflare KV.

---

## 3. Tool Schema

```json
{
  "name": "query_market_analytics",
  "description": "Fetches computed crypto market regime signals. Returns agent-optimized JSON. Use detail=full when feeding output to an LLM — it includes metric descriptions and thresholds. Use detail=basic for lightweight loops.",
  "input_schema": {
    "type": "object",
    "properties": {
      "metric": {
        "type": "string",
        "enum": [
          "liquidity-pulse",
          "stablecoin-power",
          "flow-tension",
          "market-breadth",
          "momentum-divergence",
          "market-regime",
          "dominance",
          "volatility",
          "fear-greed-index",
          "china-m2",
          "all"
        ],
        "description": "Which metric to compute. 'all' runs every metric via GlobalRegistry().BestProviders() and returns all results in a single CLIResponse envelope."
      },
      "detail": {
        "type": "string",
        "enum": ["basic", "extended", "full"],
        "default": "basic",
        "description": "Controls output verbosity. Mirrors the CLI --detail flag. Must be extracted from the tool call input and passed into the dispatcher per-call — do not read from global Viper state mid-session."
      }
    },
    "required": ["metric"]
  }
}
```

---

## 4. Directory Layout

Add to the existing tree — do not restructure what exists:

```
cmd/cryptospect-cli/
  mcp.go              # cobra subcommand: registers "mcp" command, calls internal/mcp
internal/mcp/
  server.go           # MCP server init, stdio transport, tool registration, stdout guard
  handler.go          # query_market_analytics handler: registry lookup + dispatcher call
  cache_interface.go  # Cache interface + local file adapter wrapping existing cache
```

The existing metric packages in `internal/metrics/<name>/v1/` are not touched.

---

## 5. MCP SDK

Use `github.com/modelcontextprotocol/go-sdk` — the official Go MCP SDK. It uses standard Go channels to process inbound JSON-RPC messages asynchronously over `os.Stdin` and `os.Stdout`. Configure stdio transport. Handle graceful shutdown on SIGINT/SIGTERM.

---

## 6. Critical Implementation Details

### 6a. The `all` metric branch

When `metric == "all"`, call `metrics.GlobalRegistry().BestProviders()` to get every registered provider. For each, fetch its `Def().Endpoints` through the dispatcher and call `Compute`. Collect all `output.MetricResult` values into a single `CLIResponse` envelope with a `results` array. Do not return multiple separate responses. Concurrent dispatch via `sync.WaitGroup` is acceptable but sequential is fine for v1.

### 6b. Dynamic `detail` per call

The `mcp` subcommand is a long-lived process. The `detail` argument arrives per tool call from the agent. Extract it from the tool call input and pass it into the dispatcher context for that specific invocation. Do not read `detail` from global Viper state mid-session — Viper reflects boot-time flags only.

### 6c. Stdout hygiene — do not corrupt the JSON-RPC stream

The MCP protocol owns `os.Stdout` exclusively for JSON-RPC framing. Any `fmt.Println`, `log.Print`, or Viper diagnostic output reaching stdout will corrupt the stream and crash the connection. As the first action in `server.go`, before starting the transport, redirect all application logging and Viper output to `os.Stderr`. Only the MCP SDK transport instance touches `os.Stdout` after that point.

---

## 7. Makefile

The `mcp` subcommand is part of the same binary — no separate artifact. If the existing `build` target already produces `bin/cryptospect-cli` from `./cmd/cryptospect-cli/`, no Makefile change is needed. Do not add a WASM target.

---

## 8. What to Produce

**Spec:** Describe the full data flow: MCP tool call → `Resolve(metric)` → `Def().Endpoints` → dispatcher fetch (cache-first) → `Compute(ctx, data)` → `CLIResponse` envelope → JSON-RPC response. Confirm how the existing dispatcher accepts a detail level and an endpoint list. Flag any remaining unknowns about the dispatcher's call signature before the plan step.

**Plan:** Ordered list of implementation steps with file-level granularity. Step 1: inspect the existing dispatcher in `internal/` to confirm its call signature and how `detail` is passed through. Step 2: add the MCP SDK to go.mod. All subsequent steps follow from step 1 findings.

**Implementation:** Complete, compilable Go code for each new file. No placeholders except the explicitly marked KV adapter stub. Handler must use an anonymous function — no unnecessary structs.