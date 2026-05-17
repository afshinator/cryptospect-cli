# Adversarial Review — UX Polish (2026-05-17)

Review of changes made in the 2026-05-17 UX polish session (Phases 1–4).
Source: automated adversarial code review.

---

## Issue Index

| # | Severity | Area | Title | Status |
|---|----------|------|-------|--------|
| 1 | 🔴 Critical | output/writer.go | Data race on `pretty` bool | Open |
| 2 | 🔴 Critical | output/writer.go | `pretty` global leaks between tests | Open |
| 3 | 🟡 Important | internal/version | Zero test coverage on version package | Open |
| 4 | 🟡 Important | output/writer.go | Zero test coverage on `SetPretty`/`marshal` | Open |
| 5 | 🟡 Important | internal/version + Makefile | Tagged `make build` appends `-dirty` suffix | Open |
| 6 | 🟡 Important | cmd root_test.go | Version string and command groups untested | Open |
| 7 | 🔵 Minor | internal/version | Short revision silently truncated | Open |
| 8 | 🔵 Minor | internal/output | `marshal()` is a too-generic name | Open |
| 9 | 🔵 Minor | README.md | Version table implies tagged builds work now | Open |
| 10 | 🔵 Minor | README.md | `agents.md` link potentially stale | Open |
| 11 | 🔵 Minor | LICENSE | No SPDX identifier | Open |
| 12 | 🔵 Minor | internal/version | `v1.0.0` default implies released code | Open |

---

## Critical

### #1 — Data race on `pretty` bool

**File:** `internal/output/writer.go`

`SetPretty` acquires `stdoutMu.Lock()` to write `pretty`. But `marshal()` reads
`pretty` with no lock held — it is called *before* `stdoutMu.RLock()` is taken
in both `WriteSuccess` and `WriteError`.

```go
// writer.go — current (buggy)
func WriteSuccess(...) error {
    ...
    data, err := marshal(resp)   // reads pretty — NO lock held here
    ...
    stdoutMu.RLock()             // lock acquired AFTER the read
    defer stdoutMu.RUnlock()
    _, err = stdout.Write(data)
```

This is a data race by the Go memory model. `go test -race` won't catch it
today because no concurrent caller exists, but it will fail the moment any
test or goroutine overlaps a `SetPretty` call with a `Write*` call.

**Fix:** Use `sync/atomic.Bool` for `pretty`. It has no semantic relationship
to `stdout` and should not share its mutex.

```go
var pretty atomic.Bool

func SetPretty(v bool) { pretty.Store(v) }

func marshal(v any) ([]byte, error) {
    if pretty.Load() {
        return json.MarshalIndent(v, "", "  ")
    }
    return json.Marshal(v)
}
```

---

### #2 — `pretty` global leaks between tests

**File:** `internal/output/writer.go`

The package-level `pretty` is never reset. If any test sets it (or runs
`NewRootCommand().Execute()` with a config that has `output.pretty: true`),
every later test in the same binary sees indented JSON. Failures are
order-dependent and hard to diagnose.

