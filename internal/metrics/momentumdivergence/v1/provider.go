package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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
	MetricVersion = "v1.1.0"

	// starvationThreshold is the minimum CoinCount from the shared breadth endpoint
	// that indicates a full (non-truncated) API response. Distinct from SegmentsSmallMax,
	// which is the user-facing segment boundary clamp.
	starvationThreshold = 250
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
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	cgRaw, ok := data[api.CoinGeckoCoinMarketsBreadth]
	if !ok || cgRaw == nil {
		return p.unavailable("CoinGecko coin markets breadth data not available")
	}

	ranked, err := coingecko.ParseCoinMarketsRankedResponse(cgRaw)
	if err != nil {
		return p.unavailable("failed to parse coin markets data: " + err.Error())
	}

	if len(ranked.Coins) == 0 {
		return p.unavailable("no coins with valid market_cap_rank in response")
	}

	largeCeiling := DefaultLargeCeiling
	midCeiling := DefaultMidCeiling
	smallCeiling := DefaultSmallCeiling
	clamped := false
	var clampReason string

	if segsRaw, ok := config.SegmentsFromContext(ctx); ok && segsRaw != "" {
		largeCeiling, midCeiling, smallCeiling, clamped, clampReason, err = parseSegments(segsRaw)
		if err != nil {
			return p.unavailable(err.Error())
		}
	}

	input := Input{
		Coins:        ranked.Coins,
		LargeCeiling: largeCeiling,
		MidCeiling:   midCeiling,
		SmallCeiling: smallCeiling,
	}

	dataResult, compMeta, err := Compute(input)
	if err != nil {
		return p.unavailable(err.Error())
	}

	if ranked.CoinCount < starvationThreshold {
		compMeta.Confidence = "low"
	}

	dataBytes, err := json.Marshal(dataResult)
	if err != nil {
		return p.unavailable("failed to marshal data: " + err.Error())
	}

	meta := Meta{
		PrimarySource:   "coingecko",
		Confidence:      compMeta.Confidence,
		TierCounts:      compMeta.TierCounts,
		SegmentsUsed:    SegmentsUsed{LargeCeiling: largeCeiling, MidCeiling: midCeiling, SmallCeiling: smallCeiling},
		WeightingMethod: "market_cap_weighted",
		Thresholds:      compMeta.Thresholds,
		Description:     metricDescription,
		TierDetail:      compMeta.TierDetail,
	}

	if clamped {
		meta.SegmentsClamped = true
		meta.SegmentsClampedReason = clampReason
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return p.unavailable("failed to marshal meta: " + err.Error())
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

// parseSegments validates and clamps a raw "--segments large,mid,small" string.
// Returns an error for invalid format or non-ascending values; returns clamped=true
// when boundaries were adjusted to fit enforced limits.
func parseSegments(raw string) (large, mid, small int, clamped bool, reason string, err error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false, "", fmt.Errorf("invalid --segments format: must be three comma-separated values")
	}
	vals := make([]int, 3)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		v, convErr := strconv.Atoi(part)
		if convErr != nil || v < 1 {
			return 0, 0, 0, false, "", fmt.Errorf("invalid --segments value: %s", part)
		}
		vals[i] = v
	}
	if vals[0] >= vals[1] || vals[1] >= vals[2] {
		return 0, 0, 0, false, "", fmt.Errorf("invalid --segments: boundaries must be strictly ascending")
	}
	large, mid, small = vals[0], vals[1], vals[2]
	if large < SegmentsLargeMin {
		reason = fmt.Sprintf(
			"Large-cap ceiling below minimum of %d. Adjusted from %d to %d to ensure tier averages are statistically representative.",
			SegmentsLargeMin, large, SegmentsLargeMin,
		)
		large = SegmentsLargeMin
		clamped = true
	}
	if small > SegmentsSmallMax {
		r := fmt.Sprintf("Small-cap ceiling above maximum of %d. Adjusted from %d to %d.", SegmentsSmallMax, small, SegmentsSmallMax)
		if reason != "" {
			reason += " " + r
		} else {
			reason = r
		}
		small = SegmentsSmallMax
		clamped = true
	}
	return large, mid, small, clamped, reason, nil
}

// unavailable returns a MetricResult with status "unavailable" and an error message.
func (p *Provider) unavailable(msg string) (output.MetricResult, error) {
	errorData, _ := json.Marshal(map[string]string{"error": msg})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  "unavailable",
		Data:    json.RawMessage(errorData),
	}, nil
}
