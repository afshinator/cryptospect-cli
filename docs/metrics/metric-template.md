# {metric-name}

**version:** `v{x.y.z}`
**Alias:** `{alias}`  
**Endpoints:** `{endpoint-key(s)}`

## Overview

Brief summary of what this metric measures (derived from Long Description).

## Formula (or how to compute)

```
{mathematical formula}
```

## Interpretation

What different values mean for trading decisions. This section provides context for how agents should use the metric in trading logic.

## Classification

| Condition | Threshold |
|-----------|-----------|
| `{Label}` | `{>= or <= value}` |


## Data Source(s)

- **Primary API:** `{provider}`
- **Endpoint:** `{endpoint-key}`
- **Fields used:** `{field1}`, `{field2}`
- **Other API's:** `{provider}` ...


## Cross-Source Verification

This metric uses the **Primary with Anchor-Asset Validation** pattern.

| Role | Source | Purpose |
|------|--------|---------|
| Primary | `{primary-provider}` | Main metric computation |
| Validator | `{validator-provider}` | Anchor-asset volume check |

**Anchor asset:** (if applicable) `{BTC or other}`

**Discrepancy threshold:** (if applicable) `{default: 20}%`

**Behavior on mismatch:** (if applicable) `{description of what happens}`

If no cross-source verification: "No cross-source verification in v1."

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `{flag-name}` | `{bool/string}` | `{default}` | `{description}` |

If the metric has no CLI flags: "This metric has no CLI flags."


## Output Schema

```json
{
    "metric":  "string",      // canonical metric name
    "version": "v{x.y.z}",   // SemVer with "v" prefix
    "status":  "string",      // "ok" / "degraded" / "unavailable"

    "data": {
        "{field}": "{type}",
        "classification": {
            "label":       "string",  // categorical label, e.g. "high" / "normal" / "low"
            "description": "string"   // threshold explanation, e.g. "Strong short-term conviction"
        },
        "summary": "string"           // NL sentence combining key fields + classification
    },

    "meta": {
        // Omitted when --detail basic.
        // Present when --detail extended or full:
        "cache_hit":          "bool",
        "ttl_remaining_sec":  "int",
        "primary_source":     "string",  // e.g. "coingecko"
        "validator_source":   "string",  // present if cross-source validation applies
        "discrepancy_detected": "bool",  // present if cross-source validation applies
        "discrepancy_note":   "string",  // only if discrepancy_detected == true
        "confidence":         "string"   // "high" / "medium" / "low"
        // Additionally when --detail full:
        // "thresholds": { ... }
        // "description": "string"
    }
}
```

**Enhancements** (conditional — present when specific conditions are met):

| Field | Condition | Description |
|-------|-----------|-------------|
| `{field}` | `{when it appears}` | `{explanation}` |

## Usage

```bash
# Basic
cryptospect-cli {metric-name}

# With alias
cryptospect-cli {alias}

# With detail
cryptospect-cli {metric-name} --detail full

# With flag (if applicable)
cryptospect-cli {metric-name} --{flag-name}
```

## Long Description

### High-level meaning and value
- What metric measures
- What question(s) it answers

### Exact definition and data needs, logic
- The formula/calculation logic
- Input data requirements

### Possible values and associated verdicts
- Value ranges and what each means

### Other details

**CLI Flags:** (if applicable) — explanation of each flag and when to use it

**Enhancements:** (if applicable) — explanation of conditional output fields

**Cross-Source Verification:** (if applicable) — what validation does, how discrepancy affects output

**Implementation Compromises:** (if applicable) — known limitations or simplifications

**Future Enhancements:** (if applicable) — planned but not yet implemented

**Agentic Logic (strategic notes)** (if applicable) - What an LLM or agent calls this tool, what should it look for.