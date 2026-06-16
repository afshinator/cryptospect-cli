# Examples — Reading the Market, Not Just the Numbers

Each metric in `cryptospect-cli` answers one question. The real signal comes from *combining* them — following a thread from macro context down to specific conditions. The examples below walk through three common lines of inquiry and show you exactly how to chain the commands.

---

## Example 1: "Is this rally real, or is it a BTC trap?"

You've noticed the total market cap is up 4% today. Before chasing it, you want to know: is this a genuine broad move, or is it BTC carrying an otherwise soft market?

**Step 1 — Establish the macro regime**

```bash
cryptospect-cli market-regime --detail full
```

This is always your first call. It tells you the structural state in a single label — `Alt-Season`, `Institutional Build`, `Flight to Safety`, etc. — and anchors everything else you're about to read.

Suppose the output shows:
```json
"regime": "Institutional Build",
"dominance_trend": "rising",
"market_breadth_score": 0.44
```

That's a yellow flag: BTC dominance is rising, and breadth is mixed (44% of coins are green). BTC may be outperforming while alts lag. Let's check.

**Step 2 — Verify participation**

```bash
cryptospect-cli market-breadth
```

`market-breadth` gives you the weighted composite of how many coins are green across 1h, 24h, 7d, and 30d windows, plus an explicit `divergence_detected` flag.

If you see:
```json
"market_breadth_score": 0.41,
"divergence_detected": true,
"btc_change_24h_pct": 4.2
```

That's a **Ghost Rally**: BTC is up 4.2% but fewer than half the market is participating. The rally is mechanically narrow.

**Step 3 — Check if the aggression is real**

```bash
cryptospect-cli flow-tension
```

`flow-tension` tells you *how* the move is happening: is it backed by aggressive spot buying (CVD), or are people just piling into leveraged longs?

If CVD is `aggressive_buy` but funding is `overheated` (`> 0.30% per 8h`), you've found the trap: spot demand exists but leveraged longs are crowded and paying dearly to stay open. The move is fragile.

**LLM prompt to tie it together:**

> "I ran three cryptospect-cli commands. Here's the JSON output from `market-regime`, `market-breadth`, and `flow-tension`. The regime is Institutional Build, breadth is 0.41 with divergence detected, and flow-tension shows aggressive buy CVD but overheated funding. Is this rally safe to chase, or is this a Ghost Rally setup?"

---

## Example 2: "Is there enough fuel left for the run to continue?"

The market has been in a solid uptrend for three weeks. You want to know if there's still dry powder available, or if the rally has already consumed its fuel supply.

**Step 1 — Check dry powder**

```bash
cryptospect-cli stablecoin-power --detail extended
```

`stablecoin-power` is your fuel gauge: the ratio of stablecoin market cap to volatile-asset market cap. A high ratio means sidelined capital hasn't deployed yet. A low ratio means the tank is nearly empty.

Look at two fields together:
- `stable_power_ratio` — the ratio itself (High >0.15, Normal 0.07–0.15, Low <0.07)
- `supply_trend_7d` — is stablecoin supply growing, stable, or shrinking?

If `stable_power_ratio` is `0.06` and `supply_trend_7d` is `contracting`, that's not just "fuel depleted" — it's **capital flight**. Money is leaving crypto entirely, not just rotating into volatile assets.

**Step 2 — Check how the conviction is being expressed**

```bash
cryptospect-cli liquidity-pulse
```

Low stablecoin power becomes more or less urgent depending on how actively the market is trading. A `volume_to_mcap_ratio` above `0.15` alongside low fuel means the market is burning hot on near-empty — a blow-off risk. A low ratio with low fuel means the market is coasting quietly; less immediate danger.

**Step 3 — See where remaining capital is rotating**

```bash
cryptospect-cli momentum-divergence
```

Even when aggregate fuel is low, knowing *which tier* is receiving flows matters. `momentum-divergence` segments the top 200 coins into Large (top 10), Mid (11–50), and Small (51–200) and reports whether capital is moving down the risk curve or concentrating into mega-caps.

A `risk_on` label with `tail_extension: true` means capital is rotating aggressively into smaller assets — historically this coincides with late-cycle speculative peaks. Pair that with `stablecoin-power` showing Low + contracting supply, and you have a classic blow-off setup.

