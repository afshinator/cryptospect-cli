# MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `cryptospect-cli mcp` — a stdio-based MCP server exposing a single `query_market_analytics` tool that calls metric providers in-process through the existing global registry.

**Architecture:** Extract the fetch→compute→post-process pipeline from `root.go` into `internal/runner/RunMetric`, which both the CLI and MCP handler share. The MCP handler (`internal/mcp/handler.go`) is a thin adapter that parses tool input, calls `RunMetric`, and serializes the result as a `CLIResponse` JSON string. The server (`internal/mcp/server.go`) owns the process lifecycle: stdout guard, long-lived `*api.Fetcher`, stdio transport, and signal handling.

**Tech Stack:** Go 1.25, `github.com/modelcontextprotocol/go-sdk` v1.2.0 (official MCP SDK), existing Cobra+Viper CLI infrastructure.

**Corrections vs original design:**
1. `internal/runner` defines a `Fetcher` interface (`Fetch(ctx, key) ([]byte, api.FetchMeta, error)`). `RunMetric` and `BuildHandler` accept this interface, not `*api.Fetcher` directly. This is the extensibility seam that supersedes `cache_interface.go` from the original prompt — see the spec §2 note.
2. `Tool.InputSchema` is `*jsonschema.Schema` (from `github.com/modelcontextprotocol/go-sdk/mcp/jsonschema`), not `json.RawMessage`. `buildToolSchema()` in `server.go` constructs it from the global registry dynamically, eliminating the hardcoded metric enum.
3. `BuildHandler` returns an explicit function type `func(context.Context, *sdkmcp.CallToolRequest, QueryInput) (*sdkmcp.CallToolResult, any, error)` rather than `sdkmcp.ToolHandlerFor[QueryInput, any]`, avoiding ambiguity between the SDK's internal low-level type and the convenience form accepted by `AddTool`.

---

## File Structure

| Action | Path | Responsibility |
|--------|------|---------------|
| Create | `internal/runner/runner.go` | `RunMetric` + unexported helpers moved from `meta_processing.go` |
| Create | `internal/runner/runner_test.go` | Migrated helper tests + new `RunMetric` tests |
| Create | `internal/mcp/handler.go` | `QueryInput` struct + `BuildHandler` factory |
| Create | `internal/mcp/handler_test.go` | Handler unit tests via fake provider |
| Create | `internal/mcp/server.go` | `Run(ctx, cfg)` — server init, transport, signal handling |
| Create | `cmd/cryptospect-cli/mcp.go` | `newMCPCommand()` cobra subcommand |
| Modify | `internal/output/envelope.go` | Add `NewSuccessResponse` + `NewErrorResponse` constructors |
| Modify | `cmd/cryptospect-cli/root.go` | Refactor `buildMetricRunE` + wire `newMCPCommand` |
| Modify | `go.mod` / `go.sum` | Add `github.com/modelcontextprotocol/go-sdk` |
| Delete | `cmd/cryptospect-cli/meta_processing.go` | Logic moved to `internal/runner/runner.go` |
| Delete | `cmd/cryptospect-cli/meta_processing_test.go` | Tests moved to `internal/runner/runner_test.go` |

---

## Task 1: Add go-sdk dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd /project/cryptospect-cli && go get github.com/modelcontextprotocol/go-sdk@v1.2.0
```

Expected: `go.mod` and `go.sum` updated. `go get` exits 0.

- [ ] **Step 2: Verify it resolved**

```bash
grep "modelcontextprotocol/go-sdk" go.mod
```

Expected: a line like `github.com/modelcontextprotocol/go-sdk v1.2.0`.

---

## Task 2: Add response constructors to `internal/output/envelope.go`

**Files:**
- Modify: `internal/output/envelope.go`

Centralises `CLIResponse` construction so CLI and MCP handler share one source of truth for envelope semantics. Both `WriteSuccess` and `WriteError` in `writer.go` are updated to delegate to these helpers so no existing caller changes are needed.

- [ ] **Step 1: Add constructors at the bottom of `envelope.go`**

```go
// NewSuccessResponse builds the ok CLIResponse envelope.
// Use this instead of constructing CLIResponse literals directly.
func NewSuccessResponse(results []MetricResult) CLIResponse {
	return CLIResponse{
		Status:  "ok",
		TS:      time.Now().Unix(),
		Results: results,
	}
}

