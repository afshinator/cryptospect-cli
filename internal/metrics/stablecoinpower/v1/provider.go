// provider.go — stablecoin-power metric provider (v1)
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/api/defillama"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// MetricName is the canonical name for this metric.
const MetricName = "stablecoin-power"

// MetricVersion is the SemVer version of this metric provider.
const MetricVersion = "v1.0.0"

// Classification labels for the stablecoin-power metric.
const (
	ClassificationHigh   = "high"
	ClassificationNormal = "normal"
	ClassificationLow    = "low"
)

const (
	thresholdHigh   = 0.15
	thresholdNormal = 0.07

	discrepancyMediumThreshold = 0.15
	discrepancyLowThreshold    = 0.25

	minTopN = 8
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for stablecoin-power.
type Provider struct{}

// Classification holds the categorical outcome of the stablecoin-power metric.
type Classification struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// StablecoinEntry is a single coin included in the numerator (full detail only).
type StablecoinEntry struct {
	Symbol  string              `json:"symbol"`
	McapUSD metrics.MetricFloat `json:"mcap_usd"`
}

// Data is the core metric output.
type Data struct {
	StablePowerRatio   metrics.MetricFloat `json:"stable_power_ratio"`
	StableMcapUSD      metrics.MetricFloat `json:"stable_mcap_usd"`
	VolatileMcapUSD    metrics.MetricFloat `json:"volatile_mcap_usd"`
	SupplyTrend7d      string              `json:"supply_trend_7d"`
	StablecoinsCounted int                 `json:"stablecoins_counted"`
	Classification     Classification      `json:"classification"`
	Summary            string              `json:"summary"`
}

// Meta holds extended and full-detail metadata (filtered by detail level in root.go).
type Meta struct {
	PrimarySource       string             `json:"primary_source"`
	ValidatorSource     string             `json:"validator_source"`
	DiscrepancyDetected bool               `json:"discrepancy_detected,omitempty"`
	DiscrepancyNote     string             `json:"discrepancy_note,omitempty"`
	Confidence          string             `json:"confidence"`
	SupplyTrendSource   string             `json:"supply_trend_source"`
	StablecoinScope     string             `json:"stablecoin_scope"`
	TopClamped          bool               `json:"top_clamped,omitempty"`
	TopClampedReason    string             `json:"top_clamped_reason,omitempty"`
	Thresholds          map[string]float64 `json:"thresholds,omitempty"`
	Description         string             `json:"description,omitempty"`
	TopNStablecoins     []StablecoinEntry  `json:"top_n_stablecoins,omitempty"`
}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Aliases:   []string{"sp"},
		Endpoints: []string{
			api.CoinGeckoGlobalMarket,
			api.CoinGeckoSPPStablesMarkets,
			api.DefiLlamaStablecoins,
		},
		Description: "Measures stablecoin dominance as dry powder relative to the volatile market.",
	}
}

