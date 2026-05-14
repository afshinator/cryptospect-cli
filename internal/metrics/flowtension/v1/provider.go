package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/binance"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// MetricName is the canonical name for this metric.
const MetricName = "flow-tension"

// MetricVersion is the SemVer version of this metric provider.
const MetricVersion = "v1.0.0"

// Cache key for persisting OI values across runs (for 24h change computation).
const oiCacheKey = "flowtension_oi_btc"

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for flow-tension.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Aliases:   []string{"ft"},
		Endpoints: []string{
			api.BinanceSpotCVD_BTC_1h,
			api.CoinGeckoDerivatives,
		},
		Description: "CVD-based market pressure, OI trend, and funding sentiment.",
	}
}

// unavailable is a helper to build an unavailable MetricResult.
func (p *Provider) unavailable(msg string) (output.MetricResult, error) {
	errMsg, _ := json.Marshal(map[string]string{"error": msg})
	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  "unavailable",
		Data:    json.RawMessage(errMsg),
	}, nil
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	// ── Parse Binance CVD data (required) ──
	binanceRaw, ok := data[api.BinanceSpotCVD_BTC_1h]
	if !ok || len(binanceRaw) == 0 {
		return p.unavailable("missing Binance CVD data")
	}
	klines, err := binance.ParseKlinesResponse(binanceRaw)
	if err != nil {
		return p.unavailable(fmt.Sprintf("parsing Binance klines: %v", err))
	}
	takerSell := klines.TotalVolume - klines.TakerBuyVolume

	// ── Parse CoinGecko derivatives data (optional — degraded if missing) ──
	var totalOI float64
	var fundingRate float64
	var exchangeCount int
	cgAvailable := false

	if cgRaw, ok := data[api.CoinGeckoDerivatives]; ok && len(cgRaw) > 0 {
		cgData, err := coingecko.ParseDerivativesResponse(cgRaw)
		if err == nil {
			totalOI = cgData.TotalOpenInterest
			exchangeCount = cgData.ExchangeCount
			fundingRate = cgData.MedianFundingRate
			cgAvailable = true
		}
	}

	// ── Read cached OI for 24h change ──
	var prevOI *float64
	if cgAvailable && totalOI > 0 {
		cfg, hasCfg := config.FromContext(ctx)
		if hasCfg {
			cacheDir := cfg.CacheDir()
			c, err := cache.Open(cacheDir)
			if err == nil {
				entry, err := c.Get(oiCacheKey)
				if err == nil && entry.Found {
					var cached float64
					if err := json.Unmarshal(entry.Data, &cached); err == nil && cached > 0 {
						prevOI = &cached
					}
				}
				// Write current OI to cache for next run
				if curRaw, err := json.Marshal(totalOI); err == nil {
					_ = c.Set(oiCacheKey, curRaw, 86400) // 24h TTL
				}
				_ = c.Close()
			}
		}
	}

	// ── Build Input and compute ──
	input := Input{
		TakerBuyVolume:    klines.TakerBuyVolume,
		TakerSellVolume:   takerSell,
		TotalVolume:       klines.TotalVolume,
		NumTrades:         klines.NumTrades,
		TotalOpenInterest: totalOI,
		PrevOI:            prevOI,
		FundingRate:       fundingRate,
		ExchangeCount:     exchangeCount,
	}

	computed, err := Compute(input)
	if err != nil {
		return p.unavailable(fmt.Sprintf("compute: %v", err))
	}

	dJSON, err := json.Marshal(computed)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling data: %v", err))
	}

	// ── Status (from source availability) ──
	conf := 0.9
	if !cgAvailable {
		conf = 0.6
	}
	status := metrics.DetectStatus(conf, false)

	// ── Meta ──
	sources := []string{"binance_us"}
	if cgAvailable {
		sources = append(sources, "coingecko")
	}
	meta := Meta{
		PrimarySources:  sources,
		Confidence:      metrics.FloatToConfidence(conf),
		CvdSampleTrades: klines.NumTrades,
		OIExchangeCount: exchangeCount,
		Instrument:      "btc",
	}
	// Full-detail fields computed unconditionally (filtered by detail level in root.go)
	meta.Thresholds = map[string]any{
		"cvd":     map[string]float64{"aggressive": FlowAggressiveThreshold},
		"oi":      map[string]float64{"building": OIBuildingThreshold, "unwinding": OIUnwindingThreshold},
		"funding": map[string]float64{"negative": FundingNegativeThreshold, "positive": FundingPositiveThreshold, "overheated": FundingOverheatedThreshold},
	}
	meta.Description = metricDescription

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return p.unavailable(fmt.Sprintf("marshaling meta: %v", err))
	}

	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  status,
		Data:    json.RawMessage(dJSON),
		Meta:    json.RawMessage(metaJSON),
	}, nil
}

const metricDescription = "Flow tension measures the kinetic energy of the market — how aggressively traders " +
	"are using leverage and moving assets onto exchanges to trade. Three signals: CVD (taker buy/sell aggression " +
	"from Binance spot), Open Interest (aggregated across 179+ exchanges via CoinGecko), and Funding Rate " +
	"(Binance Futures BTC perpetual). No API key required — all sources are public."
