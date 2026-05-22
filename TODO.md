# TODO

## Post-v1: Remove LLM tool config from repo

**Priority:** Low (deferred until after v1 deploy)

These files are currently tracked in git because development switches between machines and they need to roam with the repo. Once v1 is deployed and the multi-machine dev workflow stabilises, they should be removed from git tracking and added to `.gitignore`.

Files to untrack (keep locally, not in repo):

- `.cursorrules` — symlink to `agents.md`; Cursor IDE AI rules
- `.agents/` — agent framework skill definitions
- `.config/opencode/` — OpenCode agent config (`AGENTS.md`)
- `.pi-lens/` — Pi agent state files
- `skills-lock.json` — Dex/superpowers skill engine lock
- `.dex/` — Dex task engine state (`tasks.jsonl`)

**`agents.md` stays tracked** — it's project documentation that informs LLMs how to use the tool.

Execution (when ready): add all the above to `.gitignore`, then `git rm --cached` each entry, commit as `chore: untrack LLM tool config files`.

---

## China M2: Add PBoC as data source

**Priority:** Medium (DBnomics has 82-day lag as of 2026-05-20, approaching 90-day confidence downgrade)

### Problem
DBnomics China M2 data is consistently ~2 months behind (latest: Feb 2026). The PBoC publishes March and April data on schedule, but our pipeline cannot consume it.

### PBoC Source Details

#### 1. Money & Banking Statistics page (stable yearly URL)
```
https://www.pbc.gov.cn/diaochatongjisi/116219/116319/{year}ntjsj/hbtjgl/index.html
```
e.g., `https://www.pbc.gov.cn/diaochatongjisi/116219/116319/2026ntjsj/hbtjgl/index.html`

This page lists downloadable files for several datasets. The relevant row:
- **货币供应量** (Money Supply) — available as HTM, XLSX, PDF
- Latest XLSX as of 2026-05-20: `https://www.pbc.gov.cn/diaochatongjisi/attachDir/2026/05/2026051419141229799.xlsx`
- Latest HTM as of 2026-05-20: `https://www.pbc.gov.cn/diaochatongjisi/attachDir/2026/05/2026051418014041108.htm`

The XLSX URL is timestamp-based and changes each month, but can be discovered by scraping the Money & Banking page and finding the `<a>` tag whose text contains "货币供应量" and links to an `.xlsx` file.

#### 2. XLSX data structure
The XLSX has a single sheet with columns:
- Column A: Item name (e.g., "货币和准货币（M2）")
- Columns C through N: Monthly data for the year (2026.01 through 2026.12)
- Row 8 (1-indexed): M2 data — "货币和准货币（M2）" / "Money & Quasi-money"

The HTM file is an Excel HTML export with identical structure.

#### 3. M2 values extracted (in 100 million yuan)
```
2026.01: 3,471,860.39
2026.02: 3,492,159.91
2026.03: 3,538,636.53
2026.04: 3,530,425.21
```

Current value (April 2026): 3,530,425.21 × 10^8 = 353,042,521,000,000 yuan = **353.04 trillion yuan** (8.6% YoY).

#### 4. Alternative: News report page (more fragile)
The monthly Financial Statistics Report is published at a URL under:
```
https://www.pbc.gov.cn/goutongjiaoliu/113456/113469/{timestamp}/index.html
```
e.g., April 2026 report: `https://www.pbc.gov.cn/goutongjiaoliu/113456/113469/2026051414253144700/index.html`

The M2 figure is in section "三、广义货币增长8.6%" with text like:
"4月末，广义货币（M2）余额353.04万亿元,同比增长8.6%。"

This is harder to parse (Chinese NLP, monthly URL discovery) — not recommended vs XLSX.

### Implementation plan (future)

1. **Add dependency:** `go get github.com/xuri/excelize/v2` (popular Go XLSX library, no CGO)
2. **Create `internal/api/pboc/` package:**
   - `client.go` — HTTP client to fetch the Money & Banking page, find the Money Supply XLSX link, download it
   - `parse.go` — Parse the XLSX to extract M2 observations (period → value)
3. **Update `internal/metrics/chinam2/v1/provider.go`:**
   - Register PBOC endpoint in `Def().Endpoints`
   - Add PBoC parsing in `Compute()` as fallback when DBnomics data is missing or stale
   - Set `meta.PrimarySource` to `"pboc"` when PBoC data is used
4. **Fallback logic:**
   - Fetch DBnomics data first (existing behavior)
   - If DBnomics data is stale (>60 days) or missing, fetch PBoC XLSX as backup
   - Could also always warm PBoC in background for redundancy

### English version (blocked)
PBoC English site at `https://www.pbc.gov.cn/english/` returns 403 Forbidden. No English HTTP endpoint available.

### Key considerations
- XLSX approach is preferred over HTM parsing — structured, versioned format, less brittle
- excelize library is pure Go, no CGO, widely used (5k+ stars)
- Yearly URL change (`2026ntjsj` → `2027ntjsj` in January) — the code should construct the URL dynamically from `time.Now().Year()`
- Data is published monthly, typically around 14th–15th of the following month
- Units in XLSX are "亿元人民币" (100 million yuan) — same as DBnomics; divide by 10 to get CNY billion
