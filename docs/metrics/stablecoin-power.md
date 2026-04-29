# stablecoin-power

**version:** `v1.0.0`
**Alias:** `sp`
**Endpoints:** main: `coingecko.global_market`, `coingecko.stables_markets`, supplementary: `stablecoins.llama.fi/stablecoins`

## Overview

Measures the ratio of aggregate stablecoin market cap (Top 8+ by size) to the market cap of all volatile crypto assets (total market cap minus stablecoin market cap). Functions as an "Ecosystem Fuel Gauge" — a high ratio indicates substantial sidelined capital ("dry powder") available to enter the market; a low ratio indicates the market is either fully deployed or, in a more severe scenario, that capital is actively exiting crypto entirely. The two Low scenarios are structurally distinguishable via the `supply_trend_7d` field.

## Formula

```
stablecoin_power = stable_mcap_usd / (total_market_cap_usd - stable_mcap_usd)
```

The denominator isolates volatile assets by subtracting stablecoin supply from total market cap. This prevents stablecoin supply growth from inflating the denominator and masking a genuine increase in dry powder — the ratio cleanly answers "how much stable capital exists relative to the volatile market it could deploy into?"

## Interpretation

- **High (>0.15):** Large dry powder reserve relative to volatile market size — market has meaningful fuel for continued rallies. Sidelined capital is high; conditions are favorable for sustained price appreciation if conviction metrics confirm.
- **Normal (0.07–0.15):** Healthy balance between deployed and sidelined capital. Market is adequately fueled for current price levels.
- **Low (<0.07):** Dry powder is depleted relative to volatile market size. Two distinct scenarios apply — disambiguated by `supply_trend_7d`:
  - `supply_trend_7d: "stable"` or `"expanding"` → **Overextended:** volatile assets grew faster than stablecoin supply. Stables weren't redeemed; they became a smaller slice of a bigger pie. Rally ran on fumes. Risk of sharp correction if no new capital enters.
  - `supply_trend_7d: "contracting"` → **Capital Flight / Macro Exodus:** stablecoin supply itself is shrinking due to redemptions and burns. Capital is leaving crypto entirely, not rotating into volatile assets. This is the more severe scenario — worse than overextended.

`supply_trend_7d` is always present in the `data` block to support this disambiguation at all detail levels.

## Classification

| Condition | Threshold |
|-----------|-----------|
| High | >= 0.15 |
| Normal | 0.07 – 0.15 |
| Low | < 0.07 |

## Data Source(s)

- **Primary API:** CoinGecko
- **Endpoints:** `/global` (for `total_market_cap["usd"]`), `/coins/markets?category=stablecoins` (for per-stablecoin market cap, top N by size)
- **Fields used:** `total_market_cap["usd"]`, `market_cap` per stablecoin (summed for top N)
- **Secondary API:** DefiLlama — `stablecoins.llama.fi/stablecoins`
- **Secondary fields used:** aggregate circulating supply (for cross-check), 7-day supply change (for `supply_trend_7d`)

## Cross-Source Verification

This metric uses the **Dual-Source Aggregate Supply Verification** pattern.

| Role | Source | Purpose |
|------|--------|---------|
| Primary | CoinGecko | Main metric computation (top N stablecoin market caps + global market cap) |
| Validator | DefiLlama | Aggregate on-chain supply cross-check; 7d supply trend for Low disambiguation |

**Why DefiLlama, not Binance-US or CoinMarketCap:**
Binance-US reports trading volume and spot prices, not stablecoin circulating supply — there is no field to compare against `stable_mcap_usd`. CoinMarketCap tracks only ~150 stablecoins vs. ~300 for CoinGecko and DefiLlama, introducing a systematic undercount that would make the discrepancy threshold unreliable. DefiLlama reads stablecoin supply directly from on-chain token contracts — a genuinely different collection methodology from CoinGecko's exchange-feed aggregation — making it the appropriate independent validator for a supply-side metric.