// NewErrorResponse builds the error CLIResponse envelope.
// Use this instead of constructing CLIResponse literals directly.
func NewErrorResponse(err CLIError) CLIResponse {
	return CLIResponse{
		Status: "error",
		TS:     time.Now().Unix(),
		Error:  &err,
	}
}
```

Add `"time"` to the import block in `envelope.go` (it is not there yet).

- [ ] **Step 2: Update `WriteSuccess` and `WriteError` in `writer.go` to delegate**

```go
func WriteSuccess(results []MetricResult) error {
	resp := NewSuccessResponse(results)
	// ... encodeJSON + write unchanged
}

func WriteError(code int, msg, source string, retryAfterSec int) error {
	resp := NewErrorResponse(CLIError{
		Code:          code,
		Msg:           msg,
		Source:        source,
		RetryAfterSec: retryAfterSec,
	})
	// ... encodeJSON + write unchanged
}
```

- [ ] **Step 3: Verify existing tests still pass**

```bash
cd /project/cryptospect-cli && go test -race -count=1 ./internal/output/... 2>&1 | tail -10
```

Expected: all tests PASS, no behaviour change.

---

## Task 3: Write `internal/runner/runner_test.go` (failing — runner.go does not exist yet)

**Files:**
- Create: `internal/runner/runner_test.go`

- [ ] **Step 1: Create the file**

```go
// internal/runner/runner_test.go
package runner

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// fakeProvider is a test-only MetricProvider with configurable endpoints,
// meta output, and optional compute error. Pass nil Fetcher to RunMetric
// when endpoints is nil — Fetch is never called.
type fakeProvider struct {
	name       string
	endpoints  []string
	meta       json.RawMessage
	computeErr error
}

func (p fakeProvider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      p.name,
		Namespace: metrics.CoreNamespace,
		Version:   "v1.0.0",
		Endpoints: p.endpoints,
	}
}

func (p fakeProvider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
	if p.computeErr != nil {
		return output.MetricResult{}, p.computeErr
	}
	return output.MetricResult{
		Metric:    p.name,
		Version:   "v1.0.0",
		Namespace: metrics.CoreNamespace,
		Status:    "ok",
		Data:      json.RawMessage(`{"v":1}`),
		Meta:      p.meta,
	}, nil
}

// ── aggregateFetchMeta ────────────────────────────────────────────────────

func TestAggregateFetchMeta_EmptyMap(t *testing.T) {
	hit, ttl := aggregateFetchMeta(map[string]api.FetchMeta{})
	if hit {
		t.Error("empty map: cache_hit must be false")
	}
	if ttl != 0 {
		t.Errorf("empty map: ttl_remaining must be 0, got %d", ttl)
	}
}

func TestAggregateFetchMeta_SingleCacheHit(t *testing.T) {
	metas := map[string]api.FetchMeta{"ep1": {CacheHit: true, TTLRemaining: 900}}
	hit, ttl := aggregateFetchMeta(metas)
	if !hit {
		t.Error("single cache hit: cache_hit must be true")
	}
	if ttl != 900 {
		t.Errorf("single cache hit: ttl_remaining = %d, want 900", ttl)
	}
}

func TestAggregateFetchMeta_SingleMiss(t *testing.T) {
	metas := map[string]api.FetchMeta{"ep1": {CacheHit: false, TTLRemaining: 3600}}
	hit, ttl := aggregateFetchMeta(metas)
	if hit {
		t.Error("single miss: cache_hit must be false")
	}
	if ttl != 0 {
		t.Errorf("single miss: ttl_remaining must be 0, got %d", ttl)
	}
}

func TestAggregateFetchMeta_AllHits_MinTTL(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: true, TTLRemaining: 1800},
		"ep2": {CacheHit: true, TTLRemaining: 600},
		"ep3": {CacheHit: true, TTLRemaining: 3200},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if !hit {
		t.Error("all hits: cache_hit must be true")
	}
	if ttl != 600 {
		t.Errorf("all hits: ttl_remaining = %d, want min=600", ttl)
	}
}

func TestAggregateFetchMeta_MixedHitMiss(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: true, TTLRemaining: 1200},
		"ep2": {CacheHit: false, TTLRemaining: 3600},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if hit {
		t.Error("mixed: cache_hit must be false when any endpoint is a miss")
	}
	if ttl != 0 {
		t.Errorf("mixed: ttl_remaining must be 0, got %d", ttl)
	}
}

func TestAggregateFetchMeta_NegativeTTL_ClampedToZero(t *testing.T) {
	metas := map[string]api.FetchMeta{"ep1": {CacheHit: true, TTLRemaining: -5}}
	hit, ttl := aggregateFetchMeta(metas)
	if !hit {
		t.Error("negative TTL: cache_hit should still be true")
	}
	if ttl != 0 {
		t.Errorf("negative TTL: ttl_remaining must be clamped to 0, got %d", ttl)
	}
}