**LLM prompt to tie it together:**

> "Here's JSON output from `stablecoin-power`, `liquidity-pulse`, and `momentum-divergence`. Stablecoin power is 0.06 with contracting supply, liquidity pulse is 0.18 (high), and momentum-divergence shows risk_on with tail_extension true. What does this combination tell me about where we are in the cycle, and what's the appropriate posture?"

---

## Example 3: "The market just sold off hard. Is this a flush or the beginning of something worse?"

You woke up to a 12% daily drop. You want to know: is this a leveraged unwind that will stabilize, or is there a macro story underneath it?

**Step 1 — Understand the structural state first**

```bash
cryptospect-cli market-regime
```

The regime label is your first read. `Capitulation` requires high trading conviction alongside collapsing breadth — it's different from `Structural Decay`, which is a slow bleed. `Flight to Safety` means capital is concentrating into BTC rather than leaving crypto. These require different responses.

**Step 2 — Diagnose the mechanics**

```bash
cryptospect-cli flow-tension --detail extended
```

Selloffs have different causes. `flow-tension` separates them:

- **Leveraged unwind:** OI `unwinding` (>-5%) + CVD `aggressive_sell` → forced liquidations clearing the deck. Often self-limiting once OI is flushed.
- **Spot distribution:** CVD `aggressive_sell` + OI `stable` → holders selling into bids. More persistent, no forced-liquidation floor.
- **Funding normalization:** Funding `negative` (shorts paying longs) → bearish sentiment dominant. If CVD is turning neutral or positive alongside this, exhaustion may be near.

**Step 3 — Check whether the macro backdrop amplifies the risk**

```bash
cryptospect-cli stablecoin-power
```

If `supply_trend_7d` is `contracting` during a selloff, capital is being redeemed back to fiat — not rotating into stablecoins for re-entry. That's a fundamentally different situation from a normal correction where stablecoin supply grows as people take risk off.

Also check:
```bash
cryptospect-cli china-m2
```

A China M2 `slowing` classification during a crypto selloff adds macro weight. China's M2 (~$47T, roughly 2× the US) has historically shown strong correlation to BTC price cycles. Tightening monetary conditions there amplify crypto drawdowns.

**LLM prompt to tie it together:**

> "Crypto just dropped 12% today. Here's the JSON output from `market-regime`, `flow-tension`, `stablecoin-power`, and `china-m2`. Regime is Flight to Safety, flow-tension shows OI unwinding with aggressive sell CVD, stablecoin supply is contracting, and China M2 is slowing. Is this a leverage flush I should buy, or something more structural?"

---

## Quick Reference: Which Metrics Talk to Each Other

| If you're asking... | Start here | Then check |
|---------------------|-----------|------------|
| Is this rally real? | `market-regime` | `market-breadth` → `flow-tension` |
| Is there fuel for continuation? | `stablecoin-power` | `liquidity-pulse` → `momentum-divergence` |
| Is this selloff a flush or flight? | `flow-tension` | `stablecoin-power` → `china-m2` |
| Where is capital rotating? | `momentum-divergence` | `market-breadth` → `flow-tension` |
| What's the overall sentiment backdrop? | `fear-greed-index` | `market-regime` |

`market-regime` aggregates breadth and dominance internally and is always a good first call. Every other metric adds a layer of *why*.

---

## Running All Relevant Signals at Once

For a full macro snapshot, feed all outputs to an LLM in one shot:

```bash
cryptospect-cli market-regime --detail full > mr.json
cryptospect-cli stablecoin-power --detail full >> mr.json
cryptospect-cli flow-tension --detail full >> mr.json
cryptospect-cli market-breadth --detail full >> mr.json
```

Then prompt:

> "Here is JSON output from four cryptospect-cli metrics: market-regime, stablecoin-power, flow-tension, and market-breadth. Synthesize these into a single market assessment. What regime are we in, how much fuel remains, and is the current price action conviction-backed or fragile? End with a one-sentence recommended posture."

Use `--detail full` when feeding to an LLM — it includes the threshold values and metric descriptions the model needs to interpret the numbers correctly. Use `--detail basic` for automated loops where token economy matters.