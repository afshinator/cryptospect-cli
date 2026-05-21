# Design: agents.md Improvement
**Date:** 2026-05-21  
**Status:** Approved

## Goal

Transform `agents.md` from a syntax reference into a reasoning guide. An LLM given only `agents.md` should be able to: choose which metrics to run for a given question, recognize and name the 16 cross-metric patterns, avoid 6 documented gotchas, and construct grounded answers — without reading the individual metric docs.

Secondary goal: minimize file size. Operational detail that doesn't need to be in context every call lives in `README.md` and `docs/metrics/<name>.md`, with explicit pointers.

## What Is Not Changing

All existing agent-unique content is preserved, condensed, and moved to the Quick Reference section at the bottom. Nothing is deleted that an agent needs. What is removed:
- Future per-asset commands (planned, not implemented — agents must not act on roadmap)
- "Source of truth: Design-Decisions.md" header note (not useful to agents)
- Verbose block-format CLI signatures (replaced by compact table)
- Detailed API key config (collapsed to one line + README pointer)

## File Structure

```
# Agent Guide: cryptospect-cli
[2-line purpose statement]

## Intent → Metric Map
## Named Signal Combinations
## Response Construction Rules
## Quick Reference
```

Reasoning sections come first. Quick Reference is the last section — agents that already know how to invoke the tool can skip it.

**Target size:** ~150 lines.

---

## Section Designs

### § Intent → Metric Map

A markdown table mapping 8 question types to the minimal metric subset to run.

| Question Type | Run These | Why |
|---|---|---|
| Good time to enter / buy? | `mr`, `sp`, `ft`, `mb` | Macro regime + fuel check + kinetic signals + participation depth |
| Is this rally real / sustainable? | `mb`, `md`, `ft` | Ghost Rally check + rotation depth + CVD/OI confirmation |
| Should I rotate into alts? | `mr`, `md`, `mb`, `sp` | Regime label + tier rotation + breadth + dry powder |
| Macro risk level? | `mr`, `sp`, `ft` | Regime matrix + capital flight check + leverage/funding state |
| Leverage crowded / squeeze risk? | `ft` | Funding rate + OI + CVD — no other metric needed |
| Is capital leaving crypto entirely? | `sp`, `mr` | Supply trend contracting + Flight to Safety regime |
| Accumulation or distribution? | `ft`, `mb`, `sp`, `lp` | OI/CVD + breadth trend + dry powder + conviction ratio |
| Drill-down on a specific signal | That metric alone | Already have macro context from a prior `mr` call |

---

### § Named Signal Combinations

16 patterns extracted and deduplicated from the Agentic Logic sections of all 5 implemented metric docs. Where multiple docs name the same pattern (e.g. Macro Risk-Off in `mb` and `mr`), they are consolidated to one row with the union of triggers.

Format: table with columns `Pattern | Triggers | Response`.

**Patterns and their source docs:**

1. **Early Bull Phase** (ft)  
   Triggers: `ft` funding `negative→neutral` + CVD `aggressive_buy`  
   Response: Sellers exhausted, buyers regaining control. Confirm with `sp` dry powder before acting.

2. **Building Tension** (ft)  
   Triggers: `ft` OI `building` + price flat + CVD `neutral`  
   Response: Volatility breakout loading. No directional position — wait for CVD to resolve.

3. **Long Squeeze Risk** (ft)  
   Triggers: `ft` funding `overheated` + CVD fading or `aggressive_sell`  
   Response: Crowded longs at risk of forced unwind. Reduce or hedge long exposure.

4. **Deleveraging** (ft)  
   Triggers: `ft` OI `unwinding` + CVD `aggressive_sell`  
   Response: Flush in progress. Cross-check `sp` — if expanding, flush may be a buying opportunity.

5. **Capital Flight / Macro Exodus** (sp)  
   Triggers: `sp` `stable_power_ratio < 0.07` + `supply_trend_7d: "contracting"`  
   Response: Net redemptions — capital leaving crypto entirely. Full defensive posture, not just caution.

6. **Ghost Rally** (mb)  
   Triggers: `mb` `divergence_detected: true` (BTC up >2%, alts not following)  
   Response: `divergence_detected` overrides classification label. Do not enter broad-market longs regardless of base label.

7. **Ghost Rally Amplified** (mb + md)  
   Triggers: `mb` `divergence_detected: true` + `md` `top_heavy` (green day) + `mb` `narrow`  
   Response: Both metrics confirm concentration. Altcoin longs carry maximum relative underperformance risk.

8. **Max Conviction Bull** (mb + md + mr)  
   Triggers: `mb` `broad` + `sp` High + `md` `risk_on` + `tail_extension: true` OR `mr` `BTC-Led Expansion` + `sp` High + `ft` funding `neutral`  
   Response: Full rotation, fuel available, leverage not crowded. Strongest configuration for initiating broad exposure.

9. **Pre-Rotation Coil** (md)  
   Triggers: `md` `neutral` + `sp` High + `ft` OI `building`  
   Response: Dry powder present, leverage loading, no rotation yet. Watch for `md` `mid_vs_large` breaking above +5pp as ignition confirmation.

10. **Blow-Off Warning** (md + mr)  
    Triggers: (`md` `risk_on` + `small_vs_mid < 0` + `ft` funding `overheated`) OR (`mr` `Alt-Season/Mania` + `ft` funding `overheated` + `sp` Low)  
    Response: Rotation stalling or fuel depleted with crowded leverage. Late-cycle peak configuration. Reduce exposure.