// ── postProcessMeta ───────────────────────────────────────────────────────

var testFullOnlyFields = []string{"thresholds", "description", "top_n_stablecoins", "tier_detail"}

func TestPostProcessMeta_NilInput(t *testing.T) {
	result := postProcessMeta(nil, "extended", false, 0, testFullOnlyFields)
	if result != nil {
		t.Error("nil input must return nil")
	}
}

func TestPostProcessMeta_InvalidJSON_PassThrough(t *testing.T) {
	bad := json.RawMessage(`not-json`)
	result := postProcessMeta(bad, "extended", true, 300, testFullOnlyFields)
	if string(result) != string(bad) {
		t.Error("invalid JSON must be returned unchanged")
	}
}

func TestPostProcessMeta_Extended_InjectsCacheFields(t *testing.T) {
	input := json.RawMessage(`{"primary_source":"coingecko","confidence":"high"}`)
	result := postProcessMeta(input, "extended", true, 720, testFullOnlyFields)
	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if v, ok := meta["cache_hit"].(bool); !ok || !v {
		t.Errorf("cache_hit: got %v, want true", meta["cache_hit"])
	}
	if v, ok := meta["ttl_remaining_sec"].(float64); !ok || v != 720 {
		t.Errorf("ttl_remaining_sec: got %v, want 720", meta["ttl_remaining_sec"])
	}
	if meta["primary_source"] != "coingecko" {
		t.Error("existing fields must be preserved")
	}
}

func TestPostProcessMeta_Extended_StripsFullOnlyFields(t *testing.T) {
	input := json.RawMessage(`{"confidence":"high","thresholds":{"high":0.15},"description":"long text","top_n_stablecoins":[],"tier_detail":{}}`)
	result := postProcessMeta(input, "extended", false, 0, testFullOnlyFields)
	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	for _, f := range testFullOnlyFields {
		if _, ok := meta[f]; ok {
			t.Errorf("extended: field %q must be stripped", f)
		}
	}
	if meta["confidence"] != "high" {
		t.Error("non-full-only fields must be preserved")
	}
}

func TestPostProcessMeta_Full_KeepsFullOnlyFields(t *testing.T) {
	input := json.RawMessage(`{"confidence":"high","thresholds":{"high":0.15},"description":"long text","tier_detail":{}}`)
	result := postProcessMeta(input, "full", false, 0, testFullOnlyFields)
	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := meta["thresholds"]; !ok {
		t.Error("full: thresholds must be present")
	}
	if _, ok := meta["description"]; !ok {
		t.Error("full: description must be present")
	}
	if _, ok := meta["tier_detail"]; !ok {
		t.Error("full: tier_detail must be present")
	}
}

func TestPostProcessMeta_OverwritesExistingCacheFields(t *testing.T) {
	input := json.RawMessage(`{"cache_hit":true,"ttl_remaining_sec":9999,"confidence":"high"}`)
	result := postProcessMeta(input, "full", false, 0, testFullOnlyFields)
	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if v, _ := meta["cache_hit"].(bool); v {
		t.Error("cache_hit must be overwritten to false")
	}
	if v, _ := meta["ttl_remaining_sec"].(float64); v != 0 {
		t.Errorf("ttl_remaining_sec must be overwritten to 0, got %v", v)
	}
}

func TestPostProcessMeta_EmptyMeta_InjectsCacheFields(t *testing.T) {
	result := postProcessMeta(json.RawMessage(`{}`), "extended", false, 0, testFullOnlyFields)
	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := meta["cache_hit"]; !ok {
		t.Error("cache_hit must be present even in empty meta")
	}
	if _, ok := meta["ttl_remaining_sec"]; !ok {
		t.Error("ttl_remaining_sec must be present even in empty meta")
	}
}

// ── RunMetric ─────────────────────────────────────────────────────────────

func TestRunMetric_DetailBasic_MetaNil(t *testing.T) {
	p := fakeProvider{name: "fake", meta: json.RawMessage(`{"confidence":"high","thresholds":{"a":1}}`)}
	result, err := RunMetric(context.Background(), p, nil, "basic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta != nil {
		t.Errorf("detail=basic: Meta must be nil, got %s", result.Meta)
	}
}

