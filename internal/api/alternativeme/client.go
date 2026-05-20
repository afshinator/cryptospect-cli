// Package alternativeme provides URL builders and response parsers for the
// alternative.me Fear & Greed Index API (https://api.alternative.me/).
//
// Free, keyless REST API. Rate limit: 10 requests/minute.
package alternativeme

import (
	"encoding/json"
	"fmt"
)

// BaseURL is the alternative.me API base URL.
const BaseURL = "https://api.alternative.me"

// FNGData holds a single Fear & Greed Index data point.
type FNGData struct {
	Value               int    `json:"value,string"`
	ValueClassification string `json:"value_classification"`
	Timestamp           string `json:"timestamp"`
	TimeUntilUpdate     string `json:"time_until_update"`
}

// FNGResponse is the top-level API response for /fng/.
type FNGResponse struct {
	Name     string    `json:"name"`
	Data     []FNGData `json:"data"`
	Metadata struct {
		Error *string `json:"error"`
	} `json:"metadata"`
}

// FNGURL returns the URL for fetching Fear & Greed Index data.
// limit: number of data points to return (e.g., 1 for current, 7 for MA).
func FNGURL(limit int) string {
	return fmt.Sprintf("%s/fng/?limit=%d", BaseURL, limit)
}

// ParseFNGResponse parses the alternative.me /fng/ response.
func ParseFNGResponse(body []byte) ([]FNGData, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	var resp FNGResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing fng response: %w", err)
	}

	if resp.Metadata.Error != nil {
		return nil, fmt.Errorf("API error: %s", *resp.Metadata.Error)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no data points in response")
	}

	return resp.Data, nil
}