**Fix (part of #1):** After converting to `atomic.Bool`, add an exported
reset helper and use it with `t.Cleanup` in any test that calls `SetPretty`:

```go
// ResetForTest restores package defaults. Use with t.Cleanup in tests.
func ResetForTest() {
    pretty.Store(false)
    stdoutMu.Lock()
    stdout = os.Stdout
    stdoutMu.Unlock()
}
```

---

## Important

### #3 — Zero test coverage on `internal/version`

**File:** `internal/version/version.go` (no `_test.go` exists)

`String()` has four distinct branches — none are exercised by any test.

```
Branch 1: ReadBuildInfo returns !ok        → return Value
Branch 2: no vcs.revision in settings      → return Value
Branch 3: revision set, not dirty          → "Value (commit)"
Branch 4: revision set, dirty              → "Value (commit-dirty)"
```

**Fix:** Refactor `String()` to delegate to a pure, injectable helper:

```go
func formatVersion(base string, settings []debug.BuildSetting) string { ... }

func String() string {
    info, ok := debug.ReadBuildInfo()
    if !ok {
        return Value
    }
    return formatVersion(Value, info.Settings)
}
```

`formatVersion` can then be table-tested exhaustively without needing to
mock `debug.ReadBuildInfo`.

---

### #4 — Zero test coverage on `SetPretty` and `marshal`

**File:** `internal/output/writer_test.go`

`writer_test.go` was not updated this session. The new public API (`SetPretty`)
and the `pretty` branch of `marshal` have no tests. The README documents
`output.pretty: true` as a supported feature, but there is no CI assertion that
it actually produces indented JSON.

**Fix:** Add to `writer_test.go`:

```go
func TestWriteSuccessPretty(t *testing.T) {
    t.Cleanup(output.ResetForTest)
    SetPretty(true)
    var buf bytes.Buffer
    SetWriter(&buf)
    _ = WriteSuccess([]MetricResult{{Metric: "x", Status: "ok"}})
    out := buf.String()
    if !strings.HasPrefix(out, "{\n") {
        t.Errorf("expected indented JSON, got: %q", out)
    }
}

func TestWriteSuccessCompact(t *testing.T) {
    // Ensures the default (pretty=false) path is explicit.
    var buf bytes.Buffer
    SetWriter(&buf)
    defer SetWriter(os.Stdout)
    _ = WriteSuccess([]MetricResult{{Metric: "x", Status: "ok"}})
    out := buf.String()
    if strings.Contains(out, "\n") {
        t.Errorf("expected compact JSON, got newlines: %q", out)
    }
}
```

---

### #5 — Tagged `make build` appends `-dirty` suffix

**Files:** `internal/version/version.go`, `Makefile`

When `make build` runs on a tagged commit:
1. `git describe --tags --dirty` returns e.g. `v1.0.1` — no dirty marker in git describe
2. The build writes `bin/cryptospect-cli`, making the working tree dirty
3. `debug.ReadBuildInfo()` inside the binary returns `vcs.modified=true`
4. `String()` returns `v1.0.1 (abcdef1-dirty)` — wrong for a release binary

The README's version table ("with a git tag → `v1.0.0`") is currently incorrect.

**Fix:** Add a sentinel ldflags var so `String()` skips `ReadBuildInfo` entirely
on tagged builds:

```go
// version.go
var (
    Value  = "v1.0.0"
    tagged = "" // set to "true" by ldflags on tagged builds
)

func String() string {
    if tagged == "true" {
        return Value  // clean release build — no commit suffix
    }
    // ... existing ReadBuildInfo logic
}
```

```makefile
# Makefile — tagged branch only
LDFLAGS := -s -w \
    -X github.com/afshinator/cryptospect-cli/internal/version.Value=$(VERSION) \
    -X github.com/afshinator/cryptospect-cli/internal/version.tagged=true
```

---

### #6 — Version string and command groups untested

**File:** `cmd/cryptospect-cli/root_test.go`

The session introduced two user-visible structural changes — a real version
string and grouped help output — but `root_test.go` was not updated. No
assertion exists for:

- `cmd.Version` is non-empty and starts with `"v"`
- Both `"metrics"` and `"utility"` groups exist on the root command
- Metric subcommands have `GroupID == "metrics"`
- `list-metrics` and `cache-clear` have `GroupID == "utility"`
- `cmd.CompletionOptions.HiddenDefaultCmd == true`

**Fix:** Add to `root_test.go`:

```go
func TestRootVersion(t *testing.T) {
    cmd := NewRootCommand()
    if cmd.Version == "" {
        t.Fatal("Version is empty")
    }
    if !strings.HasPrefix(cmd.Version, "v") {
        t.Errorf("Version = %q, want v-prefixed string", cmd.Version)
    }
}

func TestCommandGroups(t *testing.T) {
    cmd := NewRootCommand()
    groups := map[string]bool{}
    for _, g := range cmd.Groups() {
        groups[g.ID] = true
    }
    for _, want := range []string{"metrics", "utility"} {
        if !groups[want] {
            t.Errorf("group %q not found", want)
        }
    }
    for _, sub := range cmd.Commands() {
        switch sub.Use {
        case "list-metrics", "cache-clear":
            if sub.GroupID != "utility" {
                t.Errorf("%s GroupID = %q, want utility", sub.Use, sub.GroupID)
            }
        case "market-regime", "liquidity-pulse", "flow-tension",
             "stablecoin-power", "market-breadth", "momentum-divergence":
            if sub.GroupID != "metrics" {
                t.Errorf("%s GroupID = %q, want metrics", sub.Use, sub.GroupID)
            }
        }
    }
}

func TestCompletionHidden(t *testing.T) {
    cmd := NewRootCommand()
    if !cmd.CompletionOptions.HiddenDefaultCmd {
        t.Error("CompletionOptions.HiddenDefaultCmd should be true")
    }
}
```

---

## Minor

### #7 — Short revision silently truncated

**File:** `internal/version/version.go` line 25

```go
if len(s.Value) >= 7 {
    commit = s.Value[:7]
}
```

If `vcs.revision` is shorter than 7 chars (non-git VCS, test stub), `commit`
stays empty and the function returns bare `Value` with no hint that a revision
was present. Use `min(7, len(s.Value))` instead.

---

### #8 — `marshal()` is too generic a name

**File:** `internal/output/writer.go`

In a package called `output`, `marshal` reads as a general-purpose serialiser.
Rename to `encodeJSON` to make intent clear.

---

### #9 — README version table implies tagged builds work now

**File:** `README.md` — Build section, version strings table

The table shows a row for "with a git tag → `v1.0.0`" but no tags exist in
this repo. Add a parenthetical: *(no tags yet — current builds always show
the dev format)*.

---

### #10 — `agents.md` link potentially stale

**File:** `README.md` line 168

README references `agents.md` as "the full orchestration playbook." Verify
contents are current; if stale, update or remove the reference.

---

### #11 — No SPDX identifier in LICENSE

**File:** `LICENSE`

`SPDX-License-Identifier: MIT` as the first line improves automated SBOM
tooling compatibility. Not required, but increasingly standard.

---

### #12 — `v1.0.0` default implies released code

**File:** `internal/version/version.go` line 10

SemVer convention for unreleased work is `v1.0.0-dev` or `v0.x.y`. The
current default conflates "the version we are heading toward" with "the
version currently running." Design decision — not a correctness bug.
