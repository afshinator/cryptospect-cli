package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/dbnomics"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for china-m2.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"cnm2"},
		Endpoints:   []string{api.DBnomicsChinaM2},
		Description: "China M2 money supply — macro liquidity indicator from National Bureau of Statistics of China.",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(_ context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	m2Raw, ok := data[api.DBnomicsChinaM2]
	if !ok || len(m2Raw) == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "missing China M2 data")
	}

	parsed, err := dbnomics.ParseChinaM2Response(m2Raw)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("parsing China M2 data: %v", err))
	}

	obs := make([]Observation, len(parsed.Observations))
	// API returns oldest-first; reverse so Observations[0] is most recent
	for i, o := range parsed.Observations {
		obs[len(obs)-1-i] = Observation{Period: o.Period, Value: o.Value}
	}

	input := Input{Observations: obs}

	dataResult, compMeta, err := Compute(input)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, err.Error())
	}

	dataBytes, err := json.Marshal(dataResult)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling data: %v", err))
	}

	meta := Meta{
		PrimarySource: "dbnomics",
		Confidence:    compMeta.Confidence,
		DataFrequency: "monthly",
		Units:         "CNY billion",
		Thresholds: map[string]float64{
			"expanding_min_yoy": YoYExpandingMin,
			"slowing_max_yoy":   YoYSlowingMax,
		},
		Description: metricDescription,
	}

	if compMeta.DataLagDays > 0 {
		meta.DataLagDays = compMeta.DataLagDays
		// Check for stale data
		if compMeta.DataLagDays > 90 && compMeta.Confidence == ConfidenceHigh {
			meta.Confidence = ConfidenceMedium
			compMeta.Confidence = ConfidenceMedium
		}
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