func TestRunMetric_DetailExtended_MetaPresent_FullOnlyStripped(t *testing.T) {
	p := fakeProvider{name: "fake", meta: json.RawMessage(`{"confidence":"high","thresholds":{"a":1},"description":"d"}`)}
	result, err := RunMetric(context.Background(), p, nil, "extended")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Meta) == 0 {
		t.Fatal("detail=extended: Meta must not be nil")
	}
	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if _, ok := meta["thresholds"]; ok {
		t.Error("extended: thresholds must be stripped")
	}
	if _, ok := meta["cache_hit"]; !ok {
		t.Error("extended: cache_hit must be injected")
	}
}

func TestRunMetric_DetailFull_MetaPresent_FullOnlyKept(t *testing.T) {
	p := fakeProvider{name: "fake", meta: json.RawMessage(`{"confidence":"high","thresholds":{"a":1},"description":"d"}`)}
	result, err := RunMetric(context.Background(), p, nil, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if _, ok := meta["thresholds"]; !ok {
		t.Error("full: thresholds must be kept")
	}
	if _, ok := meta["description"]; !ok {
		t.Error("full: description must be kept")
	}
}

func TestRunMetric_ComputeError_Propagates(t *testing.T) {
	p := fakeProvider{name: "fake", computeErr: errors.New("catastrophic failure")}
	_, err := RunMetric(context.Background(), p, nil, "basic")
	if err == nil {
		t.Fatal("expected error from Compute, got nil")
	}
}

func TestRunMetric_NoEndpoints_CacheHitFalse(t *testing.T) {
	// No endpoints → Fetcher never called (nil is safe) → fetchMetas empty →
	// aggregateFetchMeta returns (false, 0).
	p := fakeProvider{name: "fake", meta: json.RawMessage(`{"confidence":"high"}`)}
	result, err := RunMetric(context.Background(), p, nil, "extended")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(result.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if v, _ := meta["cache_hit"].(bool); v {
		t.Error("no endpoints: cache_hit must be false")
	}
	if v, _ := meta["ttl_remaining_sec"].(float64); v != 0 {
		t.Errorf("no endpoints: ttl_remaining_sec must be 0, got %v", v)
	}
}
```

- [ ] **Step 2: Verify the file fails to compile (runner.go doesn't exist yet)**

```bash
cd /project/cryptospect-cli && go build ./internal/runner/... 2>&1 | head -5
```

Expected: build error — `no Go files in internal/runner` or similar.

---

## Task 4: Create `internal/runner/runner.go`

**Files:**
- Create: `internal/runner/runner.go`

- [ ] **Step 1: Create the file**

```go
// internal/runner/runner.go
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// Fetcher is the minimal interface RunMetric needs from the HTTP/cache layer.
// *api.Fetcher satisfies it; a future KV or cloud adapter only needs to implement Fetch.
// TODO: KV adapter — implement Fetcher backed by Cloudflare KV or similar.
type Fetcher interface {
	Fetch(ctx context.Context, endpointKey string) ([]byte, api.FetchMeta, error)
}

// fullDetailOnlyFields lists meta keys stripped at --detail extended.
// Any new full-detail-only meta field must be added here.
var fullDetailOnlyFields = []string{
	"thresholds",
	"description",
	"top_n_stablecoins",
	"tier_detail",
}

// RunMetric fetches endpoint data for p, calls Compute, and post-processes
// the result meta to the requested detail level. It is the shared execution
// pipeline for both the CLI command runner and the MCP handler.
//
// f may be nil only when p.Def().Endpoints is empty — Fetch is never called in
// that case. Passing nil for a provider with endpoints returns an error rather
// than panicking, to catch test misuse early.
func RunMetric(
	ctx context.Context,
	p metrics.MetricProvider,
	f Fetcher,
	detail string,
) (output.MetricResult, error) {
	def := p.Def()
	if f == nil && len(def.Endpoints) > 0 {
		return output.MetricResult{}, fmt.Errorf("RunMetric: nil fetcher for provider %q which requires %d endpoint(s)", def.Name, len(def.Endpoints))
	}
	data := make(map[string]json.RawMessage)
	fetchMetas := make(map[string]api.FetchMeta)

	for _, key := range def.Endpoints {
		raw, meta, err := f.Fetch(ctx, key)
		if err != nil {
			slog.Debug("endpoint fetch failed, using unavailable", "endpoint", key, "error", err)
			data[key] = nil
			continue
		}
		data[key] = raw
		fetchMetas[key] = meta
	}

	result, err := p.Compute(ctx, data)
	if err != nil {
		return output.MetricResult{}, err
	}

	cacheHit, ttlRemaining := aggregateFetchMeta(fetchMetas)
	switch detail {
	case "basic":
		result.Meta = nil
	default:
		result.Meta = postProcessMeta(result.Meta, detail, cacheHit, ttlRemaining, fullDetailOnlyFields)
	}

	return result, nil
}

func aggregateFetchMeta(metas map[string]api.FetchMeta) (cacheHit bool, ttlRemaining int) {
	if len(metas) == 0 {
		return false, 0
	}
	for _, m := range metas {
		if !m.CacheHit {
			return false, 0
		}
	}
	first := true
	minTTL := 0
	for _, m := range metas {
		if first || m.TTLRemaining < minTTL {
			minTTL = m.TTLRemaining
			first = false
		}
	}
	if minTTL < 0 {
		minTTL = 0
	}
	return true, minTTL
}

func postProcessMeta(metaJSON json.RawMessage, detail string, cacheHit bool, ttlRemaining int, fullOnlyFields []string) json.RawMessage {
	if metaJSON == nil {
		return nil
	}
	var meta map[string]any
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return metaJSON
	}
	meta["cache_hit"] = cacheHit
	meta["ttl_remaining_sec"] = ttlRemaining
	if detail == "extended" {
		for _, f := range fullOnlyFields {
			delete(meta, f)
		}
	}
	out, err := json.Marshal(meta)
	if err != nil {
		return metaJSON
	}
	return out
}
```

---

## Task 5: Run runner tests — all must pass

**Files:** none

- [ ] **Step 1: Run the runner tests**

```bash
cd /project/cryptospect-cli && go test -race -count=1 ./internal/runner/... -v 2>&1 | tail -20
```

Expected: all tests PASS. No race conditions. If any test fails, fix `runner.go` before continuing.

---

## Task 6: Refactor `root.go` + delete `meta_processing` files

**Files:**
- Modify: `cmd/cryptospect-cli/root.go`
- Delete: `cmd/cryptospect-cli/meta_processing.go`
- Delete: `cmd/cryptospect-cli/meta_processing_test.go`

- [ ] **Step 1: Replace `buildMetricRunE` in `root.go`**

Find the function starting at line 154 and replace it entirely with:

```go
func buildMetricRunE(p metrics.MetricProvider) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cfg, ok := config.FromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("config not found in context")
		}

		ctx := cmd.Context()
		if f := cmd.Flags().Lookup("top"); f != nil {
			if n, err := strconv.Atoi(f.Value.String()); err == nil {
				ctx = config.StoreTopNInContext(ctx, n)
			}
		}
		if f := cmd.Flags().Lookup("segments"); f != nil {
			ctx = config.StoreSegmentsInContext(ctx, f.Value.String())
		}

		fetcher, err := api.New(cfg.CacheDir(), &cfg)
		if err != nil {
			return fmt.Errorf("creating fetcher: %w", err)
		}

		detailLevel, _ := config.DetailFromContext(ctx)
		// *api.Fetcher satisfies runner.Fetcher — no cast needed.
		result, err := runner.RunMetric(ctx, p, fetcher, detailLevel)
		if err != nil {
			return err
		}

		return output.WriteSuccess([]output.MetricResult{result})
	}
}
```

- [ ] **Step 2: Update imports in `root.go`**

Remove `"encoding/json"` from the import block (no longer used).

Add `"github.com/afshinator/cryptospect-cli/internal/runner"` to the internal imports group.

The import block should look like:

```go
import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/afshinator/cryptospect-cli/internal/runner"
	"github.com/afshinator/cryptospect-cli/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)
