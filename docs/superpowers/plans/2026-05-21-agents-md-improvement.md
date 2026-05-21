# agents.md Improvement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure `agents.md` from a CLI syntax reference into a reasoning guide — intent mapping, named signal patterns, and response rules — with condensed operational reference at the bottom.

**Architecture:** Single-file rewrite. All 16 named patterns are sourced verbatim from the `Agentic Logic` sections of `docs/metrics/*.md`. No content is invented. Existing agent-unique content (error handling, output envelope) is condensed and moved to a Quick Reference section at the bottom. README-duplicate content is removed or collapsed to pointers.

**Tech Stack:** Markdown only. No code changes.

**Spec:** `docs/superpowers/specs/2026-05-21-agents-md-improvement-design.md`

---

## Files

- **Modify:** `agents.md` (repo root) — complete rewrite
- **Read only (source, do not modify):**
  - `docs/metrics/flow-tension.md` — patterns 1–4
  - `docs/metrics/stablecoin-power.md` — pattern 5
  - `docs/metrics/market-breadth.md` — patterns 6–7
  - `docs/metrics/momentum-divergence.md` — patterns 8–12
  - `docs/metrics/market-regime.md` — patterns 13–16

---

### Task 1: Overwrite agents.md with new content

This is a complete replacement. Write the full file in one operation. All content is specified below — do not paraphrase, do not shorten, do not invent.

**Files:**
- Modify: `agents.md`

