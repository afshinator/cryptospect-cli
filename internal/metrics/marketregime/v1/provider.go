package v1

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	mbv1 "github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

// MetricName and MetricVersion identify this provider in the registry.
const (
	MetricName    = "market-regime"
	MetricVersion = "v1.0.0"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for market-regime.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:      MetricName,
		Namespace: metrics.CoreNamespace,
		Version:   MetricVersion,
		Aliases:   []string{"mr"},
		Endpoints: []string{
			api.CoinGeckoGlobalMarket,
			api.CoinGeckoCoinMarketsBreadth,
		},
		Description: "Composite regime classification using multiple signals.",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	// ── 1. Parse global market ──
	cgGlobalRaw, ok := data[api.CoinGeckoGlobalMarket]
	if !ok || len(cgGlobalRaw) == 0 {
		return p.unavailable("missing CoinGecko global market data")
	}
	globalData, err := coingecko.ParseGlobalResponse(cgGlobalRaw)
	if err != nil {
		return p.unavailable("parsing global market: " + err.Error())
	}

	btcDom := coingecko.ParseGlobalDominance(cgGlobalRaw)
	if btcDom == nil {
		return p.unavailable("BTC dominance not available in global market response")
	}

	volUSD, volOK := globalData.GetVolumeUSD()
	mcapUSD, mcapOK := globalData.GetMarketCapUSD()
	if !volOK || !mcapOK || volUSD == 0 || mcapUSD == 0 {
		return p.unavailable("global market volume or market cap missing or zero")
	}
	lpRatio := volUSD / mcapUSD

	// ── 2. Parse coin markets breadth ──
	cgBreadthRaw, ok := data[api.CoinGeckoCoinMarketsBreadth]
	if !ok || len(cgBreadthRaw) == 0 {
		return p.unavailable("missing CoinGecko breadth data")
	}
	cgBreadth, err := coingecko.ParseCoinMarketsBreadthResponse(cgBreadthRaw)
	if err != nil {
		return p.unavailable("parsing breadth: " + err.Error())
	}

	var btcChange24h *float64
	if cgBreadth.BTCReference.Available {
		v := cgBreadth.BTCReference.PriceChange24h
		btcChange24h = &v
	}

	// ── 3. Build mb Input and compute breadth ──
	mbInput := mbv1.Input{
		TimeframeCounts: cgBreadth.TimeframeCounts,
		CoinsCounted:    cgBreadth.CoinsWithData,
		BTCChange24h:    cgBreadth.BTCReference.PriceChange24h,
		BTCAvailable:    cgBreadth.BTCReference.Available,
		KlineAvailable:  false,
		KlineClose:      0,
		KlineOpen:       0,
		KlineOpenTimeMs: 0,
		TopN:            mbv1.DefaultTopN,
		Now:             time.Now(),
	}

	mbResult, err := mbv1.Compute(&mbInput)
	if err != nil {
		return p.unavailable("market-breadth compute: " + err.Error())
	}
	if mbResult.MetricStatus == "unavailable" {
		return p.unavailable("market-breadth returned unavailable")
	}

	// ── 4. Derive MR-specific status from mb result ──
	breadthDegraded := mbResult.CoinsCounted < DegradedCoinsThreshold

	// Detect weight redistribution
	weightRedistributed := false ||
		math.Abs(mbResult.WeightsUsed["1h"]-mbv1.Weight1h) > 1e-9 ||
		math.Abs(mbResult.WeightsUsed["24h"]-mbv1.Weight24h) > 1e-9 ||
		math.Abs(mbResult.WeightsUsed["7d"]-mbv1.Weight7d) > 1e-9 ||
		math.Abs(mbResult.WeightsUsed["30d"]-mbv1.Weight30d) > 1e-9

	// ── 5. State cache: read prior dominance, write current ──
	var priorDom *float64
	var priorAge *int
	coldStart := true

	if cfg, hasCfg := config.FromContext(ctx); hasCfg {
		if c, err := cache.Open(cfg.CacheDir()); err == nil {
			entry, err := c.Get(StateKey)
			if err == nil && entry.Found {
				var cached float64
				if err := json.Unmarshal(entry.Data, &cached); err == nil {
					ageSec := int(time.Since(entry.FetchedAt).Seconds())
					if ageSec >= 0 && ageSec <= MaxSnapshotAgeSec {
						priorDom = &cached
						priorAge = &ageSec
						coldStart = false
					}
				}
			}
			// Write current dominance back
			if curBytes, err := json.Marshal(*btcDom); err == nil {
				_ = c.Set(StateKey, curBytes, StateTTLSec)
			}
			_ = c.Close()
		}
	}

	// ── 6. Compute cache freshness meta ──
	cacheHit := false
	ttlRemaining := 0
	if cfg, hasCfg := config.FromContext(ctx); hasCfg {
		if c, err := cache.Open(cfg.CacheDir()); err == nil {
			globalEntry, gErr := c.Get(api.CoinGeckoGlobalMarket)
			breadthEntry, bErr := c.Get(api.CoinGeckoCoinMarketsBreadth)
			if gErr == nil && bErr == nil && globalEntry.Found && breadthEntry.Found {
				cacheHit = !globalEntry.Stale && !breadthEntry.Stale
			}
			if cacheHit {
				now := time.Now()
				globalTTL := calcTTLRemaining(now, globalEntry)
				breadthTTL := calcTTLRemaining(now, breadthEntry)
				ttlRemaining = minInt(globalTTL, breadthTTL)
			}
			_ = c.Close()
		}
	}

	// ── 7. Build Compute input and call ──
	input := Input{
		BTCDominancePct:     *btcDom,
		PriorDominancePct:   priorDom,
		PriorSnapshotAgeSec: priorAge,
		BreadthScore:        mbResult.MarketBreadthScore.Value(),
		LPRatio:             lpRatio,
		BTCChange24h:        btcChange24h,
		WeightRedistributed: weightRedistributed,
		BreadthDegraded:     breadthDegraded,
	}
	computed := Compute(input)

	// ── 8. Build Data JSON ──
	classDesc := classificationDescription(computed.Regime, computed.Conviction)
	dataOut := Data{
		Regime:             computed.Regime,
		Modifier:           computed.Modifier,
		DominanceTrend:     computed.DominanceTrend,
		Conviction:         computed.Conviction,
		MarketBreadthScore: computed.BreadthScore,
		Classification: Classification{
			Label:       computed.Regime,
			Description: classDesc,
		},
		Summary: computed.Summary,
	}
	dataJSON, err := json.Marshal(dataOut)
	if err != nil {
		return p.unavailable("marshaling data: " + err.Error())
	}

	// ── 9. Build Meta JSON ──
	weightsOut := WeightsUsed{
		OneHour:    mbResult.WeightsUsed["1h"],
		TwentyFour: mbResult.WeightsUsed["24h"],
		SevenDay:   mbResult.WeightsUsed["7d"],
		ThirtyDay:  mbResult.WeightsUsed["30d"],
	}

	var dominanceDelta *float64
	if !coldStart && priorDom != nil {
		delta := *btcDom - *priorDom
		dominanceDelta = &delta
	}

	metaExt := MetaExtended{
		CacheHit:                     cacheHit,
		TTLRemainingSec:              ttlRemaining,
		PrimarySource:                "coingecko",
		BTCDominancePct:              *btcDom,
		BTC24hChange:                 btcChange24h,
		Confidence:                   computed.Confidence,
		DominanceColdStart:           coldStart,
		Notes:                        computed.Notes,
		CacheHintSec:                 CacheHintSec,
		LPRatio:                      lpRatio,
		DominanceDeltaSinceLastFetch: dominanceDelta,
		PriorSnapshotAgeSec:          priorAge,
		WeightsUsed:                  weightsOut,
	}

	metaFull := MetaFull{
		MetaExtended: metaExt,
		Thresholds: map[string]float64{
			"dom_dead_band_pp":      DomDeadBandPP,
			"breadth_broad":         BreadthBroadThresh,
			"breadth_narrow":        BreadthNarrowThresh,
			"btc_dir_dead_band_pct": ModifierDeadBandPP,
			"conviction_high":       ConvictionHighThresh,
			"conviction_low":        ConvictionLowThresh,
			"capitulation_vol_min":  ConvictionHighThresh,
		},
		Description: mrDescription,
	}

	metaJSON, err := json.Marshal(metaFull)
	if err != nil {
		return p.unavailable("marshaling meta: " + err.Error())
	}

	// ── 10. Determine status ──
	mrStatus := "ok"
	if breadthDegraded {
		mrStatus = "degraded"
	}

	return output.MetricResult{
		Metric:  MetricName,
		Version: MetricVersion,
		Status:  mrStatus,
		Data:    json.RawMessage(dataJSON),
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

func calcTTLRemaining(now time.Time, entry cache.Entry) int {
	expiresAt := entry.FetchedAt.Add(time.Duration(entry.TTLSeconds) * time.Second)
	remaining := int(expiresAt.Unix() - now.Unix())
	if remaining < 0 {
		return 0
	}
	return remaining
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const mrDescription = "Market Regime is the structural context layer of the suite. " +
	"It cross-references Bitcoin Dominance trend against Market Breadth Score, " +
	"gated by BTC price direction and liquidity conviction, to classify the " +
	"market into one of ten named structural phases including BTC-Led Expansion, " +
	"Institutional Build, Flight to Safety, Steady Appreciation, Consolidation, " +
	"Stagnation, Alt-Season / Mania, Capital Rotation, Capitulation, and Structural Decay."
