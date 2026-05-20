package api

const (
	// CoinGeckoGlobalMarket is the endpoint key for CoinGecko global market data.
	CoinGeckoGlobalMarket = "coingecko.global_market"
	// CoinGeckoSPPStablesMarkets is the endpoint key for CoinGecko stablecoin market data.
	CoinGeckoSPPStablesMarkets = "coingecko.spp_stables_markets"
	// CoinGeckoDerivatives is the endpoint key for CoinGecko derivatives data.
	CoinGeckoDerivatives = "coingecko.derivatives"
	// CoinGeckoCoinMarketsBreadth is the endpoint key for CoinGecko coin markets breadth data.
	CoinGeckoCoinMarketsBreadth = "coingecko.coin_markets_breadth"
	// CoinGeckoCoinMarketsMomentum is the endpoint key for CoinGecko coin markets momentum data.
	CoinGeckoCoinMarketsMomentum = "coingecko.coin_markets_momentum"

	// BinanceSpotCVD_BTC_1h is the endpoint key for Binance US spot CVD (BTC) 1-hour data.
	BinanceSpotCVD_BTC_1h = "binance.spot_cvd_btc_1h"
	// BinanceSpotCVD_ETH_1h is the endpoint key for Binance US spot CVD (ETH) 1-hour data.
	BinanceSpotCVD_ETH_1h = "binance.spot_cvd_eth_1h"
	// BinanceSpotVol_BTC_24h is the endpoint key for Binance US spot BTC 24-hour klines for vol calc.
	BinanceSpotVol_BTC_24h = "binance.spot_vol_btc_24h"

	// CoinDeskAssetTopList is the endpoint key for CoinDesk asset top list data.
	CoinDeskAssetTopList = "coindesk.asset_top_list"

	// CoinMetricsCommunity is the endpoint key for CoinMetrics community data.
	CoinMetricsCommunity = "coinmetrics.community"

	// DefiLlamaStablecoins is the endpoint key for the DefiLlama stablecoins snapshot.
	// Returns per-asset circulating supply (current + prev week) for all tracked stablecoins.
	DefiLlamaStablecoins = "defillama.stablecoins"

	// AlternativeMeFNG is the endpoint key for the alternative.me Fear & Greed Index.
	AlternativeMeFNG = "alternativeme.fng"
	// DBnomicsChinaM2 is the endpoint key for DBnomics China M2 money supply data.
	DBnomicsChinaM2 = "dbnomics.china_m2"
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
		BinanceSpotVol_BTC_24h,
		BinanceSpotCVD_ETH_1h,
		CoinDeskAssetTopList,
		CoinMetricsCommunity,
		DefiLlamaStablecoins,
		AlternativeMeFNG,
		DBnomicsChinaM2,
	}
}
