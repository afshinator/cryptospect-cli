package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/alternativeme"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for fear-greed-index.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"fgi"},
		Endpoints:   []string{api.AlternativeMeFNG},
		Description: "Crypto Fear & Greed Index — crowd sentiment oscillator (0-100).",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(_ context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	fngRaw, ok := data[api.AlternativeMeFNG]
	if !ok || len(fngRaw) == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "missing fear & greed data")
	}

	points, err := alternativeme.ParseFNGResponse(fngRaw)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("parsing FNG data: %v", err))
	}

	values := make([]int, len(points))
	for i, pt := range points {
		values[i] = pt.Value
	}

	input := Input{Values: values}

	dataResult, compMeta, err := Compute(input)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, err.Error())
	}

	dataBytes, err := json.Marshal(dataResult)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling data: %v", err))
	}

	meta := Meta{
		PrimarySource: "alternative.me",
		Confidence:    compMeta.Confidence,
		Thresholds: map[string]float64{
			"extreme_fear_max": BandExtremeFearMax,
			"fear_max":         BandFearMax,
			"neutral_max":      BandNeutralMax,
			"greed_max":        BandGreedMax,
			"trend_dead_band":  TrendDeadBand,
		},
		Description: metricDescription,
	}

	// Populate optional meta fields
	if len(points) > 0 {
		meta.Timestamp = points[0].Timestamp
		if tus, err := strconv.Atoi(points[0].TimeUntilUpdate); err == nil {
			meta.TimeUntilUpdateSec = tus
		}
	}
	if compMeta.SevdMA != nil {
		ma := metrics.Ratio(*compMeta.SevdMA)
		meta.SevdMA = &ma
	}
	if compMeta.Trend != "" {
		meta.Trend = compMeta.Trend
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling meta: %v", err))
	}

	conf := metrics.ConfidenceToFloat(compMeta.Confidence)
	status := metrics.DetectStatus(conf, false)

	return output.MetricResult{
		Metric:    MetricName,
		Version:   MetricVersion,
		Namespace: metrics.CoreNamespace,
		Status:    status,
		Data:      json.RawMessage(dataBytes),
		Meta:      json.RawMessage(metaBytes),
	}, nil
}
