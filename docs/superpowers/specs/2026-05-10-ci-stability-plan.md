# CI Stability Fix Plan — 2026-05-10

Updated: 2026-05-11 (T1 complete, T2–T4 validated)

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

**Status: ✅ Fixed (2026-05-11).** See Progress section below.

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
- `cmd/cryptospect-cli/momentum_divergence_e2e_test.go` (review scope creep)

**Changes implemented:**

T1a — 5 envelope assertions: change `resp.Status != "ok"` to
`resp.Status != "ok" && resp.Status != "error"` with updated error message.

| File | Line (original) | Test |
|---|---|---|
| `flow_tension_e2e_test.go` | 28 | `TestFlowTensionCommand` |
| `liquidity_pulse_e2e_test.go` | 28 | `TestLiquidityPulseCommand` |
| `liquidity_pulse_e2e_test.go` | 103 | `TestLiquidityPulseOutputJSON` |
| `market_breadth_e2e_test.go` | 32 | `TestMarketBreadthCommand` |
| `stablecoin_power_e2e_test.go` | 28 | `TestStablecoinPowerCommand` |

T1b — 2 ungated meta checks in liquidity_pulse:

Changed from ungated `Meta == nil` checks to status-gated pattern matching
flow_tension, market_breadth, and stablecoin_power. New pattern:
```go
st := resp.Results[0].Status
if (st == "ok" || st == "degraded") && resp.Results[0].Meta == nil {
    t.Error("Meta should be present for extended detail when data is available")
}
```

- `liquidity_pulse_e2e_test.go` — `TestLiquidityPulseDetailExtended`
- `liquidity_pulse_e2e_test.go` — `TestLiquidityPulseDetailFull`

T1c — Adversarial code review fixes (2 blocking issues found):

1. **Bounds guards**: All detail tests that access `resp.Results[0].Status` without
   checking `len(resp.Results)` now have `if len(resp.Results) == 0 { t.Fatal("no results in response") }`
   added before the access. Applied to 8 functions across 4 files (flow_tension,
   liquidity_pulse, market_breadth, stablecoin_power).

2. **momentum_divergence detail tests**: Meta nil check changed from ungated `t.Fatal`
   to status-gated `t.Error`. Content validation blocks wrapped in `if r.Meta != nil {}`
   to avoid nil-Meta Unmarshal panics when result status is `"unavailable"`.

**Verification:** `go test -race -count=1 ./cmd/cryptospect-cli/` — 44/44 pass.
Plus `go vet`, `golangci-lint`, `go build` all clean.

---

### T2 — Pin formatter versions in CI

**File:** `.github/workflows/ci.yml`

Change from `@latest` to pinned versions:

```yaml
go install golang.org/x/tools/cmd/goimports@v0.45.0
go install mvdan.cc/gofumpt@v0.10.0
```

**Version validation (2026-05-11):**

| Tool | Plan said | Correct version | Source |
|---|---|---|---|
| gofumpt | v0.10.0 | v0.10.0 ✅ | `gofumpt --version` → `v0.10.0 (go1.25.9)` |
| goimports | v0.31.0 | **v0.45.0** ❌ | `go version -m $(which goimports)` → `golang.org/x/tools v0.45.0` |

The plan's original `v0.31.0` was a placeholder. Using that would install a 14-release-old
formatter not aligned with the codebase formatting. v0.45.0 is what is currently installed
and has already been used to format all committed code.

Note: `goimports --version` does not exist. The correct verification command is:
```bash
go version -m $(which goimports) | grep 'mod.*tools'
```

Both pinned versions are confirmed valid tags (`golang.org/x/tools v0.45.0` exists in
the module index). gofumpt v0.10.0 is still the latest release as of 2026-05-11.

**How to verify before commit:**
```bash
# Confirm the pinned versions resolve
go install golang.org/x/tools/cmd/goimports@v0.45.0 && echo "goimports@v0.45.0 OK"
go install mvdan.cc/gofumpt@v0.10.0           && echo "gofumpt@v0.10.0 OK"

# Confirm no formatting drift vs. current code
gofumpt -l cmd/ internal/  # must produce no output
goimports -l cmd/ internal/  # must produce no output
```

**Known residual gap:** `make fmt` runs whatever `goimports`/`gofumpt` is on `$PATH`.
If a developer upgrades locally, `make fmt` output could diverge from CI even after
pinning. A future fix could add a `tools.go` or version check in the Makefile.

---

### T3 — Add pre-push hook

