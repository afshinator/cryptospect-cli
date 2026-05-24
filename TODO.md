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

---

Your `CLAUDE.md` is a phenomenal technical resource, but it is currently serving two completely different masters: **your project contributors** (the development guidelines) and **the LLM/AI agents** using the tool (the orchestration rules).

Because it contains internal development rules (like style guides, make commands, and project status), copy-pasting it directly into an agent's context window wastes valuable tokens and introduces noise.

Here is an evaluation of your current file and a breakdown of how to turn it into a high-visibility marketing asset for the AI developer community:

### 1. What’s Already Great in Your `CLAUDE.md`

* 
**Clear Orchestration Sequence:** The "Orchestration" section at the bottom is exactly what an agent needs. Telling the agent to run `market-regime` first to understand the macro context before drilling down is a brilliant top-down framework.


* 
**Flag Instructions:** Telling the agent to use `--detail full` to get thresholds and descriptions ensures the LLM doesn't have to hallucinate what an arbitrary score means.


* 
**Pure JSON Input/Output Guardrails:** Explicitly stating that stdout is *always* valid JSON and stderr handles diagnostics keeps the agent from breaking when handling command outputs.



### 2. The Big Optimization Opportunity: Structural Split

To capture the attention of developers looking for turn-key "agent brains," you should split this document into two distinct assets in your repository:

1. 
**Keep `CLAUDE.md` purely for development:** Retain the project status, testing conventions, and build instructions so Claude can seamlessly maintain your Go code.


2. 
**Extract a dedicated `agents.md` (or `docs/agent-orchestration.md`):** This is your marketing goldmine. This file will be explicitly written for external developers to plug directly into their agent systems.



### 3. Draft Blueprint for Your New `agents.md`

By packaging this nicely in your repository, you provide a copy-pasteable configuration file that developers can instantly feed into LangChain, AutoGPT, CrewAI, or Claude Desktop.

Here is how you can expand and format your existing rules into a standalone agent blueprint:

```markdown
# Crypto Agent Orchestration Guide (cryptospect-cli)

This document is optimized for LLMs and AI Agents utilizing `cryptospect-cli`. Feed this guide directly into your agent's system prompt or context window to enable expert quantitative analysis.

## Core Rules for the Agent
1. **Always Use JSON Mode:** Invoke commands with the default output format. Rely on stdout containing a structured `CLIResponse` envelope. Ignore diagnostic text on stderr.
2. **Request Full Details for Analysis:** Always append the `--detail full` flag when performing reasoning. This appends quantitative thresholds and metric descriptions directly inside the `meta` object, allowing you to ground your conclusions without guessing bounds.
3. **Budget Tokens with Basic Mode:** For frequent background polling loops where reasoning is not required, use `--detail basic` to omit metadata and minimize your context footprint.

## Sequential Discovery Framework (The Brain)
To diagnose the crypto market accurately, never query metrics in a random order. Follow this multi-tiered analytical flow:

1. **Macro Context First:** Always execute `cryptospect-cli market-regime` (alias: `mr`) first. Establish the overarching macro state (e.g., capitulation, expansion) and note the weighted macro confidence score.
2. **Liquidity & Flow Drilldown:** If the market regime shows high conviction or shifting dominance, isolate the moving forces by executing:
   - `liquidity-pulse` (`lp`) to determine if real capital is active or idle.
   - `flow-tension` (`ft`) to identify immediate spot CVD direction (buyers vs. sellers).
3. **Capital Rotation:** Execute `momentum-divergence` (`md`) and `market-breadth` (`mb`) to evaluate if market movements are supported by the broader ecosystem or artificial large-cap manipulation.

## Advanced Signal Patterns to Recognize
As an advanced analyst agent, look for the following cross-metric patterns:

* **Ghost Rally Divergence:** Detected when `market-regime` notes a structural breadth failure, or when large caps push higher but `market-breadth` (`mb`) fails across multi-timeframe composites. This indicates an unstable, non-inclusive rally.
* **Flow Tension Fatigue:** When price is flat or falling but `flow-tension` (`ft`) indicates strong buying consensus via Binance spot CVD, watch for an imminent liquidity pop or short squeeze.
* **Dry Powder Expansion:** A surge in `stablecoin-power` (`sp`) paired with a high-fear macro state (`fear-greed-index` / `fgi`) indicates massive sideline liquidity waiting to buy a capitulation bottom.

```