11. **Macro Risk-Off** (mb + mr)  
    Triggers: (`md` `top_heavy` red day + `ft` OI `unwinding` + `sp` `supply_trend_7d: "contracting"`) OR (`mr` `Flight to Safety` + `sp` `supply_trend_7d: "contracting"`)  
    Response: Capital fleeing both alts and crypto entirely. Full defensive posture.

12. **Structural Decay** (mb)  
    Triggers: `mb` `narrow` + `ft` CVD `aggressive_sell`  
    Response: Market thinning with aggressive sellers. Active defensive positioning; concentrate risk in BTC/ETH only.

13. **Pressure Cooker** (mr)  
    Triggers: `mr` `regime: "Consolidation"` + `conviction: "high"` + `ft` OI `building`  
    Response: High volume, flat dominance, leverage loading. Do not initiate directional positions — wait for regime shift. Likely violent break in either direction.

14. **Confirmed Broad Selloff** (mr + md)  
    Triggers: `mr` `Structural Decay` + `md` `flight_to_safety`  
    Response: Highest urgency for defensive repositioning across the alt tier.

15. **Capitulation Floor Confirmation** (mr)  
    Triggers: `mr` `Capitulation` + `sp` High + `ft` OI `unwinding` + `ft` funding `negative`  
    Response: Closest the suite gets to a validated accumulation entry. All four conditions required — any one alone is insufficient.

16. **Barbell / Speculative Extension** (md)  
    Triggers: `md` `neutral` + `tail_extension: true`  
    Response: Long-tail moving without confirmed rotation. Non-action state unless corroborated by rising `mb` timeframe spread. Inspect `tier_detail.small` for outlier concentration before acting.

---

### § Response Construction Rules

Five rules and six gotchas. All prescriptive.

**Hierarchy:**
1. Check `status` on every result first. `degraded` or `unavailable` changes the answer before any data is read.
2. Scan Named Signal Combinations. If 2+ patterns agree → conclude using pattern name. If patterns conflict → name the conflict explicitly; do not average signals.
3. Apply the fuel-before-direction rule: check `sp.supply_trend_7d` before any directional call. A technically bullish setup with contracting stablecoin supply is a different answer than one with high dry powder.
4. Read individual metric fields for any aspect not covered by a named pattern.

**Confidence field semantics — vary by metric:**
- `lp` / `sp`: cross-source validator agreement (data quality)
- `ft`: signal completeness — all 3 signals present (not cross-source agreement; no validator exists)
- `mb`: validator directional consensus + candle freshness; `low` = validator skipped, not that breadth score is wrong
- `md`: tier completeness — ≥3 coins per tier

Treating all `confidence: "low"` fields the same will misread `mb` and `ft`.

**Unimplemented metrics:** `dom`, `vol`, `fgi`, `cnm2` are v1 implemented but may return `status: unavailable` on API failure. If unavailable, note the gap if relevant; do not attempt to interpret output.

**Gotchas:**

1. **`ft` has no composite score.** Three co-equal signals. Never summarize `flow-tension` as a single number. Use the `summary` field — do not re-derive it.

2. **`ft` OI `stable` on first run is a cold-start artifact.** `change_pct_24h` is absent; hook defaults to `"stable"`. Not a real signal. Do not act on OI `"stable"` if no prior cache exists (no `change_pct_24h` in output).

3. **`mb` `confidence: "low"` + `discrepancy_detected: true` ≠ breadth score wrong.** It means the Binance validator was skipped (stale candle or parse failure). The breadth score is unaffected. Do not downgrade confidence in the breadth reading because of this flag.

4. **`md` `top_heavy` / `flight_to_safety` dead band.** When `tier_averages.large` is within ±0.5% of zero, treat both labels as a single **Concentration** regime. The label flip at exactly zero is noise, not a signal.

5. **`sp` `low` has two opposite meanings.** Always report `supply_trend_7d` alongside it: `stable` or `expanding` → Overextended (volatile market outgrew fuel); `contracting` → Capital Flight (money leaving crypto). These require opposite urgency levels.

6. **CVD is taker aggression, not coins moving to exchanges.** It measures buy/sell imbalance within Binance-US spot. Do not describe it to users as "coins moving onto exchanges" — that requires on-chain data not available in this tool.

---

### § Quick Reference

**Invocation table** — compact, one row per metric:

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

All metrics accept `--detail basic|extended|full`. Use `--detail extended` minimum for `mr` (suppresses `notes` and `confidence` at basic).

**Utility:** `list-metrics`, `cache-clear`

**Output envelope** — abbreviated JSON with key fields annotated. Keep existing JSON block, trim to essential structure.

**Error handling** — keep existing 4 bullets verbatim.

**API key:** One line — "Precedence: `--api-key` flag > `CRYPTOSPECT_COINGECKO_KEY` / `CRYPTOSPECT_BINANCE_KEY` env vars > `~/.cryptospect.yaml`. Full config reference: `README.md`."

**Pointers:**
- CLI flags, config, caching, data sources: `README.md`
- Per-metric output schemas, classification thresholds, field definitions: `docs/metrics/<name>.md`

---

## Implementation Notes

- This is pure documentation — no code changes.
- All content is sourced from existing metric docs; nothing is invented.
- The 16 named patterns are extracted verbatim from Agentic Logic sections of: `flow-tension.md`, `stablecoin-power.md`, `market-breadth.md`, `momentum-divergence.md`, `market-regime.md`.
- Where the same pattern is named in multiple docs, they are merged to one row with the union of triggers — do not list duplicates.
- Tone: terse, technical, prescriptive. No hedging prose. Tables and definition lists over paragraphs.
- All 10 metrics are v1-complete and return real data.
