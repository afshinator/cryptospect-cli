package api

const (
	// CoinGecko endpoints
	CoinGeckoGlobalMarket        = "coingecko.global_market"
	CoinGeckoSPPStablesMarkets   = "coingecko.spp_stables_markets"
	CoinGeckoDerivatives         = "coingecko.derivatives"
	CoinGeckoCoinMarketsBreadth  = "coingecko.coin_markets_breadth"
	CoinGeckoCoinMarketsMomentum = "coingecko.coin_markets_momentum"

	// Binance US endpoints
	BinanceSpotCVD_BTC_1h = "binance.spot_cvd_btc_1h"

	// CoinDesk endpoints
	CoinDeskAssetTopList = "coindesk.asset_top_list"

	// CoinMetrics endpoints
	CoinMetricsCommunity = "coinmetrics.community"
)

// AllEndpoints returns a slice of all known endpoint keys.
func AllEndpoints() []string {
	return []string{
		CoinGeckoGlobalMarket,
		CoinGeckoSPPStablesMarkets,
		CoinGeckoDerivatives,
		CoinGeckoCoinMarketsBreadth,
		CoinGeckoCoinMarketsMomentum,
		BinanceSpotCVD_BTC_1h,
		CoinDeskAssetTopList,
		CoinMetricsCommunity,
	}
}
