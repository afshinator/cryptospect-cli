package metrics

// ConfidenceToFloat maps a confidence label string to the float used by DetectStatus.
// "high" → 0.9, "medium" → 0.6, "low" → 0.3, anything else → 0.0.
func ConfidenceToFloat(conf string) float64 {
	switch conf {
	case "high":
		return 0.9
	case "medium":
		return 0.6
	case "low":
		return 0.3
	default:
		return 0.0
	}
}

// FloatToConfidence is the inverse of ConfidenceToFloat.
// [0.8, ∞) → "high", [0.5, 0.8) → "medium", (-∞, 0.5) → "low".
func FloatToConfidence(f float64) string {
	switch {
	case f >= 0.8:
		return "high"
	case f >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

// DetectStatus maps confidence and thin-data flag to a metric status string.
// The mapping follows Design‑Decisions.md Step 12.8:
//   - confidence ≥ 0.8 → "ok"
//   - confidence ≥ 0.5 → "degraded"
//   - otherwise → "unavailable"
//
// If thinData is true, the status is downgraded by one level
// ("ok" → "degraded", "degraded" → "unavailable", "unavailable" stays "unavailable").
func DetectStatus(confidence float64, thinData bool) string {
	var status string
	switch {
	case confidence >= 0.8:
		status = "ok"
	case confidence >= 0.5:
		status = "degraded"
	default:
		status = "unavailable"
	}

	if !thinData {
		return status
	}

	// Downgrade by one level
	switch status {
	case "ok":
		return "degraded"
	case "degraded":
		return "unavailable"
	default:
		return "unavailable" // already unavailable
	}
}