**Important caveat:** DefiLlama uses CoinGecko's API for token pricing. This means the two sources share a pricing layer, so discrepancies will almost always reflect *coverage scope differences* (one source tracking a stablecoin the other doesn't yet include) rather than a fundamental data error. A flagged discrepancy should be interpreted as "coverage drift," not "bad data."

**Anchor asset:** Aggregate stablecoin supply (USDT, USDC, DAI, USDe, FDUSD, PYUSD, USDS, and others in the Top N)

**Discrepancy threshold:** 15%

**Behavior on mismatch:** Metric value is always derived from CoinGecko primary. If DefiLlama's aggregate supply differs by >15% from CoinGecko's summed top-N supply, `discrepancy_detected` is set to `true` and `confidence` is adjusted:
- Discrepancy < 15%: `confidence: "high"`
- Discrepancy 15–25%: `confidence: "medium"`
- Discrepancy >= 25%: `confidence: "low"`

`discrepancy_note` is included in `meta` when `discrepancy_detected` is `true`, and will describe the magnitude and likely cause (e.g., "Coverage scope difference: DefiLlama total exceeds CoinGecko top-8 by 18.2%; a recently-launched stablecoin may not yet appear in CoinGecko's category index.").

**Note on `supply_trend_7d`:** This field is derived from the DefiLlama validator fetch (7-day aggregate supply change) and is always present in `data` regardless of `discrepancy_detected`. It is a primary output, not a diagnostic — see Interpretation section for its role in disambiguating the Low classification.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--top` | `int` | `8` | Number of top stablecoins by market cap to include in the numerator. Minimum enforced: 8. |

**Minimum enforcement:** If `--top` is passed with a value less than 8, the CLI clamps the value to 8 and adds a note to the output:
```json
"top_clamped": true,
"top_clamped_reason": "Minimum 8 stablecoins required for metric integrity. Value adjusted from N to 8."
```
The top 5 stablecoins represent approximately 90% of total stablecoin supply; values below 8 risk excluding material supply from USDe, FDUSD, PYUSD, and exchange-backed stables, producing a hollow signal. The minimum of 8 is a data integrity floor, not an arbitrary limit.

## Output Schema

```json
{
    "metric":  "stablecoin-power",
    "version": "v1.0.0",
    "status":  "string",      // "ok" / "degraded" / "unavailable"

    "data": {
        "stable_power_ratio":  "float64",  // the core metric value
        "stable_mcap_usd":     "float64",  // aggregate stablecoin market cap (top N)
        "volatile_mcap_usd":   "float64",  // total_market_cap - stable_mcap
        "supply_trend_7d":     "string",   // "expanding" / "stable" / "contracting"
                                           // always present; critical for Low disambiguation
        "stablecoins_counted": "int",      // actual N used (reflects --top or clamp)
        "classification": {
            "label":       "string",  // "high" / "normal" / "low"
            "description": "string"   // e.g. "Dry Powder Alert" / "Healthy Balance" / "Overextended" / "Capital Flight"
        },
        "summary": "string"  // NL sentence: ratio + classification + supply_trend_7d if low
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":            "bool",
        "ttl_remaining_sec":    "int",
        "primary_source":       "coingecko",
        "validator_source":     "defillama",
        "discrepancy_detected": "bool",
        "discrepancy_note":     "string",   // only if discrepancy_detected == true
        "confidence":           "string",   // "high" / "medium" / "low"
        "top_clamped":          "bool",     // only if --top was passed below minimum
        "top_clamped_reason":   "string",   // only if top_clamped == true
        "stablecoin_scope":     "string",   // "all_usd_equivalent" — see Implementation Compromises
        "supply_trend_source":  "defillama" // documents that supply_trend_7d derives from validator
        // Additionally when --detail full:
        // "thresholds": { "high": 0.15, "normal_min": 0.07, "low": 0.07 }
        // "description": "string"  // full Long Description text
        // "top_n_stablecoins": [ { "symbol": "USDT", "mcap_usd": 186000000000 }, ... ]
        //   ^ list of each stablecoin included in the numerator with its individual market cap
    }
}
```

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `supply_trend_7d` | Always | 7-day direction of aggregate stablecoin supply: `"expanding"` / `"stable"` / `"contracting"`. Always present in `data` to support Low disambiguation. Derived from DefiLlama validator fetch. |
| `delta_24h` | Prior cache data exists | Percentage change in `stable_power_ratio` from 24h ago. Indicates momentum in dry powder accumulation or depletion. Not yet implemented in v1.0 — requires historical cache. |
| `discrepancy_note` | `discrepancy_detected == true` | Human-readable explanation of the coverage scope difference between CoinGecko and DefiLlama. |
| `top_clamped` / `top_clamped_reason` | `--top` passed below minimum of 8 | Indicates value was adjusted and explains why. |
| `top_n_stablecoins` | `--detail full` | Full breakdown of each stablecoin included in the numerator with individual market cap. |

## Usage

```bash
# Basic
cryptospect-cli stablecoin-power

