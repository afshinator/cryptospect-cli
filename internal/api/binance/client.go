// Package binance provides URL builders and response parsers for the
// Binance US public spot API (https://api.binance.us/api/v3).
//
// Only keyless public endpoints are used here. Binance US offers spot data
// only — there is no futures API on this domain. See docs/metrics/flow-tension.md
// for notes on futures data (OI, funding rate) which require a separate key.
package binance

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// BaseURL is the Binance US public spot API base URL.
const BaseURL = "https://api.binance.us/api/v3"

// KlinesData holds the parsed fields from a single kline (candlestick) relevant
// to the flow_tension CVD proxy calculation.
type KlinesData struct {
	TotalVolume     float64 // base asset volume for the interval (index 5)
	TakerBuyVolume  float64 // taker buy base asset volume (index 9)
	TakerSellVolume float64 // derived: TotalVolume - TakerBuyVolume
	NumTrades       int     // number of trades in the interval (index 8)
}

// KlinesURL returns the URL to fetch the most recent N klines for a symbol and interval.
// Typical call for flow_tension: KlinesURL("BTCUSDT", "1h", 1).
func KlinesURL(symbol, interval string, limit int) string {
	return fmt.Sprintf("%s/klines?symbol=%s&interval=%s&limit=%d", BaseURL, symbol, interval, limit)
}

// ParseKlinesResponse parses the Binance klines JSON response and returns the
// fields needed for the CVD proxy. Expects at least one kline in the response.
//
// Binance klines are arrays-of-arrays. Each inner array has 12 elements:
//
//	[0]  openTime           (int64 ms)
//	[1]  open               (string float)
//	[2]  high               (string float)
//	[3]  low                (string float)
//	[4]  close              (string float)
//	[5]  volume             (string float) ← TotalVolume
//	[6]  closeTime          (int64 ms)
//	[7]  quoteAssetVolume   (string float)
//	[8]  numberOfTrades     (int)            ← NumTrades
//	[9]  takerBuyBaseVol    (string float) ← TakerBuyVolume
//	[10] takerBuyQuoteVol   (string float)
//	[11] ignore             (string)
func ParseKlinesResponse(body []byte) (KlinesData, error) {
	if len(body) == 0 {
		return KlinesData{}, fmt.Errorf("empty response body")
	}

	// Raw decode into [][]json.RawMessage to handle the mixed-type inner arrays.
	var raw [][]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return KlinesData{}, fmt.Errorf("parsing klines response: %w", err)
	}
	if len(raw) == 0 {
		return KlinesData{}, fmt.Errorf("klines response contains no data")
	}

	kline := raw[0]
	if len(kline) < 11 {
		return KlinesData{}, fmt.Errorf("kline has %d fields, expected at least 11", len(kline))
	}

	totalVolume, err := parseStringFloat(kline[5])
	if err != nil {
		return KlinesData{}, fmt.Errorf("parsing volume (index 5): %w", err)
	}

	numTrades, err := parseInt(kline[8])
	if err != nil {
		return KlinesData{}, fmt.Errorf("parsing numberOfTrades (index 8): %w", err)
	}

	takerBuyVol, err := parseStringFloat(kline[9])
	if err != nil {
		return KlinesData{}, fmt.Errorf("parsing takerBuyBaseVol (index 9): %w", err)
	}

	return KlinesData{
		TotalVolume:     totalVolume,
		TakerBuyVolume:  takerBuyVol,
		TakerSellVolume: totalVolume - takerBuyVol,
		NumTrades:       numTrades,
	}, nil
}

// parseInt parses a JSON number (unquoted integer) into an int.
// Binance encodes numberOfTrades as a bare integer (e.g. 48).
func parseInt(raw json.RawMessage) (int, error) {
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, fmt.Errorf("expected integer, got %s: %w", raw, err)
	}
	return n, nil
}

// parseStringFloat unquotes a JSON string and parses it as float64.
// Binance encodes numeric fields as quoted strings (e.g. "68114.37000000").
func parseStringFloat(raw json.RawMessage) (float64, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("expected quoted string, got %s: %w", raw, err)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %q as float: %w", s, err)
	}
	return f, nil
}

// KlinesVolumes holds the parsed volume data from multiple klines.
type KlinesVolumes struct {
	Volumes []float64 // consecutive volume readings, oldest first
}

// ParseKlinesVolumesResponse parses a multi‑kline Binance response and returns
// all volumes in chronological order. Used for volume exhaustion detection.
func ParseKlinesVolumesResponse(body []byte) (KlinesVolumes, error) {
	if len(body) == 0 {
		return KlinesVolumes{}, fmt.Errorf("empty response body")
	}

	var raw [][]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return KlinesVolumes{}, fmt.Errorf("parsing klines response: %w", err)
	}
	if len(raw) == 0 {
		return KlinesVolumes{}, fmt.Errorf("klines response contains no data")
	}

	volumes := make([]float64, 0, len(raw))
	for i, kline := range raw {
		if len(kline) < 6 {
			return KlinesVolumes{}, fmt.Errorf("kline %d has %d fields, expected at least 6", i, len(kline))
		}
		vol, err := parseStringFloat(kline[5])
		if err != nil {
			return KlinesVolumes{}, fmt.Errorf("parsing volume for kline %d: %w", i, err)
		}
		volumes = append(volumes, vol)
	}

	return KlinesVolumes{Volumes: volumes}, nil
}
