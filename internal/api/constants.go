package api

const (
	// CoinGecko endpoints
	CoinGeckoGlobalMarket      = "coingecko.global_market"
	CoinGeckoSPPStablesMarkets = "coingecko.spp_stables_markets"
	CoinGeckoDerivatives       = "coingecko.derivatives"
	CoinGeckoCoinMarkets       = "coingecko.coin_markets"

	// Binance US endpoints
	BinanceSpotCVD = "binance.spot_cvd"

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
		CoinGeckoCoinMarkets,
		BinanceSpotCVD,
		CoinDeskAssetTopList,
		CoinMetricsCommunity,
	}
}
