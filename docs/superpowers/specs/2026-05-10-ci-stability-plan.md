# CI Stability Fix Plan — 2026-05-10

## Problem

The `cmd/cryptospect-cli` E2E tests make live API calls (no httptest mocks). When CI's IP
is rate-limited by CoinGecko, four tests assert the envelope must be `"ok"` and fail.
This has happened multiple times. Root-cause analysis confirmed three structural gaps.

## Root Cause Summary

### Gap 1 — Flaky E2E assertions (immediate CI blocker)

Five envelope-status assertions hard-require `resp.Status == "ok"` on live API calls.
Two additional `Meta == nil` checks in liquidity_pulse are ungated.

Only `momentum_divergence_e2e_test.go` is written correctly (accepts `"ok" || "error"`
and gates all data assertions on result status).

### Gap 2 — CI installs `@latest` formatters (recurring time bomb)

`ci.yml:24–25` runs `go install mvdan.cc/gofumpt@latest` and
`go install golang.org/x/tools/cmd/goimports@latest`. Any new release that changes
formatting rules breaks CI even when local checks pass. Already caused `ec74a6b`
("fix: trailing commas for gofumpt v0.10.0").

### Gap 3 — No pre-push hook (no enforcement gate)

`.git/hooks/pre-push` does not exist. CLAUDE.md documents "required before every push"
but it is convention, not enforcement. Multiple agents (claude-code, opencode) push
without running checks.

### Gap 4 — graphify-out committed to main (repo pollution)

The May 9 opencode session committed 143 generated files (743KB+ JSON). The `.gitignore`
fix is unstaged and only partially excludes the directory.

---

## Task Groups

### T1 — Fix flaky E2E test assertions (unblocks CI immediately)

**Files:**
- `cmd/cryptospect-cli/flow_tension_e2e_test.go`
- `cmd/cryptospect-cli/liquidity_pulse_e2e_test.go`
- `cmd/cryptospect-cli/market_breadth_e2e_test.go`
- `cmd/cryptospect-cli/stablecoin_power_e2e_test.go`

**Changes:**

T1a — 5 envelope assertions: change `resp.Status != "ok"` to
`resp.Status != "ok" && resp.Status != "error"` with updated error message.

| File | Line | Test |
|---|---|---|
| `flow_tension_e2e_test.go` | 28 | `TestFlowTensionCommand` |
| `liquidity_pulse_e2e_test.go` | 28 | `TestLiquidityPulseCommand` |
| `liquidity_pulse_e2e_test.go` | 103 | `TestLiquidityPulseOutputJSON` |
| `market_breadth_e2e_test.go` | 32 | `TestMarketBreadthCommand` |
| `stablecoin_power_e2e_test.go` | 28 | `TestStablecoinPowerCommand` |

T1b — 2 ungated meta checks in liquidity_pulse:
- `liquidity_pulse_e2e_test.go:125–127` (`TestLiquidityPulseDetailExtended`)
- `liquidity_pulse_e2e_test.go:147–149` (`TestLiquidityPulseDetailFull`)

Change from:
```go
if resp.Results[0].Meta == nil {
    t.Error("Meta should be present for extended detail")
}
```
To (matching pattern in flow_tension, market_breadth, stablecoin_power):
```go
st := resp.Results[0].Status
if (st == "ok" || st == "degraded") && resp.Results[0].Meta == nil {
    t.Error("Meta should be present for extended detail when data is available")
}
```

**Verification:** `go test -race -count=1 ./cmd/cryptospect-cli/` must pass.

---

### T2 — Pin formatter versions in CI

**File:** `.github/workflows/ci.yml`

Change:
```yaml
go install golang.org/x/tools/cmd/goimports@latest
go install mvdan.cc/gofumpt@latest
```
To:
```yaml
go install golang.org/x/tools/cmd/goimports@v0.31.0
go install mvdan.cc/gofumpt@v0.10.0
```

> Note: verify the exact goimports version with `goimports --version` or check the
> installed binary. Use whatever is current in the container.

---

### T3 — Add pre-push hook

**File:** `.git/hooks/pre-push` (not git-tracked; installed in local clone)

```sh
#!/bin/sh
set -e
echo "pre-push: fmt check..."
UNFORMATTED=$(find . -name "*.go" \
  -not -path "./.cache/*" -not -path "./.gomodcache/*" \
  -not -path "./.go/*" -not -path "./vendor/*" \
  -exec gofumpt -l {} \;)
if [ -n "$UNFORMATTED" ]; then
  echo "pre-push: gofumpt would reformat: $UNFORMATTED"
  echo "Run: make fmt"
  exit 1
fi
echo "pre-push: lint..."
golangci-lint run ./...
echo "pre-push: tests..."
go test -race ./...
echo "pre-push: all checks passed."
```

Make executable: `chmod +x .git/hooks/pre-push`

---

### T4 — Fix .gitignore and untrack graphify-out

**File:** `.gitignore`

Replace the partial graphify exclusion block with:
```gitignore
# Graphify local-only state and cache
graphify-out/
```

Then untrack the already-committed files (without deleting them locally):
```bash
git rm -r --cached graphify-out/
```

Stage and commit together with the `.gitignore` change.

---

## Implementation Order

1. **T1** first — unblocks CI, smallest risk, pure test changes
2. **T2** — prevents future formatter drift, CI config change
3. **T4** — cleans up repo pollution, removes tracked generated files
4. **T3** — local enforcement gate, not committed (hook files aren't tracked)

T1 + T2 should go in a single commit. T4 in a separate commit. T3 is local-only.
