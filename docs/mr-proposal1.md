## Reasoning: Market Regime (mr) Design Strategy

To reconcile the **Market Regime (mr)** metric with your architecture, I evaluated two implementation paths: the **Independent Fetcher** (fetching raw data) versus the **Pure Aggregator** (consuming other metrics).

### Step-by-Step Logic

1. **Dependency Check:** The proposed "Vol-Price Matrix" requires **Bitcoin Dominance (BTC.D)** and **Market Breadth**. While Breadth is an existing metric (`mb`), BTC Dominance is a global macro statistic typically found in the `coingecko.global_market` endpoint.
2. **Architectural Alignment:** Your `cryptospect-cli` is designed as a suite of individual tools for agents to compose. However, the "Market Regime" is explicitly defined in your summary as the "Master State" meant to establish macro context first.
3. **The Hybrid Approach:** To maintain the "single static binary" and "agent-optimized" philosophy, `mr` should act as a **higher-order logic layer**. It will fetch the global macro data (BTC.D) directly but *must* accept Breadth as an input or internal dependency to ensure the "Matrix" is mathematically consistent with the `mb` command.
4. **Addressing Ambiguities:** The initial prompt mentions "Volume" in the title ("Vol-Price Matrix") but the logic focuses on "Dominance." I have integrated **Liquidity Pulse (lp)** as the volume/conviction component to satisfy the "Vol" requirement and distinguish between "Early" and "Late" phases.

---

## Market Regime (mr) — v1.0.0 (Draft)

**Alias:** `mr`

**Endpoints:** `coingecko.global_market`, `coingecko.coin_markets_breadth`

## Overview

The "Master State" indicator. It classifies the market's structural cycle by cross-referencing **Bitcoin Dominance (BTC.D)** against **Market Breadth**. It identifies whether capital is concentrating in "Safety" (BTC), rotating into "Risk" (Alts), or fleeing the ecosystem entirely.

## Formula

The metric uses a **$2 \times 2$ Matrix** (Dominance vs. Breadth) with a **Liquidity** overlay to determine the cycle phase.

### 1. The Core Matrix (Regime)

| BTC Dominance | Breadth Score ($>0.50$) | Breadth Score ($<0.50$) |
| --- | --- | --- |
| **Rising** ($\Delta 24h > 0$) | **Institutional Build** | **Flight to Safety** |
| **Falling** ($\Delta 24h < 0$) | **Alt-Season / Expansion** | **Capitulation / Decay** |

### 2. The Phase Overlay (Maturity)

Using the `liquidity-pulse` (Volume/Mcap) ratio:

* **Early:** High Volume + Improving Breadth (Breakout).
* **Late:** Declining Volume + Narrowing Breadth (Exhaustion).

## Interpretation

* **Institutional Build:** BTC leads the market higher. Historically the "safest" entry for a new bull cycle.
* **Alt-Season:** High-risk appetite. Capital is rotating down the risk curve. High gains, but watch for "Late" phase exhaustion.
* **Flight to Safety:** Market-wide fear. Only BTC is holding value; alts are being liquidated.
* **Capitulation:** Maximum pain. Total market exit. Historically provides the best long-term "Accumulation" opportunities.

## Data Source(s)

* **Primary API:** CoinGecko
* **Endpoints:** * `/global` (for `market_cap_percentage["btc"]`)
* `/coins/markets` (shared with `mb` to pull Breadth)



## Output Schema (Draft)

```json
{
    "metric": "market-regime",
    "version": "v1.0.0",
    "status": "ok",
    "data": {
        "regime": "string",      // e.g., "Institutional Build"
        "phase": "string",       // e.g., "Early" / "Mid" / "Late"
        "btc_dominance": 52.4,
        "breadth_snapshot": 0.38,
        "classification": {
            "label": "flight_to_safety",
            "description": "BTC rising dominance with narrow participation."
        },
        "summary": "Market is in a Flight to Safety regime; capital is hiding in BTC while alts bleed."
    }
}

```

---

## Technical Reconciliation & Ambiguities

### 1. The "Volume" Ambiguity

The title provided was "Vol-Price Matrix," but the description focused on "Dominance-Breadth."

* **Resolution:** I have included **Liquidity Pulse (lp)*## Reasoning: Market Regime (mr) Design Strategy

To reconcile the **Market Regime (mr)** metric with your architecture, I evaluated two implementation paths: the **Independent Fetcher** (fetching raw data) versus the **Pure Aggregator** (consuming other metrics).

### Step-by-Step Logic

1. **Dependency Check:** The proposed "Vol-Price Matrix" requires **Bitcoin Dominance (BTC.D)** and **Market Breadth**. While Breadth is an existing metric (`mb`), BTC Dominance is a global macro statistic typically found in the `coingecko.global_market` endpoint.
2. **Architectural Alignment:** Your `cryptospect-cli` is designed as a suite of individual tools for agents to compose. However, the "Market Regime" is explicitly defined in your summary as the "Master State" meant to establish macro context first.
3. **The Hybrid Approach:** To maintain the "single static binary" and "agent-optimized" philosophy, `mr` should act as a **higher-order logic layer**. It will fetch the global macro data (BTC.D) directly but *must* accept Breadth as an input or internal dependency to ensure the "Matrix" is mathematically consistent with the `mb` command.
4. **Addressing Ambiguities:** The initial prompt mentions "Volume" in the title ("Vol-Price Matrix") but the logic focuses on "Dominance." I have integrated **Liquidity Pulse (lp)** as the volume/conviction component to satisfy the "Vol" requirement and distinguish between "Early" and "Late" phases.

---

## Market Regime (mr) — v1.0.0 (Draft)

**Alias:** `mr`

**Endpoints:** `coingecko.global_market`, `coingecko.coin_markets_breadth`

