package metrics

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
