# {metric-name}

**Alias:** `{alias}`  
**Endpoints:** `{endpoint-key(s)}`

## Overview

Brief summary of what this metric measures (derived from Long Description).

## Formula

```
{mathematical formula}
```

## Interpretation

What different values mean for trading decisions.

## Classification

| Condition | Threshold | Meaning |
|-----------|-----------|---------|
| `{Label}` | `{>= or <= value}` | `{explanation}` |

## Data Source

- **Primary API:** `{provider}`
- **Endpoint:** `{endpoint-key}`
- **Fields used:** `{field1}`, `{field2}`

## Cross-Source Verification

This metric uses the **Primary with Anchor-Asset Validation** pattern.

| Role | Source | Purpose |
|------|--------|---------|
| Primary | `{primary-provider}` | Main metric computation |
| Validator | `{validator-provider}` | Anchor-asset volume check |

**Anchor asset:** `{BTC or other}`

**Discrepancy threshold:** `{default: 20}%`

**Behavior on mismatch:** `{description of what happens}`

If no cross-source verification: "No cross-source verification in v1."

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `{flag-name}` | `{bool/string}` | `{default}` | `{description}` |

If the metric has no CLI flags: "This metric has no CLI flags."

## Output Schema

**Base fields** (always present):
```json
{
  "data": {
    "{field}": "{type}",
    "classification": "string",
    "summary": "string"
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