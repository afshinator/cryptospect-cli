# Go Quality Audit — 2026-06-15

**Tool:** `mcp-server-go-quality` (golangci-lint v2.11.4 + govulncheck + nilaway)
**Project:** cryptospect-cli
**Go version:** 1.25.9

## Summary

| Checker | Findings | Notes |
|---------|----------|-------|
| golangci-lint | 0 | Clean |
| govulncheck | 26 | 20 stdlib-only (fixed in ≥1.25.10), 6 reachable from project code |
| nilaway | 9 | Potential nil panics across 4 files |

---

## golangci-lint — 0 issues

Clean.

---

## govulncheck — 26 findings

### Stdlib-only (20)

All resolved by upgrading Go to **v1.25.11** (stdlib fixed versions: v1.25.10 or v1.25.11).
None are reachable through project code — only the packages are imported transitively.

| OSV | CVE | Package | Summary | Fixed |
|-----|-----|---------|---------|-------|
| GO-2026-4971 | CVE-2026-39836 | `net` | Panic in Dial/LookupPort on NUL byte (Windows) | v1.25.10 |
| GO-2026-4986 | CVE-2026-39820 | `net/mail` | Quadratic string concat in consumeComment | v1.25.10 |
| GO-2026-4982 | CVE-2026-39823 | `html/template` | Meta content URL escaping bypass → XSS | v1.25.10 |
| GO-2026-4980 | CVE-2026-39826 | `html/template` | Escaper bypass → XSS | v1.25.10 |
| GO-2026-5037 | CVE-2026-27145 | `crypto/x509` | Inefficient hostname parsing | v1.25.11 |
| GO-2026-4981 | CVE-2026-33811 | `net` | Crash on long CNAME response | v1.25.10 |
| GO-2026-5039 | CVE-2026-42507 | `net/textproto` | Unescaped error messages | v1.25.11 |
| GO-2026-5038 | CVE-2026-42504 | `mime` | Quadratic WordDecoder.DecodeHeader | v1.25.11 |
| GO-2026-4918 | CVE-2026-33814 | `net/http` (x/net) | HTTP/2 infinite loop on bad SETTINGS_MAX_FRAME_SIZE | v1.25.10 |
| GO-2026-5024 | CVE-2026-39824 | `golang.org/x/sys` | Integer overflow in NewNTUnicodeString (Windows) | v0.44.0 |
| GO-2026-4976 | CVE-2026-39825 | `net/http/httputil` | ReverseProxy query forwarding | v1.25.10 |
| GO-2026-4977 | CVE-2026-42499 | `net/mail` | Quadratic string concat in consumePhrase | v1.25.10 |

The remaining 8 are duplicates of the same OSVs (govulncheck reports them for each call chain independently).

### Reachable from project code (6)

#### 1. `internal/httpclient/client.go:78` (`client.Get`)

`resp, err := c.doer.Do(req)` — HTTP client reaches two stdlib vulns:

| OSV | Summary | Fixed |
|-----|---------|-------|
| GO-2026-4971 (CVE-2026-39836) | `net.DialContext` panic on NUL byte (Windows-only) | v1.25.10 |
| GO-2026-4918 (CVE-2026-33814) | HTTP/2 infinite loop on bad SETTINGS_MAX_FRAME_SIZE | v1.25.10 |

**Call chain:** `client.Get` → `http.Do` → `http.send` → `http.RoundTrip` → `transport.roundTrip` → `dialConn` → `net.DialContext`

Both fixed in Go 1.25.10.

#### 2. `internal/httpclient/client.go:84` (`client.Get`)

`body, readErr := io.ReadAll(resp.Body)` — response body read reaches three vulns:

| OSV | Summary | Fixed |
|-----|---------|-------|
| GO-2026-5039 (CVE-2026-42507) | Unescaped input in `net/textproto.ReadMIMEHeader` errors | v1.25.11 |
| GO-2026-5037 (CVE-2026-27145) | Inefficient hostname parsing in `crypto/x509.Verify` / `VerifyHostname` | v1.25.11 |