```

- [ ] **Step 3: Delete the moved files**

```bash
rm /project/cryptospect-cli/cmd/cryptospect-cli/meta_processing.go
rm /project/cryptospect-cli/cmd/cryptospect-cli/meta_processing_test.go
```

---

## Task 7: Run full test suite — refactor must be behaviour-neutral

**Files:** none

- [ ] **Step 1: Run all tests**

```bash
cd /project/cryptospect-cli && go test -race -count=1 ./... 2>&1 | tail -30
```

Expected: all packages pass. The tests previously in `meta_processing_test.go` now live in `internal/runner/runner_test.go` and pass there. Net test count is preserved.

- [ ] **Step 2: Check build compiles cleanly**

```bash
cd /project/cryptospect-cli && go build ./... 2>&1
```

Expected: no output (clean build).

---

## Task 8: Commit the runner extraction checkpoint

**Files:** none

- [ ] **Step 1: Stage and commit**

```bash
cd /project/cryptospect-cli
git add internal/runner/ cmd/cryptospect-cli/root.go cmd/cryptospect-cli/meta_processing.go cmd/cryptospect-cli/meta_processing_test.go go.mod go.sum
git commit -m "refactor: extract metric runner pipeline to internal/runner

Moves fetch→compute→post-process logic from package main into
internal/runner.RunMetric, shared by CLI and the upcoming MCP handler.
Deletes meta_processing.go; tests migrated to internal/runner/runner_test.go."
```

---

## Task 9: Write `internal/mcp/handler_test.go` (failing — handler.go does not exist yet)

**Files:**
- Create: `internal/mcp/handler_test.go`

**Background:** `fakeProvider` registers itself via `init()` into the global registry. Because this test binary only imports `internal/mcp` (no metric blank-imports), only `fakeProvider` is registered. The `all` branch therefore returns 1 result in unit tests — that is expected and correct. The fake provider has no endpoints so the `*api.Fetcher` is never called; `nil` is safe to pass.

- [ ] **Step 1: Create the file**

```go
// internal/mcp/handler_test.go
package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/afshinator/cryptospect-cli/internal/config"
	internalmcp "github.com/afshinator/cryptospect-cli/internal/mcp"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/afshinator/cryptospect-cli/internal/runner"
)

