# Go Version Bump: 1.25.9 → 1.25.11 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the pinned Go version from 1.25.9 to 1.25.11, fixing 26 govulncheck CVEs (11 distinct), and align CI to read version from `go.mod` as single source of truth.

**Architecture:** Three-phase change: (1) install Go 1.25.11 in the container, (2) update project files in dependency order (local first → CI second), (3) verify all gates pass.

**Critical ordering constraint:** `go.mod` must NOT be updated before the build environment has Go 1.25.11 installed. If go.mod says `go 1.25.11` and the system Go is 1.25.9, every `go` command will fail with `module requires go 1.25.11`. CI must also be updated to pin explicitly or use `go-version-file` to avoid the same failure in CI.

**Verification gates (must all pass before marking complete):**
- `go build ./cmd/cryptospect-cli` — clean
- `go test -race -cover ./...` — all tests pass
- `golangci-lint run ./...` — clean
- `go vet ./...` — clean
- `gofumpt -l cmd/ internal/` — no output
- `goimports -l cmd/ internal/` — no output

**Files changed:**
- `/usr/local/go/` — Go installation (replace entire directory)
- `go.mod:3` — `go 1.25.9` → `go 1.25.11`
- `.go-version:1` — `1.25.9` → `1.25.11`
- `.github/workflows/ci.yml:17-19` — `go-version: "1.25"` → `go-version-file: go.mod`

---

### Task 1: Verify Go 1.25.11 is available

**Files:** None (read-only check)

- [ ] **Step 1: Confirm 1.25.11 exists on the Go download server**

Run:
```bash
curl -sL 'https://go.dev/dl/?mode=json' | jq '[.[] | select(.version | test("go1\\.25")) | .version]'
```

Expected:
```json
["go1.25.11"]
```

If 1.25.11 is NOT listed (e.g., only shows `"go1.25.10"` or older), stop and fall back to the latest available 1.25.x patch. Update all references in the plan to use that version instead.

- [ ] **Step 2: Verify SHA256 checksum is published**

Run:
```bash
curl -sL 'https://go.dev/dl/?mode=json' | jq '.[] | select(.version == "go1.25.11") | .files[] | select(.filename == "go1.25.11.linux-amd64.tar.gz") | {filename, sha256, size}'
```

Expected: sha256 and size fields present.

---

### Task 2: Install Go 1.25.11 in the container

**Files:**
- Remove + replace: `/usr/local/go/` (the entire Go installation directory)

**Prerequisite:** Task 1 must complete (confirmed 1.25.11 exists and checksum is known).

**Why this must be first:** The current go.mod says `go 1.25.9`. If we update go.mod to `go 1.25.11` first, every subsequent `go` command (build, test, tidy) will fail because the Go 1.25.9 toolchain will refuse to process a module requiring 1.25.11.

- [ ] **Step 1: Download Go 1.25.11 tarball**

```bash
curl -sL -o /tmp/go1.25.11.linux-amd64.tar.gz https://go.dev/dl/go1.25.11.linux-amd64.tar.gz
```

- [ ] **Step 2: Verify tarball checksum**

```bash
echo "$(curl -sL 'https://go.dev/dl/?mode=json' | jq -r '.[] | select(.version == "go1.25.11") | .files[] | select(.filename == "go1.25.11.linux-amd64.tar.gz") | .sha256')  /tmp/go1.25.11.linux-amd64.tar.gz" | sha256sum -c -
```

Expected: `go1.25.11.linux-amd64.tar.gz: OK`

- [ ] **Step 3: Extract to /usr/local/go**

```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf /tmp/go1.25.11.linux-amd64.tar.gz
```

Note: `sudo` is required because `/usr/local/go` is owned by root. If the container has no `sudo` or the user is root, adjust accordingly.

- [ ] **Step 4: Verify installation**

```bash
go version
```

Expected: `go version go1.25.11 linux/amd64`

- [ ] **Step 5: Clean up tarball**

```bash
rm /tmp/go1.25.11.linux-amd64.tar.gz
```

- [ ] **Step 6: Run go mod tidy to confirm the current go.mod is still valid with new Go**

```bash
cd /project/cryptospect-cli
go mod tidy
```

Expected: no changes to go.mod or go.sum (the go.mod currently requires 1.25.9, and 1.25.11 is backward compatible).

---

### Task 3: Update go.mod go directive from 1.25.9 to 1.25.11

**Files:**
- Modify: `go.mod:3`

**Prerequisite:** Task 2 must complete (Go 1.25.11 installed in container). If we change go.mod first, `go mod tidy` in the next step would fail.

- [ ] **Step 1: Update the go directive**

`go.mod` line 3: `go 1.25.9` → `go 1.25.11`

Use exact edit:
```
go 1.25.9
```
→
```
go 1.25.11
```

- [ ] **Step 2: Run go mod tidy to refresh module state**

```bash
cd /project/cryptospect-cli
go mod tidy
```

Expected: no changes to go.mod or go.sum. The `go` directive version change doesn't affect dependency resolution—it only sets the minimum Go toolchain version.

---

### Task 4: Update .go-version

**Files:**
- Modify: `.go-version:1`

**Prerequisite:** None (no dependency on other tasks—just keeps local tooling in sync).