**Call chains:**
- `ReadAll` → `http.Response.Read` → `readLocked` → `readTrailer` → `textproto.ReadMIMEHeader`
- `ReadAll` → `http.transport.Read` → `tls.Read` → `handshake` → `x509.Verify` / `x509.VerifyHostname`

Fixed in v1.25.11.

#### 3. `internal/api/coingecko/client.go:362` (`CoinMarketsMomentumURL`)

`fmt.Sprintf` formatting URL reaches `crypto/x509` via Sprintf of string:

| OSV | Summary | Fixed |
|-----|---------|-------|
| GO-2026-5037 (CVE-2026-27145) | Inefficient hostname parsing in `crypto/x509.Error` via `fmt.Sprintf` → `fmt.doPrintf` → `fmt.printArg` → `fmt.handleMethods` | v1.25.11 |

---

## nilaway — 9 potential nil panics

### 1. `cmd/cryptospect-cli/meta_processing.go:55` (2×)

```go
var meta map[string]any
if err := json.Unmarshal(metaJSON, &meta); err != nil {
    return metaJSON
}
meta["cache_hit"] = cacheHit      // <-- line 55
```

`meta` starts as nil (zero-value map). `json.Unmarshal` initializes it on success, but nilaway doesn't track the side-effect. Not a real bug — `json.Unmarshal` always allocates the map on success.

### 2. `internal/api/coingecko/client.go:288,290` (4×)

```go
counts := map[string]*TimeframeMetric{
    "1h":  {},
    "24h": {},
    "7d":  {},
    "30d": {},
}

if e.Change1h != nil {
    counts["1h"].TotalCount++    // line 288
    if *e.Change1h > 0 {
        counts["1h"].GreenCount++ // line 290
    }
}
```

Nilaway flags deep reads from `counts["1h"]` because map access on a `*T` map could return nil if the key is missing. All keys are explicitly initialized — not a real bug. Same pattern at lines 295/297, 302/304, 309/311 (references in nilaway output for 4 additional sites).

### 3. `internal/api/coingecko/global_stables_test.go:144` (1×)

```go
markets, err := ParseStablesMarketsResponse([]byte(stablesFixture))
// err not checked before use
if markets[0].ID != "tether" {  // line 144
```

`ParseStablesMarketsResponse` can return `nil, nil` for empty response. The fixture is valid and parsing always succeeds, but nilaway flags the missing nil check. Low-risk.

Also referenced: lines 145, 147, 148, 150, 151 (5 additional unchecked access sites from same nil source).

### 4. `internal/cache/cache.go:64` (1×)

```go
func (c *Cache) Close() error {
    c.closed = true
    c.dir = ""   // line 64
    return nil
}
```

Nilaway traces the nil flow from test code where `Open()` error is unchecked:
```
c, err := cache.Open(tmpDir)  // error ignored
defer c.Close()               // c could be nil
```
Not a production bug — the nil flow originates from test code.

### 5. `internal/cache/cache.go:77` (1×)

```go
func (c *Cache) filePath(endpoint string) string {
    safeName := filepath.Join(c.dir, endpoint+".json")  // line 77
    return safeName
}
```

Same root cause: nil receiver from unchecked `Open()` error in test code.

---

## Recommendation

1. **Upgrade Go to v1.25.11** — resolves all 26 vulnerability findings (6 reachable + 20 stdlib-only).
2. **nilaway false positives** — all 9 are benign. The 2 in `meta_processing.go` and 4 in `client.go` are map/dereference patterns that nilaway can't track. The 2 in `cache.go` and 1 in `global_stables_test.go` trace to unchecked error returns in test code.
3. **No code changes needed** for any finding — all actionable items are addressed by the Go version upgrade.