// fakeProvider is a no-endpoint metric registered once at test binary startup.
type fakeProvider struct{}

func (fakeProvider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      "test-metric",
		Namespace: metrics.CoreNamespace,
		Version:   "v1.0.0",
		Endpoints: nil,
	}
}

func (fakeProvider) Compute(_ context.Context, _ map[string]json.RawMessage) (output.MetricResult, error) {
	return output.MetricResult{
		Metric:    "test-metric",
		Version:   "v1.0.0",
		Namespace: metrics.CoreNamespace,
		Status:    "ok",
		Data:      json.RawMessage(`{"score":42}`),
		Meta:      json.RawMessage(`{"confidence":"high","thresholds":{"a":1},"description":"test"}`),
	}, nil
}

func init() {
	metrics.MustRegister(fakeProvider{})
}

// callHandler invokes BuildHandler directly — no MCP server needed.
// nil is passed as runner.Fetcher; safe because fakeProvider has no endpoints.
func callHandler(t *testing.T, metric, detail string) *sdkmcp.CallToolResult {
	t.Helper()
	h := internalmcp.BuildHandler(config.Config{}, nil)
	result, _, err := h(context.Background(), nil, internalmcp.QueryInput{
		Metric: metric,
		Detail: detail,
	})
	if err != nil {
		t.Fatalf("handler returned unexpected Go error: %v", err)
	}
	return result
}

func parseResponse(t *testing.T, result *sdkmcp.CallToolResult) output.CLIResponse {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("CallToolResult.Content is empty")
	}
	tc, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected *sdkmcp.TextContent, got %T", result.Content[0])
	}
	var resp output.CLIResponse
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal CLIResponse: %v\nraw: %s", err, tc.Text)
	}
	return resp
}

func TestBuildHandler_UnknownMetric_IsError(t *testing.T) {
	result := callHandler(t, "does-not-exist", "")
	if !result.IsError {
		t.Error("IsError must be true for unknown metric")
	}
	resp := parseResponse(t, result)
	if resp.Status != "error" {
		t.Errorf("status = %q, want %q", resp.Status, "error")
	}
	if resp.Error == nil || resp.Error.Code != 400 {
		t.Errorf("error.code = %v, want 400", resp.Error)
	}
}

func TestBuildHandler_ValidMetric_OK(t *testing.T) {
	result := callHandler(t, "test-metric", "basic")
	if result.IsError {
		t.Error("IsError must be false for valid metric")
	}
	resp := parseResponse(t, result)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Metric != "test-metric" {
		t.Errorf("metric = %q, want %q", resp.Results[0].Metric, "test-metric")
	}
}

func TestBuildHandler_DetailBasic_MetaNil(t *testing.T) {
	resp := parseResponse(t, callHandler(t, "test-metric", "basic"))
	if resp.Results[0].Meta != nil {
		t.Error("detail=basic: Meta must be nil")
	}
}

func TestBuildHandler_DetailFull_MetaPresent(t *testing.T) {
	resp := parseResponse(t, callHandler(t, "test-metric", "full"))
	if len(resp.Results[0].Meta) == 0 {
		t.Error("detail=full: Meta must not be nil")
	}
}

