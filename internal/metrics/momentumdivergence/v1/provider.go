package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

// MetricName and MetricVersion identify this provider in the registry.
const (
	MetricName    = "momentum-divergence"
	MetricVersion = "v1.0.0"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for momentum-divergence.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"md"},
		Endpoints:   []string{api.CoinGeckoCoinMarketsBreadth},
		Description: "Risk appetite gauge — measures capital rotation across market-cap tiers.",
	}
}

// RegisterFlags implements the flagRegistrar interface.
func (p *Provider) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("segments", "10,50,200",
		"Comma-separated tier boundaries: large_ceiling,mid_ceiling,small_ceiling")
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) { //nolint:gocyclo
	cgRaw, ok := data[api.CoinGeckoCoinMarketsBreadth]
	if !ok || cgRaw == nil {
		return p.unavailable("CoinGecko coin markets breadth data not available"), nil
	}

	ranked, err := coingecko.ParseCoinMarketsRankedResponse(cgRaw)
	if err != nil {
		return p.unavailable("failed to parse coin markets data: " + err.Error()), nil //nolint:nilerr
	}

	if len(ranked.Coins) == 0 {
		return p.unavailable("no coins with valid market_cap_rank in response"), nil
	}

	largeCeiling := DefaultLargeCeiling
	midCeiling := DefaultMidCeiling
	smallCeiling := DefaultSmallCeiling
	clamped := false
	var clampReason string

	if segsRaw, ok := config.SegmentsFromContext(ctx); ok && segsRaw != "" {
		parts := strings.Split(segsRaw, ",")
		if len(parts) != 3 {
			return p.unavailable("invalid --segments format: must be three comma-separated values"), nil
		}

		vals := make([]int, 3)
		for i, part := range parts {
			part = strings.TrimSpace(part)
			v, err := strconv.Atoi(part)
			if err != nil || v < 1 {
				return p.unavailable("invalid --segments value: " + part), nil //nolint:nilerr
			}
			vals[i] = v
		}

		if vals[0] >= vals[1] || vals[1] >= vals[2] {
			return p.unavailable("invalid --segments: boundaries must be strictly ascending"), nil
		}

		largeCeiling, midCeiling, smallCeiling = vals[0], vals[1], vals[2]

		if largeCeiling < SegmentsLargeMin {
			clampReason = fmt.Sprintf("Large-cap ceiling below minimum of %d. Adjusted from %d to %d.",
				SegmentsLargeMin, largeCeiling, SegmentsLargeMin)
			largeCeiling = SegmentsLargeMin
			clamped = true
		}
		if smallCeiling > SegmentsSmallMax {
			reason := fmt.Sprintf("Small-cap ceiling above maximum of %d. Adjusted from %d to %d.",
				SegmentsSmallMax, smallCeiling, SegmentsSmallMax)
			if clampReason != "" {
				clampReason += " " + reason
			} else {
				clampReason = reason
			}
			smallCeiling = SegmentsSmallMax
			clamped = true
		}
	}

	input := Input{
		Coins:                 ranked.Coins,
		LargeCeiling:          largeCeiling,
		MidCeiling:            midCeiling,
		SmallCeiling:          smallCeiling,
		SegmentsClamped:       clamped,
		SegmentsClampedReason: clampReason,
	}

	dataResult, compMeta, err := Compute(input)
	if err != nil {
		return p.unavailable(err.Error()), nil
	}

	if ranked.CoinCount < 250 {
		compMeta.Confidence = "low"
	}

	dataBytes, err := json.Marshal(dataResult)
	if err != nil {
		return p.unavailable("failed to marshal data: " + err.Error()), nil //nolint:nilerr
	}

	metaMap := map[string]interface{}{
		"cache_hit":         false,
		"ttl_remaining_sec": 0,
		"primary_source":    "coingecko",
		"confidence":        compMeta.Confidence,
		"tier_counts":       compMeta.TierCounts,
		"segments_used":     map[string]int{"large_ceiling": largeCeiling, "mid_ceiling": midCeiling, "small_ceiling": smallCeiling},
		"data_timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	if clamped {
		metaMap["segments_clamped"] = true
		metaMap["segments_clamped_reason"] = clampReason
	}

	metaMap["thresholds"] = compMeta.Thresholds
	metaMap["description"] = compMeta.LabelDescription
	if compMeta.TierDetail != nil {
		metaMap["tier_detail"] = compMeta.TierDetail
	}

	metaBytes, err := json.Marshal(metaMap)
	if err != nil {
		return p.unavailable("failed to marshal meta: " + err.Error()), nil //nolint:nilerr
	}

	status := "ok"
	if compMeta.Confidence == "low" {
		status = "degraded"
	}

	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  status,
		Data:    json.RawMessage(dataBytes),
		Meta:    json.RawMessage(metaBytes),
	}, nil
}

// unavailable returns a MetricResult with status "unavailable" and an error message.
func (p *Provider) unavailable(msg string) output.MetricResult {
	errorData, _ := json.Marshal(map[string]string{"error": msg})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  "unavailable",
		Data:    json.RawMessage(errorData),
	}
}
