# Daily Brief — cryptospect-cli

A canonical morning checkup sequence. Run this with no prior trigger, no market observation, no thesis. It answers: **what is the market doing right now?**

For scenario-driven chains (diagnosing a rally, reading a selloff), see [examples.md](examples.md). For signal pattern recognition and synthesis rules, see [agents.md](agents.md).

---

## The Sequence

Four commands. Run them in this order. The cache is shared across calls, so all four typically trigger only 2–3 actual API requests.

```bash
cryptospect-cli market-regime  --detail extended
cryptospect-cli stablecoin-power --detail extended
cryptospect-cli flow-tension   --detail extended
cryptospect-cli fear-greed-index
```

**Why this order:**
- `mr` first — establishes the structural regime label before any other signal is read. Everything else is colored by it.
- `sp` second — answers whether capital is available to move. A bullish regime with no fuel is a different answer than one with abundant dry powder.
- `ft` third — answers whether that capital is *actually moving*, and in which direction. Kinetics on top of structural context.
- `fgi` last — sentiment overlay. Confirms or contradicts the mechanical picture. Extreme fear during a structurally bullish regime = opportunity. Extreme greed during a weak regime = caution.

Use `--detail extended` (not `basic`) for `mr`, `sp`, and `ft` — it exposes `confidence`, `notes`, and the cold-start flags you need to reason correctly with partial data.

---

## Reading the Output

### Step 1 — Regime label sets the frame

From `market-regime`, read `data.regime` first. Ten possible labels:

| Label | What it means in one sentence |
|-------|-------------------------------|
| `BTC-Led Expansion` | BTC dominance rising, broad participation — new money coming in behind BTC |
| `Institutional Build` | BTC dominance rising, mixed breadth — institutions accumulating, alts not yet following |
| `Flight to Safety` | BTC dominance rising, weak breadth — market-wide fear, capital hiding in BTC |
| `Steady Appreciation` | BTC dominance neutral, broad participation — healthy alt-inclusive bull |
| `Consolidation` | BTC dominance neutral, mixed breadth — market seeking direction |
| `Stagnation` | BTC dominance neutral, weak breadth — low conviction, nothing moving |
| `Alt-Season / Mania` | BTC dominance falling, strong breadth — full risk-on rotation |
| `Capital Rotation` | BTC dominance falling, mixed breadth — alts absorbing flows, uneven participation |
| `Structural Decay` | BTC dominance falling, weak breadth — broad selloff, no safe sector |
| `Capitulation` | All metrics collapsing + high conviction — maximum pain, potential floor |

Then read `data.modifier`: `positive_momentum`, `negative_momentum`, or `neutral`. This is BTC's 24h price direction — it tells you whether the regime is under pressure or confirming.

Then check `meta.confidence` and `meta.notes`. If you see `"cold_start"` in notes, the dominance trend is defaulting to neutral — the regime label is valid but the dominance direction component is uninformed. Note this and proceed.

### Step 2 — Fuel check shapes the conclusion

From `stablecoin-power`, two fields together tell the full story:

| `stable_power_ratio` | `supply_trend_7d` | Reading |
|----------------------|-------------------|---------|
| > 0.15 | any | Abundant dry powder — sustained moves are mechanically possible |
| 0.07–0.15 | `expanding` or `stable` | Healthy balance — adequate fuel |
| 0.07–0.15 | `contracting` | Normal ratio but supply shrinking — watch for transition to Low |
| < 0.07 | `stable` or `expanding` | Overextended — volatile market outgrew fuel; rally ran on fumes |
| < 0.07 | `contracting` | **Capital Flight** — money leaving crypto entirely; most severe scenario |

Never report a Low `stable_power_ratio` without `supply_trend_7d`. The two scenarios have opposite implications.

### Step 3 — Flow tension reads the kinetics

From `flow-tension`, read the three hooks together, not individually:

| CVD hook | OI hook | Funding hook | What it means |
|----------|---------|--------------|---------------|
| `aggressive_buy` | `building` | `neutral` or `positive` | Strong early bull — spot demand + growing leverage, not yet crowded |
| `aggressive_buy` | `stable` | `neutral` | Genuine spot demand, no leverage amplification — structurally clean |
| `aggressive_buy` | `building` | `overheated` | **Long Squeeze Risk** — crowded longs, fragile |
| `neutral` | `building` | any | **Building Tension** — volatility coil, direction unclear |
| `aggressive_sell` | `unwinding` | `negative` or `neutral` | **Deleveraging** — flush in progress |
| `aggressive_sell` | `stable` | any | Spot distribution — holders selling, no leverage involved, more persistent |
| any | `stable` | `negative` | Shorts paying longs — bearish sentiment dominant; watch CVD for exhaustion |

If `open_interest.change_pct_24h` is absent from the output, the OI hook is a cold-start default (`"stable"`) — treat it as no signal, not as a real stable reading.

If the metric `status` is `"degraded"`, only CVD is available (CoinGecko transient failure). CVD alone is insufficient for a directional conclusion — use it to cross-check `mr` and `mb` only.

### Step 4 — Sentiment as a contrarian overlay

From `fear-greed-index`, read `data.value` (0–100) and `data.classification.label`:

| Value range | Label | Contrarian read |
|-------------|-------|-----------------|
| 0–25 | `extreme_fear` | Historically: accumulation opportunity *if* `mr` and `sp` are structurally sound |
| 26–45 | `fear` | Caution, not panic — watch `ft` for capitulation signals |
| 46–55 | `neutral` | No sentiment edge — mechanical signals dominate |
| 56–75 | `greed` | Elevated, not extreme — normal bull market state |
| 76–100 | `extreme_greed` | Historically: reduce exposure *if* `ft` funding is `overheated` |