- [ ] **Step 1: Write the complete new agents.md**

  Use the Write tool to overwrite `agents.md` with exactly this content:

  ```markdown
  # Agent Guide: cryptospect-cli

  Reasoning reference for LLMs using this tool. For CLI flags, config, and per-metric output schemas see [README.md](README.md) and [`docs/metrics/<name>.md`](docs/metrics/).

  ## Intent → Metric Map

  Run the minimal metric subset for the question type. Running all 10 every call is wasteful.

  | Question Type | Run These | Why |
  |---|---|---|
  | Good time to enter / buy? | `mr`, `sp`, `ft`, `mb` | Macro regime + fuel check + kinetic signals + participation depth |
  | Is this rally real / sustainable? | `mb`, `md`, `ft` | Ghost Rally check + rotation depth + CVD/OI confirmation |
  | Should I rotate into alts? | `mr`, `md`, `mb`, `sp` | Regime label + tier rotation + breadth + dry powder |
  | Macro risk level? | `mr`, `sp`, `ft` | Regime matrix + capital flight check + leverage/funding state |
  | Leverage crowded / squeeze risk? | `ft` | Funding rate + OI + CVD — no other metric needed |
  | Is capital leaving crypto entirely? | `sp`, `mr` | Supply trend contracting + Flight to Safety regime |
  | Accumulation or distribution? | `ft`, `mb`, `sp`, `lp` | OI/CVD + breadth trend + dry powder + conviction ratio |
  | Drill-down on a specific signal | That metric alone | Run `mr` first for macro context if not already in hand |

  ## Named Signal Combinations

  Recognize these patterns by name. Where multiple metric docs name the same pattern they are merged below.

  | Pattern | Triggers | Response |
  |---|---|---|
  | **Early Bull Phase** | `ft` funding `negative→neutral` + CVD `aggressive_buy` | Sellers exhausted, buyers regaining control. Confirm with `sp` dry powder before acting. |
  | **Building Tension** | `ft` OI `building` + price flat + CVD `neutral` | Volatility breakout loading in either direction. No directional position — wait for CVD resolution. |
  | **Long Squeeze Risk** | `ft` funding `overheated` + CVD fading or `aggressive_sell` | Crowded longs at risk of forced unwind. Reduce or hedge long exposure. |
  | **Deleveraging** | `ft` OI `unwinding` + CVD `aggressive_sell` | Flush in progress. Cross-check `sp` — if `supply_trend_7d` expanding, flush may be a buying opportunity. |
  | **Capital Flight / Macro Exodus** | `sp` ratio `< 0.07` + `supply_trend_7d: "contracting"` | Net redemptions — capital leaving crypto entirely. Full defensive posture, not just caution. |
  | **Ghost Rally** | `mb` `divergence_detected: true` (BTC up >2%, alts not following) | `divergence_detected` overrides the classification label. Do not enter broad-market longs regardless of base label. Severity scales with `btc_change_24h_pct` — 2.1% is borderline caution, 6.5%+ is a hard block. |
  | **Ghost Rally Amplified** | `mb` `divergence_detected: true` + `md` `top_heavy` (green day) + `mb` `narrow` | Both metrics confirm concentration. Altcoin longs carry maximum relative underperformance risk. |
  | **Max Conviction Bull** | `mb` `broad` + `sp` High + `md` `risk_on` + `tail_extension: true`; OR `mr` `BTC-Led Expansion` + `sp` High + `ft` funding `neutral` | Full rotation, fuel available, leverage not yet crowded. Strongest configuration for initiating broad exposure. |
  | **Pre-Rotation Coil** | `md` `neutral` + `sp` High + `ft` OI `building` | Dry powder present, leverage loading, no rotation confirmed. Watch for `md` `mid_vs_large > +5pp` as ignition confirmation. |
  | **Blow-Off Warning** | (`md` `risk_on` + `small_vs_mid < 0` + `ft` funding `overheated`) OR (`mr` `Alt-Season/Mania` + `ft` funding `overheated` + `sp` Low) | Rotation stalling or fuel depleted with crowded leverage. Late-cycle peak configuration. Reduce exposure. |
  | **Macro Risk-Off** | (`md` `top_heavy` red day + `ft` OI `unwinding` + `sp` `supply_trend_7d: "contracting"`) OR (`mr` `Flight to Safety` + `sp` `supply_trend_7d: "contracting"`) | Capital fleeing both alts and crypto entirely. Full defensive posture. |
  | **Structural Decay** | `mb` `narrow` + `ft` CVD `aggressive_sell` | Market thinning with aggressive sellers. Active defensive positioning; concentrate risk in BTC/ETH only. |
  | **Pressure Cooker** | `mr` `regime: "Consolidation"` + `conviction: "high"` + `ft` OI `building` | High capital turnover, leverage loading, no directional resolution. Do not initiate positions — wait for regime shift. Likely violent break. |
  | **Confirmed Broad Selloff** | `mr` `Structural Decay` + `md` `flight_to_safety` | Both macro matrix and tier rotation confirm broad selloff. Highest urgency for defensive repositioning across the alt tier. |
  | **Capitulation Floor Confirmation** | `mr` `Capitulation` + `sp` High + `ft` OI `unwinding` + `ft` funding `negative` | All four conditions required — any one alone is insufficient. Closest the suite gets to a validated accumulation entry signal. |
  | **Barbell / Speculative Extension** | `md` `neutral` + `tail_extension: true` | Long-tail moving without confirmed rotation. Non-action state unless corroborated by rising `mb` timeframe spread. Inspect `tier_detail.small` for outlier concentration before acting. |

  ## Response Construction Rules

  **Signal hierarchy — apply in order:**
  1. Check `status` on every result first. `degraded` or `unavailable` changes the answer before any data field is read.
  2. Scan Named Signal Combinations. If 2+ patterns agree → conclude, name the pattern. If patterns conflict → name the conflict explicitly; do not average signals away.
  3. Apply fuel-before-direction: check `sp.supply_trend_7d` before any directional call. A technically bullish setup with contracting stablecoin supply is a different answer than one with high dry powder.
  4. Read individual metric fields for aspects not covered by a named pattern.

  **Confidence field semantics — not uniform across metrics:**

  | Metric | `confidence` means |
  |---|---|
  | `lp`, `sp` | Cross-source validator agreement (data quality) |
  | `ft` | Signal completeness — all 3 signals present; NOT cross-source agreement (no validator exists) |
  | `mb` | Validator directional consensus + candle freshness; `low` = validator skipped, breadth score unaffected |
  | `md` | Tier completeness — ≥3 coins per tier computed |

  Treating all `confidence: "low"` fields the same will misread `mb` and `ft`.

  **Gotchas:**

  1. **`ft` has no composite score.** Three co-equal signals. Never summarize `flow-tension` as a single number. Use the `summary` field — do not re-derive it.

  2. **`ft` OI `"stable"` on first run is a cold-start artifact.** `change_pct_24h` is absent on first run; hook defaults to `"stable"`. Not a real signal — do not act on it if `open_interest.change_pct_24h` is missing from output.

  3. **`mb` `confidence: "low"` does not mean the breadth score is wrong.** It means the Binance validator was skipped (stale candle or parse failure). The breadth score is unaffected. Do not downgrade confidence in the breadth reading because of this flag.

  4. **`md` `top_heavy` / `flight_to_safety` dead band.** When `tier_averages.large` is within ±0.5% of zero, treat both labels as a single **Concentration** regime. The label flip at exactly zero is noise, not a signal.

  5. **`sp` `low` has two opposite meanings.** Always report `supply_trend_7d` alongside it: `stable`/`expanding` → Overextended (volatile market outgrew fuel); `contracting` → Capital Flight (money leaving crypto). Never report "low stablecoin power" without the supply trend.

  6. **CVD is taker aggression, not coins moving to exchanges.** It measures buy/sell imbalance within Binance-US spot. Do not describe it to users as "coins moving onto exchanges."

  ## Quick Reference

  ### Metrics

  | Metric | Alias | Unique Flags |
  |---|---|---|
  | `liquidity-pulse` | `lp` | — |
  | `stablecoin-power` | `sp` | `--top N` |
  | `flow-tension` | `ft` | — |
  | `market-breadth` | `mb` | `--top N` |
  | `momentum-divergence` | `md` | `--segments N` |
  | `market-regime` | `mr` | — |
  | `dominance` | `dom` | — |
  | `volatility` | `vol` | — |
  | `fear-greed-index` | `fgi` | — |
  | `china-m2` | `cnm2` | — |

  All metrics: `--detail basic|extended|full`. Use `--detail extended` minimum for `mr` — basic suppresses `notes`, `confidence`, and `dominance_cold_start`.

  Utility: `list-metrics`, `cache-clear`

  ### Output Envelope

  Every invocation returns this structure on stdout:

      {
        "status": "ok|error",
        "ts": 1744444800,
        "results": [
          {
            "metric": "liquidity-pulse",
            "status": "ok|degraded|unavailable",
            "data": { ... },
            "meta": { ... }    // omitted at --detail basic
          }
        ],
        "error": { "code": 429, "msg": "rate_limited", "retry_after_sec": 60, "source": "coingecko" }
      }

  - `--detail basic` (default): `meta` omitted
  - `--detail extended`: `meta` includes cache hit, TTL, source timestamps
  - `--detail full`: `meta` adds thresholds and metric description

  ### Error Handling

  - On 429: parse `retry_after_sec`, sleep, retry
  - On error: check `error.source` to identify which API failed
  - Never parse stderr — debug logs only
  - Exit 0 for success and handled errors; non-zero only for unrecoverable failures

  ### API Key

  Precedence: `--api-key` flag > `CRYPTOSPECT_COINGECKO_KEY` / `CRYPTOSPECT_BINANCE_KEY` env vars > `~/.cryptospect.yaml`. Full config: [README.md](README.md).

  ### Further Reference

  - CLI flags, config, caching, data sources: [README.md](README.md)
  - Per-metric output schemas, classification thresholds, field definitions: [`docs/metrics/<name>.md`](docs/metrics/)
  ```

