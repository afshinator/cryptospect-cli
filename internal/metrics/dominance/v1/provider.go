package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/afshinator/cryptospect-cli/internal/api"
	"github.com/afshinator/cryptospect-cli/internal/api/coingecko"
	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func init() { metrics.MustRegister(&Provider{}) }

// Provider implements metrics.MetricProvider for dominance.
type Provider struct{}

// Def implements metrics.MetricProvider.
func (p *Provider) Def() metrics.MetricDef {
	return metrics.MetricDef{
		Name:        MetricName,
		Namespace:   metrics.CoreNamespace,
		Version:     MetricVersion,
		Aliases:     []string{"dom"},
		Endpoints:   []string{api.CoinGeckoGlobalMarket},
		Description: "Tracks BTC and ETH market cap dominance and trend direction.",
	}
}

// Compute implements metrics.MetricProvider.
func (p *Provider) Compute(ctx context.Context, data map[string]json.RawMessage) (output.MetricResult, error) {
	cgRaw, ok := data[api.CoinGeckoGlobalMarket]
	if !ok || len(cgRaw) == 0 {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, "missing CoinGecko global market data")
	}

	domData, err := coingecko.ParseGlobalDominanceBoth(cgRaw)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("parsing dominance data: %v", err))
	}

	// ── State cache: read prior dominance, write current ──
	var priorBTC, priorETH *float64
	var priorAge *int

	if cfg, hasCfg := config.FromContext(ctx); hasCfg {
		if c, err := cache.Open(cfg.CacheDir()); err == nil {
			// Read prior BTC
			if entry, err := c.Get(StateKeyBTCDom); err == nil && entry.Found {
				var cached float64
				if err := json.Unmarshal(entry.Data, &cached); err == nil {
					ageSec := int(time.Since(entry.FetchedAt).Seconds())
					if ageSec >= 0 {
						priorBTC = &cached
						priorAge = &ageSec
					}
				}
			}
			// Read prior ETH
			if entry, err := c.Get(StateKeyETHDom); err == nil && entry.Found {
				var cached float64
				if err := json.Unmarshal(entry.Data, &cached); err == nil {
					if priorAge == nil {
						// Only set ETH prior if we also have BTC prior (consistent snapshot)
						priorETH = &cached
					} else {
						ageSec := int(time.Since(entry.FetchedAt).Seconds())
						if ageSec >= 0 {
							priorETH = &cached
						}
					}
				}
			}

			// Write current values
			if curBTC, err := json.Marshal(domData.BTC); err == nil {
				_ = c.Set(StateKeyBTCDom, curBTC, StateTTLSec)
			}
			if curETH, err := json.Marshal(domData.ETH); err == nil {
				_ = c.Set(StateKeyETHDom, curETH, StateTTLSec)
			}
			_ = c.Close()
		}
	}

	// Build Compute input
	input := Input{
		BTCDominancePct:     domData.BTC,
		ETHDominancePct:     domData.ETH,
		PriorBTCDominance:   priorBTC,
		PriorETHDominance:   priorETH,
		PriorSnapshotAgeSec: priorAge,
	}

	computed, compMeta, err := Compute(input)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, err.Error())
	}

	dataBytes, err := json.Marshal(computed)
	if err != nil {
		return metrics.UnavailableResult(MetricName, MetricVersion, metrics.CoreNamespace, fmt.Sprintf("marshaling data: %v", err))
	}

	meta := Meta{
		PrimarySource:       "coingecko",
		Confidence:          compMeta.Confidence,
		ColdStart:           compMeta.ColdStart,
		PriorSnapshotAgeSec: priorAge,
		Thresholds: map[string]float64{
			"btc_dead_band_pp": BTCDeadBandPP,
			"eth_dead_band_pp": ETHDeadBandPP,
		},
		Description: metricDescription,
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