func TestBuildHandler_EmptyDetail_DefaultsToBasic(t *testing.T) {
	// Empty detail string must default to "basic" → Meta is nil.
	resp := parseResponse(t, callHandler(t, "test-metric", ""))
	if resp.Results[0].Meta != nil {
		t.Error("empty detail must default to basic: Meta must be nil")
	}
}

func TestBuildHandler_AllMetric_ReturnsResults(t *testing.T) {
	result := callHandler(t, "all", "basic")
	if result.IsError {
		t.Error("IsError must be false for 'all'")
	}
	resp := parseResponse(t, result)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if len(resp.Results) == 0 {
		t.Error("all: must return at least 1 result")
	}
}
```

- [ ] **Step 2: Verify the file fails to compile**

```bash
cd /project/cryptospect-cli && go build ./internal/mcp/... 2>&1 | head -5
```

Expected: compile error — `no Go files in internal/mcp` or `cannot find package`.

---

## Task 10: Create `internal/mcp/handler.go`

**Files:**
- Create: `internal/mcp/handler.go`

- [ ] **Step 1: Create the file**

```go
// internal/mcp/handler.go
package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/afshinator/cryptospect-cli/internal/runner"
)

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
// f accepts runner.Fetcher (satisfied by *api.Fetcher). Pass nil in tests
// where all registered providers have no endpoints.
func BuildHandler(cfg config.Config, f runner.Fetcher) func(context.Context, *sdkmcp.CallToolRequest, QueryInput) (*sdkmcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, in QueryInput) (*sdkmcp.CallToolResult, any, error) {
		detail := in.Detail
		if detail == "" {
			detail = "basic"
		}

		ctx = config.StoreInContext(ctx, cfg)
		ctx = config.StoreDetailInContext(ctx, detail)

		if in.Metric == "all" {
			var results []output.MetricResult
			for _, p := range metrics.GlobalRegistry().BestProviders() {
				result, err := runner.RunMetric(ctx, p, f, detail)
				if err != nil {
					def := p.Def()
					result, _ = metrics.UnavailableResult(def.Name, def.Version, def.Namespace, err.Error())
				}
				results = append(results, result)
			}
			return successResult(results), nil, nil
		}

		p, err := metrics.GlobalRegistry().Resolve(in.Metric)
		if err != nil {
			return errorResult(400, "unknown_metric"), nil, nil
		}

		result, err := runner.RunMetric(ctx, p, f, detail)
		if err != nil {
			return errorResult(500, "compute_error"), nil, nil
		}

		return successResult([]output.MetricResult{result}), nil, nil
	}
}

func successResult(results []output.MetricResult) *sdkmcp.CallToolResult {
	data, _ := json.Marshal(output.NewSuccessResponse(results))
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
	}
}

func errorResult(code int, msg string) *sdkmcp.CallToolResult {
	data, _ := json.Marshal(output.NewErrorResponse(output.CLIError{
		Code:   code,
		Msg:    msg,
		Source: "mcp",
	}))
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		IsError: true,
	}
}
```

---

## Task 11: Run handler tests — all must pass

**Files:** none

- [ ] **Step 1: Run handler tests**

```bash
cd /project/cryptospect-cli && go test -race -count=1 ./internal/mcp/... -v 2>&1 | tail -20
```

Expected: all 6 tests PASS. If any fail, fix `handler.go` before continuing.

---

## Task 12: Create `internal/mcp/server.go`

**Files:**
- Create: `internal/mcp/server.go`

- [ ] **Step 1: Create the file**

```go
// internal/mcp/server.go
package mcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp/jsonschema"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/afshinator/cryptospect-cli/internal/version"
)

// buildToolSchema constructs the JSON Schema for query_market_analytics dynamically
// from the global registry so the metric enum stays in sync without a hardcoded list.
func buildToolSchema() *jsonschema.Schema {
	providers := metrics.GlobalRegistry().BestProviders()
	enum := make([]any, 0, len(providers)+1)
	for _, p := range providers {
		enum = append(enum, p.Def().Name)
	}
	enum = append(enum, "all")
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"metric": {
				Type:        "string",
				Enum:        enum,
				Description: "Which metric to compute. 'all' runs every metric and returns all results in a single CLIResponse envelope.",
			},
			"detail": {
				Type:        "string",
				Enum:        []any{"basic", "extended", "full"},
				Description: "Controls output verbosity. Mirrors the CLI --detail flag.",
			},
		},
		Required: []string{"metric"},
	}
}

