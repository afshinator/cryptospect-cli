// Package coindesk provides URL builders and response parsers for the CoinDesk API.
// This is a placeholder — actual implementation pending.
package coindesk

import "fmt"

// BaseURL is the CoinDesk API base URL (to be confirmed).
const BaseURL = "https://api.coindesk.com/v1"

// AssetTopListURL returns the URL for the asset top list endpoint.
// Not yet implemented.
func AssetTopListURL() string {
	// TODO: implement proper URL building
	return ""
}

// ParseAssetTopListResponse parses the CoinDesk asset top list response.
// Not yet implemented.
func ParseAssetTopListResponse(body []byte) (interface{}, error) {
	return nil, fmt.Errorf("coindesk client not implemented")
}