- [ ] **Step 2: Verify section headers are present and in correct order**

  Run:
  ```bash
  grep "^## " agents.md
  ```

  Expected output (exactly this order):
  ```
  ## Intent → Metric Map
  ## Named Signal Combinations
  ## Response Construction Rules
  ## Quick Reference
  ```

- [ ] **Step 3: Verify all 16 patterns are present**

  Run:
  ```bash
  grep "^\| \*\*" agents.md | wc -l
  ```

  Expected: `16`

- [ ] **Step 4: Verify key gotcha content is present**

  Run:
  ```bash
  grep -c "cold-start\|composite score\|taker aggression\|dead band\|two opposite\|breadth score" agents.md
  ```

  Expected: `6` (one match per gotcha)

- [ ] **Step 5: Verify old content is gone**

  Run:
  ```bash
  grep -c "Future Per.Asset\|Design.Decisions\|cryptospect-cli regime\|cryptospect-cli zscore" agents.md
  ```

  Expected: `0`

- [ ] **Step 6: Spot-check one pattern against its source doc**

  Verify the **Capitulation Floor Confirmation** triggers match `docs/metrics/market-regime.md`:
  ```bash
  grep -A3 "Capitulation Floor" docs/metrics/market-regime.md
  ```

  The source doc should read: `mr Capitulation + sp High + ft OI unwinding + ft funding negative`. Confirm the agents.md row matches.

- [ ] **Step 7: Check file size is reasonable**

  Run:
  ```bash
  wc -l agents.md
  ```

  Expected: between 130 and 160 lines. If significantly outside this range, something went wrong in the write step — re-check that the full content was written.

- [ ] **Step 8: Commit**

  ```bash
  git add agents.md
  git commit -m "docs: restructure agents.md as reasoning guide

  - Add Intent → Metric Map (8 question types → minimal metric subset)
  - Add Named Signal Combinations (16 cross-metric patterns extracted from metric docs)
  - Add Response Construction Rules with confidence semantics table and 6 gotchas
  - Condense existing CLI/envelope/error content into Quick Reference section
  - Remove future per-asset commands (planned roadmap, not implemented)
  - Remove Design-Decisions.md reference (not useful to agents)
  - Add pointers to README.md and docs/metrics/ for per-metric detail"
  ```