// Run starts the MCP server on stdio and blocks until ctx is cancelled or
// the client disconnects.
func Run(ctx context.Context, cfg config.Config) error {
	// Stdout guard: the MCP transport owns os.Stdout exclusively for JSON-RPC
	// framing. Belt-and-suspenders protection against any stray write to stdout:
	//   - output.SetWriter(io.Discard) covers output.WriteSuccess / WriteError.
	//   - log.SetOutput(os.Stderr) covers the standard log package (default is
	//     already stderr, but explicit is safer).
	//   - slog is already on stderr: PersistentPreRunE runs before this function
	//     and calls slog.SetDefault with a stderr handler.
	//   - panic traces always go to stderr — no action needed.
	output.SetWriter(io.Discard)
	log.SetOutput(os.Stderr)

	fetcher, err := api.New(cfg.CacheDir(), &cfg)
	if err != nil {
		return fmt.Errorf("creating fetcher: %w", err)
	}

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "cryptospect-cli",
		Version: version.String(),
	}, nil)

	sdkmcp.AddTool[QueryInput, any](server, &sdkmcp.Tool{
		Name: "query_market_analytics",
		Description: "Fetches computed crypto market regime signals. Returns agent-optimized JSON. " +
			"Use detail=full when feeding output to an LLM — it includes metric descriptions and thresholds. " +
			"Use detail=basic for lightweight loops.",
		InputSchema: buildToolSchema(),
	}, BuildHandler(cfg, fetcher))

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return server.Run(ctx, &sdkmcp.StdioTransport{})
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /project/cryptospect-cli && go build ./internal/mcp/... 2>&1
```

Expected: no output.

---

## Task 13: Create `cmd/cryptospect-cli/mcp.go` and wire into `root.go`

**Files:**
- Create: `cmd/cryptospect-cli/mcp.go`
- Modify: `cmd/cryptospect-cli/root.go`

- [ ] **Step 1: Create `mcp.go`**

```go
// cmd/cryptospect-cli/mcp.go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/afshinator/cryptospect-cli/internal/config"
	mcpserver "github.com/afshinator/cryptospect-cli/internal/mcp"
)

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

- [ ] **Step 2: Add `newMCPCommand` to `NewRootCommand()` in `root.go`**

In `NewRootCommand()`, find the block that adds `listCmd` and `cacheCmd` and add after it:

```go
cmd.AddCommand(newMCPCommand())
```

The surrounding context looks like:

```go
	listCmd := newListCommand()
	listCmd.GroupID = "utility"
	cmd.AddCommand(listCmd)

	cacheCmd := newCacheClearCommand()
	cacheCmd.GroupID = "utility"
	cmd.AddCommand(cacheCmd)

	cmd.AddCommand(newMCPCommand()) // ← add this line
```

---

## Task 14: Build binary and smoke-test

**Files:** none

- [ ] **Step 1: Build**

```bash
cd /project/cryptospect-cli && go build -o bin/cryptospect-cli ./cmd/cryptospect-cli/ 2>&1
```

Expected: exits 0, `bin/cryptospect-cli` created.

- [ ] **Step 2: Verify `mcp` appears in help**

```bash
./bin/cryptospect-cli --help 2>&1 | grep -A5 "Utility"
```

Expected: output includes `mcp` alongside `list` and `cache-clear` under the Utility section.

- [ ] **Step 3: Verify `mcp --help` shows the long description**

```bash
./bin/cryptospect-cli mcp --help 2>&1
```

Expected: shows "Start a Model Context Protocol server over stdio." and mentions `query_market_analytics`.

---

## Task 15: Full test suite, lint, and final commit

**Files:** none

- [ ] **Step 1: Run all tests with race detector**

```bash
cd /project/cryptospect-cli && go test -race -count=1 ./... 2>&1 | tail -30
```

Expected: all packages pass, 0 failures, 0 races.

- [ ] **Step 2: Run linter**

```bash
cd /project/cryptospect-cli && golangci-lint run ./... 2>&1
```

Expected: no issues. If lint flags unused imports or missing error checks, fix before committing.

- [ ] **Step 3: Run formatter**

```bash
cd /project/cryptospect-cli && gofumpt -w ./... && goimports -w ./... 2>&1
```

Expected: no output. Re-run tests if formatter changes any files.

- [ ] **Step 4: Commit**

```bash
cd /project/cryptospect-cli
git add internal/mcp/ cmd/cryptospect-cli/mcp.go cmd/cryptospect-cli/root.go
git commit -m "feat: add MCP server (cryptospect-cli mcp)

Exposes query_market_analytics tool over stdio MCP transport.
Single tool, detail per call, CLIResponse envelope, IsError on failures.
Uses github.com/modelcontextprotocol/go-sdk v1.2.0."
```
