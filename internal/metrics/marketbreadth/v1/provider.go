package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

// MetricName and MetricVersion identify this provider in the registry.
const (
	MetricName    = "market-breadth"
	MetricVersion = "v1.0.0"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for market-breadth.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Aliases:   []string{"mb"},
		Endpoints: []string{
			api.CoinGeckoCoinMarketsBreadth,
			api.BinanceSpotCVD_BTC_1h,
		},
		Description: "Measures market participation across top assets.",
	}
}

// RegisterFlags satisfies the flagRegistrar interface.
func (p *Provider) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().Int("top", DefaultTopN, "Number of top coins by market cap to include (min 50, max 250 in v1)")
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	cgRaw, ok := data[api.CoinGeckoCoinMarketsBreadth]
	if !ok || len(cgRaw) == 0 {
		return p.unavailable("missing CoinGecko breadth data")
	}
	cgData, err := coingecko.ParseCoinMarketsBreadthResponse(cgRaw)
	if err != nil {
		return p.unavailable(fmt.Sprintf("parsing CoinGecko breadth: %v", err))
	}

	var klineClose, klineOpen float64
	var klineOpenTimeMs int64
	klineAvailable := false

	if bnRaw, ok := data[api.BinanceSpotCVD_BTC_1h]; ok && len(bnRaw) > 0 {
		klines, err := binance.ParseKlinesResponse(bnRaw)
		if err == nil {
			klineClose = klines.Close
			klineOpen = klines.Open
			klineOpenTimeMs = klines.OpenTime
			klineAvailable = true
		}
	}

	topN := DefaultTopN
	topClamped := false
	topClampedReason := ""
	if n, ok := config.TopNFromContext(ctx); ok {
		switch {
		case n < MinTopN:
			topClamped = true
			topClampedReason = fmt.Sprintf("Minimum %d coins required for statistically significant breadth. Value adjusted from %d to %d.", MinTopN, n, MinTopN)
			topN = MinTopN
		case n > MaxTopN:
			topClamped = true
			topClampedReason = fmt.Sprintf("Maximum %d coins enforced in v1 to maintain single-call predictability. Values above %d require pagination and risk rate limits on the Free/Demo tier.", MaxTopN, MaxTopN)
			topN = MaxTopN
		default:
			topN = n
		}
	}

	input := Input{
		TimeframeCounts: cgData.TimeframeCounts,
		CoinsCounted:    cgData.CoinsWithData,
		BTCChange24h:    cgData.BTCReference.PriceChange24h,
		BTCAvailable:    cgData.BTCReference.Available,
		KlineClose:      klineClose,
		KlineOpen:       klineOpen,
		KlineOpenTimeMs: klineOpenTimeMs,
		KlineAvailable:  klineAvailable,
		TopN:            topN,
		Now:             time.Now(),
	}

	result, err := Compute(&input)
	if err != nil {
		return p.unavailable(fmt.Sprintf("compute: %v", err))
	}

	dJSON, err := json.Marshal(result)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling data: %v", err))
	}

	timeframeCounts := make(map[string]coingecko.TimeframeMetric, 4)
	for _, tf := range timeframeOrder {
		timeframeCounts[tf] = coingecko.TimeframeMetric{
			GreenCount: result.GreenCounts[tf],
			TotalCount: result.TotalCounts[tf],
		}
	}

	meta := Meta{
		PrimarySource:       "coingecko",
		ValidatorSource:     "binance_us",
		DiscrepancyDetected: result.DiscrepancyDetected,
		DiscrepancyNote:     result.DiscrepancyNote,
		Confidence:          result.ValidatorConfidence,
		WeightsUsed:         result.WeightsUsed,
		TimeframeCounts:     timeframeCounts,
		Thresholds: map[string]float64{
			"broad":                     BroadThreshold,
			"narrow":                    NarrowThreshold,
			"divergence_btc_change_min": DivergenceBTCChangeMin,
			"divergence_breadth_max":    NarrowThreshold,
		},
		Description: metricDescription,
	}

	if topClamped {
		meta.TopClamped = true
		meta.TopClampedReason = topClampedReason
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling meta: %v", err))
	}

	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  result.MetricStatus,
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

const metricDescription = "Market Breadth measures the percentage of top assets in uptrends across four " +
	"timeframes (1h, 24h, 7d, 30d). A broad market has many coins participating above their recent ranges; " +
	"a narrow market suggests the rally is concentrated in few leaders. Cross-referenced with Binance BTC spot " +
	"klines to detect divergences between breadth and price direction."