`fgi` is a contrarian overlay, not a directional signal. Extreme fear during a structurally healthy regime (`mr` Steady Appreciation or BTC-Led Expansion + `sp` High) has historically been a buying opportunity. Extreme greed during a structurally weak regime is a warning. The combination matters, not either value in isolation.

---

## Assembling the Brief

Once you have all four outputs, combine them into a single assessment using this structure:

**Regime** → what structural phase is the market in?  
**Fuel** → does the structure have the capital supply to sustain a move?  
**Kinetics** → is that capital actually in motion, and which direction?  
**Sentiment** → does crowd positioning confirm or contradict the mechanical picture?

Where signals agree, state the combined verdict confidently and name the pattern if it matches one from agents.md. Where signals conflict, name the conflict explicitly — do not average it away. A bullish regime with bearish kinetics is not "mixed" — it is a specific condition (structural bull with spot distribution) that carries a specific risk profile.

**LLM prompt for the daily brief:**

> "Here is JSON output from four cryptospect-cli commands run in sequence: market-regime, stablecoin-power, flow-tension, and fear-greed-index. Using the signal hierarchy from agents.md, produce a daily market brief covering: (1) current structural regime and what it means, (2) fuel availability and its direction of travel, (3) whether capital is actually moving and in what direction, (4) how sentiment positions against the mechanical picture. End with a one-sentence recommended posture and identify any Named Signal Combination patterns present."

---

## When Data Is Degraded

The daily brief sequence will occasionally return `status: "degraded"` or `status: "unavailable"` on one or more metrics. This is normal — not a reason to discard the run. Here is how to reason with partial data.

### Example: `china-m2` returning degraded

`china-m2` is a monthly-frequency macro metric. Its data often carries a significant lag — the underlying NBS series updates monthly, and the most recent datapoint may be 30–107 days old. A real output:

```json
{
  "metric": "china-m2",
  "status": "degraded",
  "data": {
    "yoy_change_pct": 7.9,
    "classification": { "label": "normal" },
    "summary": "China M2 YoY growth 7.9% — normal monetary expansion."
  },
  "meta": {
    "confidence": "medium",
    "data_lag_days": 107,
    "primary_source": "dbnomics"
  }
}
```

**How to read this:** The YoY figure is real and the classification is valid — China M2 is in normal expansion at 7.9%. The `degraded` status and 107-day lag mean you're reading the state of monetary policy as of roughly three months ago, not today. Use the trend direction (expanding/normal/slowing) as a macro backdrop, not a real-time signal. Do not use it as a trigger for a trade. Do use it to contextualize the `mr` regime — a `normal` China M2 backdrop during a `Flight to Safety` regime is less alarming than a `slowing` one.

### Example: `flow-tension` returning degraded (OI/funding unavailable)

```json
{
  "metric": "flow-tension",
  "status": "degraded",
  "data": {
    "signals": {
      "cvd": { "ratio": 0.31, "hook": "aggressive_buy" },
      "open_interest": { "current_usd": null, "hook": "stable" },
      "funding_rate": { "rate": null, "hook": "neutral" }
    },
    "summary": "CVD aggressive buy; OI and funding unavailable (CoinGecko transient failure)."
  },
  "meta": { "confidence": "medium" }
}
```

**How to read this:** CVD (`aggressive_buy`) is real — Binance-US returned cleanly. OI and funding are null due to a CoinGecko transient failure; their hooks (`stable`, `neutral`) are defaults, not signals. Do not act on those hooks. Use CVD to confirm or cross-check `mr` and `mb` directional reads. For the OI and funding picture, either retry in a few minutes or accept reduced confidence on the kinetics layer for this run.

### General degraded-data rules

| Condition | What to do |
|-----------|-----------|
| `status: "unavailable"` | Skip this metric's data entirely. Note the gap in your brief. Do not fill the gap with assumptions. |
| `status: "degraded"` + `confidence: "medium"` | Use the available fields. Explicitly note which fields are absent or stale when drawing conclusions. |
| `status: "degraded"` + a cold-start note | The absent field has a defaulted value (e.g., OI `"stable"`, dominance trend `"neutral"`). Treat defaulted values as no-signal, not as real readings. |
| One metric unavailable, others ok | Proceed with the remaining metrics. Name the gap in your conclusion. Do not block the whole brief on one failed source. |
| All primary metrics (`mr`, `sp`, `ft`) degraded simultaneously | Likely a CoinGecko rate limit or outage. Run `cryptospect-cli cache-clear`, wait 60 seconds, retry. |

The brief is useful even when one or two metrics are degraded. A structural regime read from `mr` plus a sentiment read from `fgi` is a partial picture — name it as partial, not as a failure.

---

## Optional Additions

After the core four, add these based on what the regime suggests:

| If `mr` shows... | Also run | Why |
|-----------------|----------|-----|
| Any regime + want rotation depth | `md` | Where is capital flowing across large/mid/small cap tiers? |
| `Flight to Safety` or `Structural Decay` | `mb` | Confirm Ghost Rally or broad selloff with participation data |
| `Alt-Season` or `Capital Rotation` | `mb`, `md` | Verify rotation is broad and extending into mid/small caps |
| Any regime + macro uncertainty | `cnm2` | China monetary backdrop (use as trend context, not real-time signal) |
| Any regime + want volatility context | `vol` | BTC/ETH realized vol and spread — is the market calm or turbulent? |

Running all ten every morning is unnecessary and wasteful. The core four cover the primary axes. Add selectively based on what the regime label suggests.