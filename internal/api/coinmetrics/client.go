// Package coinmetrics provides URL builders and response parsers for the CoinMetrics Community API.
// This is a placeholder — actual implementation pending.
package coinmetrics

import "fmt"

// BaseURL is the CoinMetrics Community API base URL (to be confirmed).
const BaseURL = "https://api.coinmetrics.io/v4"

// CommunityURL returns the URL for the community endpoint.
// Not yet implemented.
func CommunityURL() string {
	// TODO: implement proper URL building
	return ""
}

// ParseCommunityResponse parses the CoinMetrics community response.
// Not yet implemented.
func ParseCommunityResponse(body []byte) (interface{}, error) {
	return nil, fmt.Errorf("coinmetrics client not implemented")
}