**File:** `.git/hooks/pre-push` (not git-tracked; installed in local clone)

**Updated script (fixes both gaps found during review):**

```sh
#!/bin/sh
set -e

echo "pre-push: fmt check (gofumpt)..."
UNFORMATTED=$(find . -name "*.go" \
  -not -path "./.cache/*" -not -path "./.gomodcache/*" \
  -not -path "./.go/*" -not -path "./vendor/*" \
  -exec gofumpt -l {} \;)
if [ -n "$UNFORMATTED" ]; then
  echo "pre-push: gofumpt would reformat: $UNFORMATTED"
  echo "Run: make fmt"
  exit 1
fi

echo "pre-push: fmt check (goimports)..."
UNFORMATTED=$(find . -name "*.go" \
  -not -path "./.cache/*" -not -path "./.gomodcache/*" \
  -not -path "./.go/*" -not -path "./vendor/*" \
  -exec goimports -l {} \;)
if [ -n "$UNFORMATTED" ]; then
  echo "pre-push: goimports would reformat: $UNFORMATTED"
  echo "Run: make fmt"
  exit 1
fi

echo "pre-push: lint..."
golangci-lint run ./...

echo "pre-push: unit/provider tests (skipping live E2E)..."
go test -race -short ./...

echo "pre-push: all checks passed."
```

Make executable: `chmod +x .git/hooks/pre-push`

**Gaps found in original plan and fixed above:**

1. **Missing goimports check**: Original hook only ran `gofumpt -l`. CI checks both
   `goimports -l` and `gofumpt -l`. Added goimports parity.

2. **Live E2E tests on push**: Running `go test -race ./...` includes the `cmd/cryptospect-cli/`
   E2E tests which call live CoinGecko/Binance APIs. These are slow (~30–60s) and
   subject to rate-limiting failures. Changed to `go test -race -short ./...` which
   skips E2E tests while still running unit/provider tests.

**Verification:** After installing the hook, run `git push --dry-run` or manually
invoke `sh .git/hooks/pre-push` to confirm all steps pass.

**Known limitation:** Hook files are not git-tracked. Each clone must be manually
configured. A `scripts/install-hooks.sh` or git-tracked hooks directory approach
could be added later if this becomes a recurring issue across environments.

---

### T4 — Fix .gitignore and untrack graphify-out

**File:** `.gitignore`

Replace the partial graphify exclusion block:
```gitignore
# Graphify local-only state and cache
graphify-out/manifest.json
graphify-out/cost.json
graphify-out/cache/
```
With:
```gitignore
# Graphify local-only state and cache
graphify-out/
```

Then untrack the already-committed files (without deleting them locally):
```bash
git rm -r --cached graphify-out/
```

Stage and commit together with the `.gitignore` change.

**How to verify:**
```bash
# After commit, graphify-out/ must not appear in tracked files
git ls-files graphify-out/  # must produce no output

# Local files still exist on disk
ls graphify-out/  # files should still be present

# New files in graphify-out/ show ignored status
echo test > graphify-out/test.txt && git status --short  # must not appear
rm graphify-out/test.txt
```

**Known limitation:** `git rm --cached` only stops future tracking. The 143 blobs
(743KB+) from commit `b529118` remain in git history. Full removal would require
`git filter-branch` or BFG Repo-Cleaner — deferred as non-blocking.

---

## Progress

### T1 — Complete ✅ (2026-05-11)
- 5 envelope assertions: `"ok"` → `"ok" || "error"`
- 2 Meta nil checks: ungated → status-gated on `(st == "ok" || st == "degraded")`
- 8 bounds guards (`len(resp.Results) == 0`) added to detail tests
- 2 momentum-divergence detail tests: ungated `t.Fatal` → gated `t.Error` + content blocks wrapped
- Verified: 44/44 tests pass, lint/vet/build clean, 5 files changed

### T2 — Not started (version corrections applied to plan)
### T3 — Not started (script gaps fixed in plan)
### T4 — Not started (verification steps added to plan)

---

## Implementation Order

1. ~~**T1** — unblocks CI, smallest risk, pure test changes~~ ✅ Done
2. **T2** — prevents future formatter drift, CI config change (1 file, 2 lines)
3. **T4** — cleans up repo pollution, removes tracked generated files
4. **T3** — local enforcement gate, not committed (hook files aren't tracked)

**Commit strategy:** T2 + T4 in separate commits. T3 is local-only.
(Original plan had T1+T2 in one commit, but T1 is already applied on the working branch.
T2 goes in its own commit. T4 in a separate commit.)