// RegisterFlags satisfies the private flagRegistrar interface in root.go.
func (p *Provider) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().Int("top", minTopN, "Number of top stablecoins by market cap to include (minimum 8)")
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	// --- Global market cap ---
	globalRaw := data[api.CoinGeckoGlobalMarket]
	if len(globalRaw) == 0 {
		return p.unavailable("missing global market data")
	}
	globalData, err := coingecko.ParseGlobalResponse(globalRaw)
	if err != nil {
		return p.unavailable(fmt.Sprintf("parsing global: %v", err))
	}
	totalMcap, ok := globalData.GetMarketCapUSD()
	if !ok || totalMcap == 0 {
		return p.unavailable("total market cap usd not available")
	}

	// --- Stablecoin markets ---
	stablesRaw := data[api.CoinGeckoSPPStablesMarkets]
	if len(stablesRaw) == 0 {
		return p.unavailable("missing stablecoin markets data")
	}
	stables, err := coingecko.ParseStablesMarketsResponse(stablesRaw)
	if err != nil {
		return p.unavailable(fmt.Sprintf("parsing stables: %v", err))
	}
	if len(stables) == 0 {
		return p.unavailable("no stablecoin data in response")
	}

	// Filter by config IDs when set.
	cfg, hasCfg := config.FromContext(ctx)
	if hasCfg && len(cfg.Metrics.StablecoinPower.StablecoinIDs) > 0 {
		idSet := make(map[string]struct{}, len(cfg.Metrics.StablecoinPower.StablecoinIDs))
		for _, id := range cfg.Metrics.StablecoinPower.StablecoinIDs {
			idSet[id] = struct{}{}
		}
		filtered := stables[:0]
		for _, s := range stables {
			if _, ok := idSet[s.ID]; ok {
				filtered = append(filtered, s)
			}
		}
		stables = filtered
		if len(stables) == 0 {
			return p.unavailable("no stablecoins match configured stablecoin_ids")
		}
	}

	// Sort by market cap descending.
	sort.Slice(stables, func(i, j int) bool {
		return stables[i].MarketCap > stables[j].MarketCap
	})

	// --- Top-N selection ---
	topN := minTopN
	topClamped := false
	topClampedReason := ""
	if n, ok := config.TopNFromContext(ctx); ok {
		if n < minTopN {
			topClamped = true
			topClampedReason = fmt.Sprintf(
				"Minimum %d stablecoins required for metric integrity. Value adjusted from %d to %d.",
				minTopN, n, minTopN,
			)
		} else {
			topN = n
		}
	}
	if topN > len(stables) {
		topN = len(stables)
	}
	topStables := stables[:topN]

	// --- Compute ratio ---
	var stableMcap float64
	for _, s := range topStables {
		stableMcap += s.MarketCap
	}
	volatileMcap := totalMcap - stableMcap
	if volatileMcap <= 0 {
		return p.unavailable("volatile market cap is zero or negative")
	}
	ratio := stableMcap / volatileMcap

	// --- DefiLlama validator (optional) ---
	supplyTrend := defillama.TrendStable
	confidence := "high"
	discrepancyDetected := false
	discrepancyNote := ""

	if dlRaw := data[api.DefiLlamaStablecoins]; len(dlRaw) > 0 {
		if dlResp, err := defillama.ParseStablecoinsResponse(dlRaw); err == nil {
			dlCurrent, dlPrev := defillama.AggregateSupply(dlResp)
			supplyTrend = defillama.ClassifyTrend(dlCurrent, dlPrev)
			if dlCurrent > 0 {
				discPct := math.Abs(stableMcap-dlCurrent) / dlCurrent
				switch {
				case discPct >= discrepancyLowThreshold:
					confidence = "low"
					discrepancyDetected = true
					discrepancyNote = fmt.Sprintf(
						"Coverage scope difference: %.1f%% delta between CoinGecko top-%d and DefiLlama aggregate supply.",
						discPct*100, len(topStables),
					)
				case discPct >= discrepancyMediumThreshold:
					confidence = "medium"
					discrepancyDetected = true
					discrepancyNote = fmt.Sprintf(
						"Coverage scope difference: %.1f%% delta between CoinGecko top-%d and DefiLlama aggregate supply.",
						discPct*100, len(topStables),
					)
				}
			}
		}
	}

	// --- Classify ---
	cl := classify(ratio, supplyTrend)
	summaryStr := buildSummary(ratio, cl, supplyTrend)

	// --- Build top-N breakdown (for full detail) ---
	topNList := make([]StablecoinEntry, len(topStables))
	for i, s := range topStables {
		topNList[i] = StablecoinEntry{
			Symbol:  strings.ToUpper(s.Symbol),
			McapUSD: metrics.Currency(s.MarketCap),
		}
	}

	// --- Data ---
	d := Data{
		StablePowerRatio:   metrics.Ratio(ratio),
		StableMcapUSD:      metrics.Currency(stableMcap),
		VolatileMcapUSD:    metrics.Currency(volatileMcap),
		SupplyTrend7d:      supplyTrend,
		StablecoinsCounted: len(topStables),
		Classification:     cl,
		Summary:            summaryStr,
	}
	dJSON, err := json.Marshal(d)
	if err != nil {
		return p.unavailable("marshaling data: " + err.Error())
	}

	// --- Meta ---
	meta := Meta{
		PrimarySource:     "coingecko",
		ValidatorSource:   "defillama",
		Confidence:        confidence,
		SupplyTrendSource: "defillama",
		StablecoinScope:   "all_usd_equivalent",
		Thresholds: map[string]float64{
			"high":       thresholdHigh,
			"normal_min": thresholdNormal,
		},
		Description:     metricDescription,
		TopNStablecoins: topNList,
	}
	if discrepancyDetected {
		meta.DiscrepancyDetected = true
		meta.DiscrepancyNote = discrepancyNote
	}
	if topClamped {
		meta.TopClamped = true
		meta.TopClampedReason = topClampedReason
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return p.unavailable("marshaling meta: " + err.Error())
	}

	status := metrics.DetectStatus(metrics.ConfidenceToFloat(confidence), false)
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  status,
		Data:    json.RawMessage(dJSON),
		Meta:    json.RawMessage(metaJSON),
	}, nil
}

func (p *Provider) unavailable(msg string) (output.MetricResult, error) {
	errMsg, _ := json.Marshal(map[string]string{"error": msg})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  "unavailable",
		Data:    json.RawMessage(errMsg),
	}, nil
}

func classify(ratio float64, trend string) Classification {
	switch {
	case ratio >= thresholdHigh:
		return Classification{Label: ClassificationHigh, Description: "Dry Powder Alert"}
	case ratio >= thresholdNormal:
		return Classification{Label: ClassificationNormal, Description: "Healthy Balance"}
	default:
		if trend == defillama.TrendContracting {
			return Classification{Label: ClassificationLow, Description: "Capital Flight"}
		}
		return Classification{Label: ClassificationLow, Description: "Overextended"}
	}
}

func buildSummary(ratio float64, cl Classification, trend string) string {
	if cl.Label == ClassificationLow {
		return fmt.Sprintf("SP Ratio: %.4f | %s | Supply: %s", ratio, cl.Description, trend)
	}
	return fmt.Sprintf("SP Ratio: %.4f | %s", ratio, cl.Description)
}

const metricDescription = "Stablecoin Power measures the ratio of aggregate stablecoin market cap to the " +
	"volatile crypto market cap. High (>0.15): Dry Powder Alert — substantial sidelined capital. " +
	"Normal (0.07–0.15): Healthy Balance — adequate fuel for current price levels. " +
	"Low (<0.07): depleted dry powder — disambiguated by supply_trend_7d: stable/expanding = Overextended " +
	"(rally on fumes); contracting = Capital Flight (macro exodus)."
