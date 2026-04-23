# {metric-name}

**Alias:** `{alias}`  
**Endpoint:** `{endpoint-key}`

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
|-----------|-----------|--------|
| `{Label}` | `{>= or <= value}` | `{explanation}` |

## Data Source

- **API:** `{provider}`
- **Endpoint:** `{endpoint-key}`
- **Fields used:** `{field1}`, `{field2}`

## Output Schema

```json
{
  "data": {
    "{field_name}": "{type}",
    "{classification_field}": "{type}"
  },
  "meta": {
    "sources": [...],
    "thresholds": {...}
  }
}
```

## Usage

```bash
# Basic
cryptospect-cli {metric-name}

# With detail
cryptospect-cli {metric-name} --detail full
```

## Long Description

Comprehensive explanation of what this metric measures, why it matters for market analysis, and how it should be interpreted in different market conditions. This is the authoritative source from which all other sections are derived.