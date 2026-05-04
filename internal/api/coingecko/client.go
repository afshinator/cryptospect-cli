package coingecko

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BaseURL is the base URL for the CoinGecko API v3.
const BaseURL = "https://api.coingecko.com/api/v3"

// GlobalData holds the global market data from CoinGecko /global endpoint.
type GlobalData struct {
	TotalMarketCap                  map[string]float64 `json:"total_market_cap"`
	TotalVolume                     map[string]float64 `json:"total_volume"`
	MarketCapChangePercentage24hUsd float64            `json:"market_cap_change_percentage_24h_usd"`
}

// GlobalResponse is the wrapper around GlobalData in the API response.
type GlobalResponse struct {
	Data GlobalData `json:"data"`
}

// GlobalURL returns the URL for the CoinGecko /global endpoint.
func GlobalURL() string {
	return fmt.Sprintf("%s/global", BaseURL)
}

// ParseGlobalResponse parses the raw JSON response from /global.
func ParseGlobalResponse(body []byte) (GlobalData, error) {
	var resp GlobalResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return GlobalData{}, fmt.Errorf("parsing global response: %w", err)
	}
	return resp.Data, nil
}

// GetVolumeUSD returns the total volume in USD, if present.
func (g *GlobalData) GetVolumeUSD() (float64, bool) {
	v, ok := g.TotalVolume["usd"]
	return v, ok
}

// GetMarketCapUSD returns the total market cap in USD, if present.
func (g *GlobalData) GetMarketCapUSD() (float64, bool) {
	v, ok := g.TotalMarketCap["usd"]
	return v, ok
}