## Overview

The "Master State" indicator. It classifies the market's structural cycle by cross-referencing **Bitcoin Dominance (BTC.D)** against **Market Breadth**. It identifies whether capital is concentrating in "Safety" (BTC), rotating into "Risk" (Alts), or fleeing the ecosystem entirely.

## Formula

The metric uses a **$2 \times 2$ Matrix** (Dominance vs. Breadth) with a **Liquidity** overlay to determine the cycle phase.

### 1. The Core Matrix (Regime)

| BTC Dominance | Breadth Score ($>0.50$) | Breadth Score ($<0.50$) |
| --- | --- | --- |
| **Rising** ($\Delta 24h > 0$) | **Institutional Build** | **Flight to Safety** |
| **Falling** ($\Delta 24h < 0$) | **Alt-Season / Expansion** | **Capitulation / Decay** |

### 2. The Phase Overlay (Maturity)

Using the `liquidity-pulse` (Volume/Mcap) ratio:

* **Early:** High Volume + Improving Breadth (Breakout).
* **Late:** Declining Volume + Narrowing Breadth (Exhaustion).

## Interpretation

* **Institutional Build:** BTC leads the market higher. Historically the "safest" entry for a new bull cycle.
* **Alt-Season:** High-risk appetite. Capital is rotating down the risk curve. High gains, but watch for "Late" phase exhaustion.
* **Flight to Safety:** Market-wide fear. Only BTC is holding value; alts are being liquidated.
* **Capitulation:** Maximum pain. Total market exit. Historically provides the best long-term "Accumulation" opportunities.

## Data Source(s)

* **Primary API:** CoinGecko
* **Endpoints:** * `/global` (for `market_cap_percentage["btc"]`)
* `/coins/markets` (shared with `mb` to pull Breadth)



## Output Schema (Draft)

```json
{
    "metric": "market-regime",
    "version": "v1.0.0",
    "status": "ok",
    "data": {
        "regime": "string",      // e.g., "Institutional Build"
        "phase": "string",       // e.g., "Early" / "Mid" / "Late"
        "btc_dominance": 52.4,
        "breadth_snapshot": 0.38,
        "classification": {
            "label": "flight_to_safety",
            "description": "BTC rising dominance with narrow participation."
        },
        "summary": "Market is in a Flight to Safety regime; capital is hiding in BTC while alts bleed."
    }
}

```

---

## Technical Reconciliation & Ambiguities

### 1. The "Volume" Ambiguity

The title provided was "Vol-Price Matrix," but the description focused on "Dominance-Breadth."

* **Resolution:** I have included **Liquidity Pulse (lp)** logic as the "Phase" detector. Without volume, we cannot distinguish between "Healthy Accumulation" (low vol) and "Panic Capitulation" (high vol).

### 2. Dependency Management

* **Ambiguity:** Should `mr` call `mb.Compute()`?
* **Resolution:** Yes. To ensure the CLI remains a "single source of truth," `market-regime` should internalize the logic of `market-breadth`. If a user runs `mb` and `mr` in a script, the Breadth score must be identical.

### 3. "Shelf-Life" Logic

* **Ambiguity:** You mentioned a longer "shelf life" (hourly/daily).
* **Resolution:** The CLI is stateless (single invocation). To respect "shelf life," the `mr` output will include a `recommended_ttl_sec` of 3600 (1 hour), signaling to the calling agent that frequent re-polling is unnecessary for this specific metric.

### 4. Definition of "Rising/Falling" Dominance

* **Ambiguity:** Dominance is a snapshot. How do we know it's "Rising"?
* **Resolution:** The implementation will require `market_cap_percentage_24h_change` from the `/global` endpoint. If unavailable, it will fall back to a "Neutral" trend and flag the confidence as `medium`.

---

**Most likely way this is wrong:** The $2 \times 2$ matrix may be too reductive for "choppy" sideways markets. In a range-bound environment where BTC Dominance fluctuates by only $0.1\%$, the metric might flip-flop between "Institutional Build" and "Expansion" rapidly, creating "noise" for an agent. A "Neutral" dead-zone for Dominance changes (e.g., $\pm 0.5\%$) should be implemented to prevent this.* logic as the "Phase" detector. Without volume, we cannot distinguish between "Healthy Accumulation" (low vol) and "Panic Capitulation" (high vol).

### 2. Dependency Management

* **Ambiguity:** Should `mr` call `mb.Compute()`?
* **Resolution:** Yes. To ensure the CLI remains a "single source of truth," `market-regime` should internalize the logic of `market-breadth`. If a user runs `mb` and `mr` in a script, the Breadth score must be identical.

### 3. "Shelf-Life" Logic

* **Ambiguity:** You mentioned a longer "shelf life" (hourly/daily).
* **Resolution:** The CLI is stateless (single invocation). To respect "shelf life," the `mr` output will include a `recommended_ttl_sec` of 3600 (1 hour), signaling to the calling agent that frequent re-polling is unnecessary for this specific metric.

### 4. Definition of "Rising/Falling" Dominance

* **Ambiguity:** Dominance is a snapshot. How do we know it's "Rising"?
* **Resolution:** The implementation will require `market_cap_percentage_24h_change` from the `/global` endpoint. If unavailable, it will fall back to a "Neutral" trend and flag the confidence as `medium`.

---

**Most likely way this is wrong:** The $2 \times 2$ matrix may be too reductive for "choppy" sideways markets. In a range-bound environment where BTC Dominance fluctuates by only $0.1\%$, the metric might flip-flop between "Institutional Build" and "Expansion" rapidly, creating "noise" for an agent. A "Neutral" dead-zone for Dominance changes (e.g., $\pm 0.5\%$) should be implemented to prevent this.