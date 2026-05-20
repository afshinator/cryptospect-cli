package v1

import (
	"fmt"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
)

// Compute performs the pure China M2 classification.
// It is side-effect-free: no I/O, no state mutation.
func Compute(in Input) (Data, computedMeta, error) {
	if len(in.Observations) == 0 {
		return Data{}, computedMeta{}, fmt.Errorf("no observations provided")
	}

	latest := in.Observations[0]
	// Convert from 100 million yuan to CNY billion (divide by 10)
	m2Level := latest.Value / 10.0

	meta := computedMeta{
		Confidence: ConfidenceHigh,
	}

	// Compute YoY if we have enough history
	var yoyChange *float64
	var label string
	var desc string
	insufficientHistory := false

	if len(in.Observations) >= MinHistoryMonths {
		yoy := (latest.Value - in.Observations[12].Value) / in.Observations[12].Value * 100
		yoyChange = &yoy
		label = classifyYoY(yoy)
		desc = classificationDescription(label)
	} else {
		insufficientHistory = true
		label = LabelNormal
		desc = "Insufficient history for YoY calculation — showing latest level only"
		meta.Confidence = ConfidenceMedium
	}

	// Data lag: days since the period
	dataLagDays := 0
	if len(latest.Period) >= 7 {
		// Period format: "2026-02"
		t, err := time.Parse("2006-01", latest.Period)
		if err == nil {
			// End of month
			endOfMonth := t.AddDate(0, 1, -1)
			dataLagDays = int(time.Since(endOfMonth).Hours() / 24)
			if dataLagDays < 0 {
				dataLagDays = 0
			}
		}
	}
	meta.DataLagDays = dataLagDays

	classification := Classification{
		Label:       label,
		Description: desc,
	}

	summary := buildSummary(m2Level, latest.Period, yoyChange, label, insufficientHistory)

	var yoyMetric *metrics.MetricFloat
	if yoyChange != nil {
		y := metrics.Ratio(*yoyChange)
		yoyMetric = &y
	}

	data := Data{
		M2LevelCNYBillion: metrics.Currency(m2Level),
		YoYChangePct:      yoyMetric,
		Period:            latest.Period,
		Classification:    classification,
		Summary:           summary,
	}

	return data, meta, nil
}

// classifyYoY returns the classification label for a YoY change.
func classifyYoY(yoy float64) string {
	switch {
	case yoy > YoYExpandingMin:
		return LabelExpanding
	case yoy >= YoYSlowingMax:
		return LabelNormal
	default:
		return LabelSlowing
	}
}

// classificationDescription returns a human-readable description.
func classificationDescription(label string) string {
	switch label {
	case LabelExpanding:
		return "China M2 expanding — strong liquidity tailwind"
	case LabelNormal:
		return "China M2 growing at normal pace"
	case LabelSlowing:
		return "China M2 growth slowing — potential tightening"
	default:
		return label
	}
}

// buildSummary constructs a one-line natural-language summary.
func buildSummary(level float64, period string, yoyChange *float64, label string, insufficientHistory bool) string {
	if insufficientHistory {
		return fmt.Sprintf("China M2: %.0fB CNY (%s) [INSUFFICIENT_HISTORY — YoY unavailable, need 13+ months].",
			level, period)
	}

	yoyStr := ""
	if yoyChange != nil {
		yoyStr = fmt.Sprintf(", %+.1f%% YoY", *yoyChange)
	}

	verdict := ""
	switch label {
	case LabelExpanding:
		verdict = " — expanding, strong liquidity tailwind."
	case LabelSlowing:
		verdict = " — slowing, potential tightening."
	default:
		verdict = " — normal expansion."
	}

	return fmt.Sprintf("China M2: %.0fB CNY (%s)%s%s", level, period, yoyStr, verdict)
}
