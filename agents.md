# Agent Guide: cryptospect-cli

Reasoning reference for LLMs using this tool. For CLI flags, config, and per-metric output schemas see [README.md](README.md) and [`docs/metrics/<name>.md`](docs/metrics/).

---

## How to Read Any Metric

Every metric in this suite follows the same interpretation loop. Internalize this and you can reason correctly with any output — including partial or degraded data — without needing a pre-written example for the specific situation.

**Step 1 — Check `status` before reading any data field.**

| `status` | Meaning | Action |
|----------|---------|--------|
| `ok` | All signals computed normally | Proceed |
| `degraded` | Primary data returned but incomplete — one or more signals are missing or stale | Read what's available; note what's absent; lower your confidence proportionally |
| `unavailable` | Primary data fetch failed entirely | Do not interpret data fields; report the gap; suggest `cache-clear` or retry |

Never read `data` or `meta` fields before checking `status`. A `degraded` metric that looks clean can produce misleading signals — for example, `ft` OI hook defaulting to `"stable"` on cold start when no cached prior value exists.

**Step 2 — Read `classification.label` for the verdict.**

Every metric (except `ft`, which has no composite label) produces a single categorical verdict in `data.classification.label`. This is the conclusion. Everything else is evidence for or against it.

**Step 3 — Read `data.summary` for the plain-language version.**

The `summary` field is pre-synthesized from all signals. For `ft` especially — which has three co-equal signals and no composite label — use `summary` directly rather than re-deriving it yourself.

**Step 4 — Check `meta.confidence` to calibrate trust — but know what it means per metric.**

`confidence` is not uniform. The same `"low"` value means different things depending on which metric emitted it:

| Metric | `confidence: "low"` means |
|--------|--------------------------|
| `lp`, `sp` | Cross-source validator disagreed — data quality concern |
| `ft` | One or more of the three signals is missing (OI/funding transient failure) — not a data quality issue, a completeness issue |
| `mb` | Binance candle validator was skipped — **breadth score is unaffected**; do not downgrade the breadth reading |
| `md` | A tier had fewer than 3 valid coins — tier average for that tier is unreliable |
| `mr` | Dominance cold start (no prior snapshot for delta) — regime label valid, but dominance trend defaults to neutral |

Once you've run these four steps for each metric in your set, apply the signal hierarchy in Response Construction Rules below.

---

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

---

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

---

## Response Construction Rules

**Signal hierarchy — apply in order:**
1. Check `status` on every result first. `degraded` or `unavailable` changes the answer before any data field is read.
2. Scan Named Signal Combinations. If 2+ patterns agree → conclude, name the pattern. If patterns conflict → name the conflict explicitly; do not average signals away.
3. Apply fuel-before-direction: check `sp.supply_trend_7d` before any directional call. A technically bullish setup with contracting stablecoin supply is a different answer than one with high dry powder.
4. Read individual metric fields for aspects not covered by a named pattern.

Treating all `confidence: "low"` fields the same will misread `mb` and `ft`.

**Gotchas:**

1. **`ft` has no composite score.** Three co-equal signals. Never summarize `flow-tension` as a single number. Use the `summary` field — do not re-derive it.

2. **`ft` OI `"stable"` on first run is a cold-start artifact.** `change_pct_24h` is absent on first run; hook defaults to `"stable"`. Not a real signal — do not act on it if `open_interest.change_pct_24h` is missing from output.

3. **`mb` `confidence: "low"` does not mean the breadth score is wrong.** It means the Binance validator was skipped (stale candle or parse failure). The breadth score is unaffected. Do not downgrade confidence in the breadth reading because of this flag.

4. **`md` `top_heavy` / `flight_to_safety` dead band.** When `tier_averages.large` is within ±0.5% of zero, treat both labels as a single **Concentration** regime. The label flip at exactly zero is noise, not a signal.

5. **`sp` `low` has two opposite meanings.** Always report `supply_trend_7d` alongside it: `stable`/`expanding` → Overextended (volatile market outgrew fuel); `contracting` → Capital Flight (money leaving crypto). Never report "low stablecoin power" without the supply trend.

6. **CVD is taker aggression, not coins moving to exchanges.** It measures buy/sell imbalance within Binance-US spot. Do not describe it to users as "coins moving onto exchanges."

---

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
- Daily cold-start sequence and degraded-data reasoning: [DAILY_BRIEF.md](DAILY_BRIEF.md)

---

## Repository Quality Gate

This repo uses [`mcp-server-go-quality`](https://github.com/afshinator/mcp-server-go-quality) for agent-driven quality sweeps. CI enforces golangci-lint (v2.11.4); govulncheck and nilaway are available via the MCP server for session-level checking.

**MCP configuration** lives in `.mcp.json` at the repo root. Any MCP-compatible agent client picks it up automatically.

**Pre-flight:** Call `install_tools` once at session start to pre-install the pinned tool binaries (no-op if already present). Then call `run_code_checks` with `project_path` set to the repo root to sweep all three checkers.

**Interpreting results:** Check `error` first (non-empty = tool failure), then navigate `file:line:column`. The `native` field carries full raw output for remediation. The `severity` field is only present for golangci-lint; govulncheck and nilaway have no severity concept.

**Verification:** `run_code_checks` golangci-lint output must agree with `go tool golangci-lint run ./...` (currently 0 issues). Disagreement indicates a version mismatch or tool installation problem.