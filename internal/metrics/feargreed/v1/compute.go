package v1

import (
	"fmt"
)

// Compute performs the pure FGI classification.
// It is side-effect-free: no I/O, no state mutation.
func Compute(in Input) (Data, computedMeta, error) {
	if len(in.Values) == 0 {
		return Data{}, computedMeta{}, fmt.Errorf("no values provided")
	}

	value := in.Values[0]
	label := classifyValue(value)
	desc := classificationDescription(label)

	classification := Classification{
		Label:       label,
		Description: desc,
	}

	meta := computedMeta{
		Confidence: ConfidenceHigh,
	}

	// 7-day MA (requires 7+ data points)
	var sevdMA *float64
	var trend string
	if len(in.Values) >= MAWindow {
		ma := movingAverage(in.Values[:MAWindow])
		sevdMA = &ma
		trend = classifyTrend(float64(value), ma)
		meta.SevdMA = sevdMA
		meta.Trend = trend
	} else {
		meta.Confidence = ConfidenceMedium
	}

	summary := buildSummary(float64(value), label, sevdMA, trend)

	data := Data{
		Value:          value,
		Classification: classification,
		Summary:        summary,
	}

	return data, meta, nil
}

// classifyValue returns the FGI classification label for a raw value.
func classifyValue(value int) string {
	switch {
	case value <= BandExtremeFearMax:
		return LabelExtremeFear
	case value <= BandFearMax:
		return LabelFear
	case value <= BandNeutralMax:
		return LabelNeutral
	case value <= BandGreedMax:
		return LabelGreed
	default:
		return LabelExtremeGreed
	}
}

// classificationDescription returns a human-readable description.
func classificationDescription(label string) string {
	switch label {
	case LabelExtremeFear:
		return "Extreme Fear — historically a buying opportunity"
	case LabelFear:
		return "Fear — cautious sentiment, market may be undervalued"
	case LabelNeutral:
		return "Neutral — balanced sentiment, no contrarian signal"
	case LabelGreed:
		return "Greed — elevated sentiment, market may be overvalued"
	case LabelExtremeGreed:
		return "Extreme Greed — historically a correction risk"
	default:
		return label
	}
}

// classifyTrend returns trend direction: value vs 7-day MA.
func classifyTrend(value, ma float64) string {
	diff := value - ma
	switch {
	case diff > TrendDeadBand:
		return TrendImproving
	case diff < -TrendDeadBand:
		return TrendDeteriorating
	default:
		return TrendStable
	}
}

// movingAverage computes the simple moving average of values.
func movingAverage(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// buildSummary constructs a one-line natural-language summary.
func buildSummary(value float64, label string, sevdMA *float64, trend string) string {
	base := fmt.Sprintf("Fear & Greed: %.0f/100 (%s)", value, classificationTitle(label))

	if sevdMA != nil && trend != "" {
		base += fmt.Sprintf(", sentiment %s (7d MA: %.0f)", trend, *sevdMA)
	} else if sevdMA == nil {
		base += " [INSUFFICIENT_HISTORY — need 7+ days for MA/trend]"
	}

	return base
}

func classificationTitle(label string) string {
	switch label {
	case LabelExtremeFear:
		return "Extreme Fear"
	case LabelFear:
		return "Fear"
	case LabelNeutral:
		return "Neutral"
	case LabelGreed:
		return "Greed"
	case LabelExtremeGreed:
		return "Extreme Greed"
	default:
		return label
	}
}