- [ ] **Step 1: Update version string**

`.go-version` line 1: `1.25.9` → `1.25.11`

Use exact edit:
```
1.25.9
```
→
```
1.25.11
```

---

### Task 5: Update CI to read version from go.mod

**Files:**
- Modify: `.github/workflows/ci.yml:17-19`

**Critical context:** CI currently uses `go-version: "1.25"` (line 19). This resolves to the latest available 1.25.x on GitHub Actions runners—but there's no guarantee it resolves to ≥1.25.11. If it resolves to 1.25.9 or 1.25.10, CI will fail because `go.mod` now requires 1.25.11. The fix is to switch to `go-version-file: go.mod`, which tells `actions/setup-go` to read the exact version from go.mod.

**Why this task must come AFTER Tasks 2-3:** If CI is updated to `go-version-file: go.mod` before go.mod says 1.25.11, CI would install Go 1.25.9 (from the old go.mod value). That would still work, but there's no benefit to updating CI before the go.mod change. Updating after ensures the first CI run on the merged PR uses the correct version.

- [ ] **Step 1: Replace `go-version` with `go-version-file`**

In `.github/workflows/ci.yml`, change lines 17-19:

```yaml
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
```

→

```yaml
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
```

Note: `actions/setup-go@v5` supports `go-version-file`. It reads the `go` directive from go.mod and installs that exact version. Version resolution: local cache → go-versions manifest → direct download from go.dev.

Also note: the `go-version` and `go-version-file` inputs are mutually exclusive—if both are provided, `go-version` takes precedence. So we must remove `go-version` entirely.

---

### Task 6: Build and verify

**Files:** None (verification only)

**Prerequisite:** All file changes complete.

- [ ] **Step 1: Build the binary**

```bash
cd /project/cryptospect-cli
go build -o /dev/null ./cmd/cryptospect-cli
```

Expected: clean exit, no errors.

- [ ] **Step 2: Run the full test suite with race detector**

```bash
GOMODCACHE=$(PWD)/.gomodcache GOCACHE=$(PWD)/.cache/go go test -race -cover ./...
```

Expected: all tests pass. If any fail, investigate whether they are pre-existing or caused by the Go version change.

- [ ] **Step 3: Run golangci-lint**

```bash
golangci-lint run ./...
```

Expected: clean (0 issues).

- [ ] **Step 4: Run go vet**

```bash
go vet ./...
```

Expected: clean.

- [ ] **Step 5: Check formatting**

```bash
gofumpt -l cmd/ internal/
goimports -l cmd/ internal/
```

Expected: both produce no output.

---

### Task 7: Update Design-Decisions.md

**Files:**
- Modify: `Design-Decisions.md:13-18`

**Prerequisite:** None (documentation update, no code dependency).

**Why:** The Design-Decisions.md is the source-of-truth doc. It currently says "Exact patch: 1.25.9". This needs updating to reflect the new pinned version and the reasoning.

- [ ] **Step 1: Update the Go Version section**

In `Design-Decisions.md`, lines 12-18:

```
### Go Version
- Target: Go 1.25 (previous stable, fully supported)
- Exact patch: 1.25.9 (pinned in .go-version and go.mod toolchain directive)
- Rationale: 1.26 is too new (2 months old), 1.24 and below are end-of-life.
  Go 1.25 is settled in CI images and Docker base images. Nothing in 1.26
  changes this project's capabilities.
- Upgrade plan: bump to 1.26 when 1.27 ships (~August 2026), at which point
  1.25 drops out of support.
```

→

```
### Go Version
- Target: Go 1.25 (previous stable, fully supported)
- Exact patch: 1.25.11 (pinned in .go-version and go.mod toolchain directive)
- Rationale: 1.26 is too new (2 months old), 1.24 and below are end-of-life.
  Go 1.25 is settled in CI images and Docker base images. Nothing in 1.26
  changes this project's capabilities.
- Patch update (2026-06-15): 1.25.9 → 1.25.11 to address 26 govulncheck
  findings (11 distinct CVEs, 6 reachable from project code). Full analysis:
  docs/reviews/go-version-analysis.md
- Upgrade plan: bump to 1.26 when 1.27 ships (~August 2026), at which point
  1.25 drops out of support.
```

---

## Rollback Plan

If any verification step in Task 6 fails:

1. **Revert go.mod:** `go 1.25.11` → `go 1.25.9`
2. **Revert .go-version:** `1.25.11` → `1.25.9`
3. **Revert ci.yml:** `go-version-file: go.mod` → `go-version: "1.25"`
4. **Reinstall Go 1.25.9:** Download and extract `go1.25.9.linux-amd64.tar.gz` to `/usr/local/go`
5. **Verify rollback:** Repeat all Task 6 steps

## Fallback: If 1.25.11 doesn't exist

If `go1.25.11` is not available on go.dev/dl, use the latest available 1.25.x patch (e.g., 1.25.10). Replace every occurrence of `1.25.11` in this plan with that version. Note: 1.25.10 would fix 8 of the 11 CVEs but leave 3 reachable CVEs (CVE-2026-27145, CVE-2026-42504, CVE-2026-42507) unpatched.