# With alias
cryptospect-cli sp

# Extended detail
cryptospect-cli stablecoin-power --detail extended

# Full detail — includes thresholds, description, per-stablecoin breakdown
cryptospect-cli stablecoin-power --detail full

# Custom top N (minimum 8 enforced)
cryptospect-cli stablecoin-power --top 12

# Combined
cryptospect-cli stablecoin-power --detail full --top 12
```

## Long Description

### High-level meaning and value

This metric functions as the "Ecosystem Fuel Gauge." It answers: *"Is there enough cash on the sidelines to sustain or initiate a rally?"*

Stablecoins are the primary store of sidelined capital within the crypto ecosystem. Unlike fiat sitting in a bank account, stablecoins are already on-chain and can be deployed into volatile assets within seconds, on any DEX, at any time. The ratio of stablecoin supply to volatile asset market cap therefore represents the immediately accessible fuel available to push prices higher — or the absence of it.

A high ratio means the market has substantial reserve capacity: even a small rotation from stablecoins into volatile assets can produce significant upward pressure. A low ratio means the market has already burned most of its fuel — further price appreciation requires fresh capital inflows from fiat, not just on-chain rotation.

### Exact definition and data needs, logic

**Formula:** `stable_mcap_usd / (total_market_cap_usd - stable_mcap_usd)`

**Denominator isolation:** Subtracting stablecoin supply from total market cap in the denominator is deliberate. It isolates the volatile market being "chased" by the dry powder. Without this, a large growth in stablecoin supply would simultaneously increase both numerator and denominator, dampening the signal. With it, the ratio cleanly measures the leverage ratio between deployable cash and the volatile market it can move.

**Top-N selection:** Rather than summing all ~300 stablecoins tracked by CoinGecko, the metric aggregates the top N by market cap (default: 8). The top 5 stablecoins represent approximately 90% of total supply; top 8 captures roughly 93–95%, covering all major assets including USDT, USDC, DAI, USDe, FDUSD, PYUSD, USDS, and the next largest exchange-backed or synthetic stables. Values below 8 risk excluding material supply from newer but significant assets like USDe.

**Stablecoin scope (USD-equivalent):** The metric includes all stablecoins in CoinGecko's stablecoin category — USD-pegged (USDT, USDC, DAI, etc.) and non-USD-pegged (EURC, GBPT, XSGD, etc.) alike. CoinGecko denominates all market caps in USD equivalent, so inclusion is automatic and numerically consistent. Non-USD stablecoins represent approximately 3–4% of total stablecoin supply, so the practical impact is negligible. The design rationale: any stablecoin holder can swap to USDC or USDT on a DEX within seconds and deploy into volatile assets — the friction is near-zero in liquid markets, making all stablecoins genuinely equivalent dry powder for practical purposes.

**`supply_trend_7d`:** Derived from DefiLlama's 7-day aggregate supply change during the validator fetch. Classified as `"expanding"` (>+1%), `"stable"` (-1% to +1%), or `"contracting"` (<-1%). Always present in the data block.

### Possible values and associated verdicts

**High (>0.15) — Dry Powder Alert**
The ecosystem holds substantial sidelined capital relative to volatile market size. Conditions are favorable for continued appreciation: even modest rotation from stablecoins into volatile assets represents meaningful buying pressure. However, this condition can persist for extended periods during bear market accumulation phases, where high dry powder exists but price doesn't move due to low overall conviction. Cross-reference with `liquidity-pulse` before drawing directional conclusions.

**Normal (0.07–0.15) — Healthy Balance**
Adequate fuel for current price levels. Market is neither over-leveraged nor running on empty. No special signal.

**Low (<0.07) — Two distinct scenarios, disambiguated by `supply_trend_7d`:**

- **`supply_trend_7d: "stable"` or `"expanding"` → Overextended / Liquidity Thinning**
  Volatile asset market cap grew faster than stablecoin supply. Stablecoins were not redeemed — they simply became a smaller fraction of a larger pie. This means the rally consumed available dry powder without replenishment. The exit is narrowing: further upward moves require fresh fiat inflows, and a sharp reversal could find limited buying support to absorb selling.

- **`supply_trend_7d: "contracting"` → Capital Flight / Macro Exodus**
  Aggregate stablecoin supply is declining due to net redemptions and burns. This is the more severe scenario: capital is not rotating from stables into volatile assets (which would keep supply flat while volatile assets rose) — it is leaving the crypto ecosystem entirely, being redeemed back to fiat. This is a macro risk-off signal, not simply a "fully deployed" market. An agent encountering this scenario should treat it with higher urgency than simple overextension.

These two scenarios are opposite in implication and require different responses. The metric would be misleading without this disambiguation.

### Cross-Source Verification

CoinGecko aggregates stablecoin supply from exchange price feeds and self-reported circulating supply figures. This is fast and broadly comprehensive but does not independently verify on-chain balances. DefiLlama reads stablecoin supply directly from token contracts across 500+ chains, providing a genuinely different collection methodology.

**What discrepancy actually means here:** Because DefiLlama uses CoinGecko's API for token pricing, the two sources share a pricing layer. Discrepancies almost always reflect coverage scope differences — one source has indexed a recently-launched stablecoin that the other hasn't yet added — rather than a data quality error. The 15% discrepancy threshold is therefore calibrated to catch material coverage drift, not minor counting differences.

**Validation behavior:**
- Fetch DefiLlama aggregate supply in parallel with CoinGecko primary data
- Compare totals; flag `discrepancy_detected: true` if delta > 15%
- Metric value always computed from CoinGecko primary — validator only reports
- `confidence` in metadata reflects validation result
- `supply_trend_7d` is always derived from DefiLlama's 7d change field, independent of discrepancy status

### Other details

**CLI Flags:**
- `--top N`: Controls how many stablecoins are summed for the numerator. Minimum 8 enforced. Values of 10–15 are reasonable for users who want to capture a broader long tail of exchange-backed stablecoins (FDUSD, TUSD, etc.). Very large values (>20) add negligible supply while increasing API call complexity.

**Enhancements:**
- `supply_trend_7d`: Always present. Its primary purpose is to split the Low classification into two actionable verdicts. At Normal and High, it provides useful context on whether dry powder is growing or stable, but does not change the verdict.
- `delta_24h` (not yet implemented): Will show the percentage change in `stable_power_ratio` from the prior cached value. For this metric, percentage change is strongly preferred over absolute difference — a move from 0.08 to 0.09 is a 12.5% increase in dry powder and is far more signal-rich than the absolute 0.01 difference. To be implemented when historical cache is available.

**Implementation Compromises:**
- **Deployed stablecoins inflate the numerator.** Stablecoins deposited into DeFi protocols (Curve pools, Aave, Uniswap liquidity positions) are counted in `stable_mcap_usd` but are not freely available dry powder — they are already deployed. The metric has no way to distinguish "idle" from "deployed" stablecoins in v1. This means the ratio likely overstates true available dry powder, particularly during periods of high DeFi activity. This is a known limitation with no simple fix given available public APIs.
- **Non-USD stablecoins included by default.** EURC, GBPT, XSGD, and other non-USD stables are included at USD-equivalent market cap. Their supply can grow for reasons unrelated to crypto market sentiment — for example, MiCA regulatory compliance driving EURC adoption in Europe. A supply expansion of EURC could cause the ratio to tick upward without any genuine increase in crypto-specific dry powder. This effect is currently minor (~3–4% of total supply) but may grow as non-USD stablecoins expand.
- **Algorithmic and synthetic stablecoins included at market price, not par.** USDe, DAI, and similar assets are included in the numerator at their market price. If any of these depeg, their market price drops below $1, understating true supply-side dry powder. The metric does not filter de-pegged stablecoins in v1.
- **Top-N, not total supply.** The numerator uses the top N stablecoins, not the full universe. Top 8 captures ~93–95% of stablecoin supply. The excluded long tail is intentionally omitted to reduce API surface area and exclude low-liquidity stablecoins whose supply figures are less reliable.

**Future Enhancements:**
- `delta_24h`: Percentage change in ratio from prior cached value. Requires historical cache. See Enhancements section.
- `--usd-only` flag: Restricts the numerator to USD-pegged stablecoins only (USDT, USDC, DAI, etc.), excluding EURC and other non-USD stables. Useful for institutional or precision users who want to isolate USD-denominated dry powder and eliminate FX-driven supply noise. Design choice for v1 is inclusive (all stablecoins) because non-USD stables are trivially swappable to USD equivalents on-chain.
- **Deployed vs. idle stablecoin filtering.** A future enhancement could subtract stablecoins locked in DeFi protocols (available from DefiLlama TVL data) from the numerator to approximate "truly idle" dry powder. This would require a more complex multi-endpoint fetch but would produce a more precise fuel gauge signal.
- **Peg-deviation filtering.** Exclude stablecoins trading >1% off peg from the numerator, since a de-pegged stable is not reliable dry powder. Requires per-stablecoin price data already available from CoinGecko's `/coins/markets` call.

**Agentic Logic (Strategic Notes)**

When an LLM or agent calls this tool, it should use the following heuristics:

- **Fuel Check:** If `stable_power_ratio > 0.15`, the market has meaningful reserve capacity. A sustained rally is mechanically plausible. Verdict: *Dry Powder Present — confirm with `liquidity-pulse` for conviction.*

- **Liquidity Thinning (Overextended):** If `stable_power_ratio < 0.07` AND `supply_trend_7d` is `"stable"` or `"expanding"`, the volatile market has outgrown its fuel supply. Fresh fiat inflows are required for continuation. Verdict: *Overextended / Exit Narrowing — caution on new long entries.*

- **Capital Flight (Macro Exodus):** If `stable_power_ratio < 0.07` AND `supply_trend_7d` is `"contracting"`, this is a structurally different and more severe condition. Net redemptions mean capital is leaving crypto, not rotating. Verdict: *Macro Risk-Off — consider full defensive posture, not just caution.*

- **The "DeFi Inflation" Risk:** `confidence: "high"` means the primary and validator sources agree on supply totals. It does not mean the dry powder is truly idle — a high ratio during a peak DeFi yield season may be partially composed of stablecoins locked in liquidity pools, which are not freely deployable. The LLM should cross-reference DeFi TVL context if available.

- **Cross-metric use:** `stablecoin-power` should be read alongside `liquidity-pulse`. High stablecoin power + high liquidity pulse = capital-rich, actively-trading market (strong bull signal). High stablecoin power + low liquidity pulse = dry powder exists but conviction is absent (accumulation phase, not yet actionable). Low stablecoin power + high liquidity pulse = market is trading heavily on depleted fuel (blow-off or panic risk).