// ParseGlobalDominance extracts BTC dominance (%) from a global response.
// Returns nil if not available.
func ParseGlobalDominance(body []byte) *float64 {
	if len(body) == 0 {
		return nil
	}

	var resp struct {
		Data struct {
			MarketCapPercentage map[string]float64 `json:"market_cap_percentage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	btc, ok := resp.Data.MarketCapPercentage["btc"]
	if !ok {
		return nil
	}
	return &btc
}

// SPPStableIDs is the authoritative list of stablecoin CoinGecko IDs for the
// stablecoin‑power metric. Ordered by approximate market cap rank.
var SPPStableIDs = []string{
	"tether", "usd-coin", "usds", "ethena-usde", "dai", "paypal-usd",
	"usd1-wlfi", "falcon-finance", "global-dollar", "bfusd", "ripple-usd",
	"usdtb", "usdd", "first-digital-usd", "usual-usd", "true-usd", "gho", "usdb",
}

// StableCoinMarket holds market data for a single stablecoin from /coins/markets.
type StableCoinMarket struct {
	ID            string   `json:"id"`
	Symbol        string   `json:"symbol"`
	MarketCap     float64  `json:"market_cap"`
	TotalVolume   float64  `json:"total_volume"`
	McapChange24h float64  `json:"market_cap_change_percentage_24h"`
	CurrentPrice  *float64 `json:"current_price"`
}

// StablesMarketsURL returns the URL for the CoinGecko /coins/markets endpoint
// configured to fetch the stablecoins defined in SPPStableIDs.
func StablesMarketsURL() string {
	return fmt.Sprintf(
		"%s/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&price_change_percentage=24h",
		BaseURL, strings.Join(SPPStableIDs, ","),
	)
}

// ParseStablesMarketsResponse parses the raw JSON response from /coins/markets
// for stablecoins.
func ParseStablesMarketsResponse(body []byte) ([]StableCoinMarket, error) {
	var resp []StableCoinMarket
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing stables markets response: %w", err)
	}
	return resp, nil
}

// DerivativesData holds aggregated BTC perpetual derivatives data
// computed from the CoinGecko /derivatives endpoint response.
type DerivativesData struct {
	TotalOpenInterest float64 // sum of OI across all BTC perpetual exchanges (USD)
	MedianFundingRate float64 // median funding rate across all BTC perpetual exchanges
	ExchangeCount     int     // number of exchanges contributing BTC perpetual data
}

// DerivativesEntry is a single entry from the /derivatives response.
type DerivativesEntry struct {
	Market       string  `json:"market"`
	Symbol       string  `json:"symbol"`
	IndexID      string  `json:"index_id"`
	ContractType string  `json:"contract_type"`
	FundingRate  float64 `json:"funding_rate"`
	OpenInterest float64 `json:"open_interest"`
}

// DerivativesURL returns the URL for the CoinGecko /derivatives endpoint.
// When key is non‑empty, the CoinGecko demo API key is appended as a query param.
func DerivativesURL(key string) string {
	u := fmt.Sprintf("%s/derivatives", BaseURL)
	if key != "" {
		u += fmt.Sprintf("?x_cg_demo_api_key=%s", key)
	}
	return u
}

// ParseDerivativesResponse parses the CoinGecko /derivatives response and
// returns aggregated BTC perpetual data: total OI (sum across exchanges)
// and median funding rate (resistant to outlier exchanges).
//
// The raw response contains entries for all coins and all exchanges.
// This function filters for index_id=="BTC" and contract_type=="perpetual" only.
func ParseDerivativesResponse(body []byte) (DerivativesData, error) {
	if len(body) == 0 {
		return DerivativesData{}, fmt.Errorf("empty response body")
	}

	var entries []DerivativesEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return DerivativesData{}, fmt.Errorf("parsing derivatives response: %w", err)
	}

	var totalOI float64
	var fundingRates []float64

	for _, e := range entries {
		if e.IndexID != "BTC" || e.ContractType != "perpetual" {
			continue
		}
		totalOI += e.OpenInterest
		fundingRates = append(fundingRates, e.FundingRate)
	}

	data := DerivativesData{
		TotalOpenInterest: totalOI,
		ExchangeCount:     len(fundingRates),
	}

	if len(fundingRates) > 0 {
		data.MedianFundingRate = median(fundingRates)
	}

	return data, nil
}

// TimeframeMetric holds raw green and total counts for a single timeframe.
type TimeframeMetric struct {
	GreenCount int `json:"green"`
	TotalCount int `json:"total"`
}

// BTCReference holds the BTC 24h price change extracted from the response.
type BTCReference struct {
	PriceChange24h float64 `json:"price_change_24h_pct"`
	Available      bool    `json:"-"`
}

// CoinMarketsBreadthData holds per-timeframe counts and BTC reference
// from the CoinGecko /coins/markets response.
type CoinMarketsBreadthData struct {
	TimeframeCounts map[string]TimeframeMetric `json:"timeframe_counts"`
	CoinCount       int                        `json:"coin_count"`
	CoinsWithData   int                        `json:"-"`
	BTCReference    BTCReference               `json:"btc_reference"`
}

// CoinMarketsBreadthEntry is a single coin entry from /coins/markets with
// multi‑timeframe price change percentages.
type CoinMarketsBreadthEntry struct {
	ID        string   `json:"id"`
	Symbol    string   `json:"symbol"`
	Change1h  *float64 `json:"price_change_percentage_1h_in_currency"`
	Change24h *float64 `json:"price_change_percentage_24h_in_currency"`
	Change7d  *float64 `json:"price_change_percentage_7d_in_currency"`
	Change30d *float64 `json:"price_change_percentage_30d_in_currency"`
}

// CoinMarketsBreadthURL returns the URL for the CoinGecko /coins/markets
// endpoint configured for breadth calculation. Fetches the top N coins
// with multi‑timeframe price change percentages.
func CoinMarketsBreadthURL(perPage int) string {
	return fmt.Sprintf(
		"%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&page=1&sparkline=false&price_change_percentage=1h%%2C24h%%2C7d%%2C30d",
		BaseURL, perPage,
	)
}

// ParseCoinMarketsBreadthResponse parses the CoinGecko /coins/markets response
// and computes per‑timeframe green coin counts and related data.
//
// Each timeframe has its own denominator (coins with non‑null values for that
// timeframe). Null price change values are excluded from that timeframe's total.
// BTC reference data is extracted from the response if present.
func ParseCoinMarketsBreadthResponse(body []byte) (CoinMarketsBreadthData, error) {
	if len(body) == 0 {
		return CoinMarketsBreadthData{}, fmt.Errorf("empty response body")
	}

	var entries []CoinMarketsBreadthEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return CoinMarketsBreadthData{}, fmt.Errorf("parsing coin markets response: %w", err)
	}
	if len(entries) == 0 {
		return CoinMarketsBreadthData{}, fmt.Errorf("no coins in response")
	}

	counts := map[string]*TimeframeMetric{
		"1h":  {},
		"24h": {},
		"7d":  {},
		"30d": {},
	}

	var coinsWithData int
	var btcRef BTCReference

	for _, e := range entries {
		hasData := false

		if e.Change1h != nil {
			counts["1h"].TotalCount++
			if *e.Change1h > 0 {
				counts["1h"].GreenCount++
			}
			hasData = true
		}
		if e.Change24h != nil {
			counts["24h"].TotalCount++
			if *e.Change24h > 0 {
				counts["24h"].GreenCount++
			}
			hasData = true
		}
		if e.Change7d != nil {
			counts["7d"].TotalCount++
			if *e.Change7d > 0 {
				counts["7d"].GreenCount++
			}
			hasData = true
		}
		if e.Change30d != nil {
			counts["30d"].TotalCount++
			if *e.Change30d > 0 {
				counts["30d"].GreenCount++
			}
			hasData = true
		}

		if hasData {
			coinsWithData++
		}

		if e.ID == "bitcoin" && e.Change24h != nil {
			btcRef.PriceChange24h = *e.Change24h
			btcRef.Available = true
		}
	}

	tfCounts := make(map[string]TimeframeMetric, len(counts))
	for k, v := range counts {
		tfCounts[k] = *v
	}

	return CoinMarketsBreadthData{
		TimeframeCounts: tfCounts,
		CoinCount:       len(entries),
		CoinsWithData:   coinsWithData,
		BTCReference:    btcRef,
	}, nil
}

// CoinMarketsMomentumData holds per‑coin data needed for momentum divergence
// computation. All slices are parallel — one entry per coin, ordered by market
// cap descending.
type CoinMarketsMomentumData struct {
	PriceChanges []float64 // 24h price change % (0 if null from API)
	Volumes      []float64 // 24h trading volume in USD
	MarketCaps   []float64 // market cap in USD
}

// CoinMarketsMomentumEntry is a single coin entry from /coins/markets with
// 24h price change, volume, and market cap for momentum divergence calculation.
type CoinMarketsMomentumEntry struct {
	ID        string   `json:"id"`
	Symbol    string   `json:"symbol"`
	Change24h *float64 `json:"price_change_percentage_24h_in_currency"`
	Volume    float64  `json:"total_volume"`
	MarketCap float64  `json:"market_cap"`
}

// CoinMarketsMomentumURL returns the URL for the CoinGecko /coins/markets
// endpoint configured for momentum divergence. Fetches the top N coins
// with 24h price change percentage.
func CoinMarketsMomentumURL(perPage int) string {
	return fmt.Sprintf(
		"%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&page=1&sparkline=false&price_change_percentage=24h",
		BaseURL, perPage,
	)
}

// ParseCoinMarketsMomentumResponse parses the CoinGecko /coins/markets response
// and returns CoinMarketsMomentumData with price changes, volumes, and market caps.
//
// Null price change values are treated as 0 (conservative).
func ParseCoinMarketsMomentumResponse(body []byte) (CoinMarketsMomentumData, error) {
	if len(body) == 0 {
		return CoinMarketsMomentumData{}, fmt.Errorf("empty response body")
	}

	var entries []CoinMarketsMomentumEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return CoinMarketsMomentumData{}, fmt.Errorf("parsing coin markets momentum response: %w", err)
	}
	if len(entries) == 0 {
		return CoinMarketsMomentumData{}, fmt.Errorf("no coins in response")
	}

	changes := make([]float64, len(entries))
	volumes := make([]float64, len(entries))
	marketCaps := make([]float64, len(entries))

	for i, e := range entries {
		if e.Change24h != nil {
			changes[i] = *e.Change24h
		}
		volumes[i] = e.Volume
		marketCaps[i] = e.MarketCap
	}

	return CoinMarketsMomentumData{
		PriceChanges: changes,
		Volumes:      volumes,
		MarketCaps:   marketCaps,
	}, nil
}

// TopGainerEntry is a single coin entry for top gainers extraction.
type TopGainerEntry struct {
	Symbol    string   `json:"symbol"`
	Change24h *float64 `json:"price_change_percentage_24h_in_currency"`
}

// ParseTopGainers extracts the top N gainers from a CoinGecko coin markets response.
// Returns gainers sorted by 24h change descending.
func ParseTopGainers(body []byte, n int) ([]TopGainerEntry, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	var entries []TopGainerEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing coin markets for gainers: %w", err)
	}

	gainers := make([]TopGainerEntry, 0, len(entries))
	for _, e := range entries {
		change := 0.0
		if e.Change24h != nil {
			change = *e.Change24h
		}
		gainers = append(gainers, TopGainerEntry{
			Symbol:    strings.ToUpper(e.Symbol),
			Change24h: &change,
		})
	}

	sort.Slice(gainers, func(i, j int) bool {
		return *gainers[i].Change24h > *gainers[j].Change24h
	})

	if len(gainers) > n {
		gainers = gainers[:n]
	}

	return gainers, nil
}

// median returns the median value of a float64 slice.
// For even‑length slices, returns the average of the two middle values.
// Returns 0 for empty slices.
func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	n := len(vals)
	if n%2 == 1 {
		return vals[n/2]
	}
	return (vals[n/2-1] + vals[n/2]) / 2
}
