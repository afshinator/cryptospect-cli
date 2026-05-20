// Package dbnomics provides URL builders and response parsers for the
// DBnomics API (https://api.db.nomics.world/).
//
// Free, keyless REST API. Used for China M2 money supply data from the
// National Bureau of Statistics of China.
package dbnomics

import (
	"encoding/json"
	"fmt"
)

// BaseURL is the DBnomics API v22 base URL.
const BaseURL = "https://api.db.nomics.world/v22"

// ChinaM2Observation holds a single China M2 data point.
type ChinaM2Observation struct {
	Period string  `json:"period"`
	Value  float64 `json:"value"`
}

// ChinaM2Data holds the parsed China M2 series data.
type ChinaM2Data struct {
	Observations []ChinaM2Observation
	SeriesName   string
	Units        string
}

// ChinaM2URL returns the URL for fetching China M2 data.
func ChinaM2URL() string {
	return BaseURL + "/series/NBS/M_A0D01/A0D0101?observations=true&limit=13"
}

// ParseChinaM2Response parses the DBnomics China M2 response.
func ParseChinaM2Response(body []byte) (ChinaM2Data, error) {
	if len(body) == 0 {
		return ChinaM2Data{}, fmt.Errorf("empty response body")
	}

	var resp struct {
		Series struct {
			Docs []struct {
				Period     []string  `json:"period"`
				Value      []float64 `json:"value"`
				SeriesName string    `json:"series_name"`
				Dimensions struct {
					Unit string `json:"unit"`
				} `json:"dimensions"`
			} `json:"docs"`
		} `json:"series"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return ChinaM2Data{}, fmt.Errorf("parsing china m2 response: %w", err)
	}

	if len(resp.Series.Docs) == 0 {
		return ChinaM2Data{}, fmt.Errorf("no series docs in response")
	}

	doc := resp.Series.Docs[0]
	if len(doc.Period) == 0 {
		return ChinaM2Data{}, fmt.Errorf("no periods in series")
	}

	obs := make([]ChinaM2Observation, len(doc.Period))
	for i := range doc.Period {
		obs[i] = ChinaM2Observation{
			Period: doc.Period[i],
			Value:  doc.Value[i],
		}
	}

	return ChinaM2Data{
		Observations: obs,
		SeriesName:   doc.SeriesName,
		Units:        doc.Dimensions.Unit,
	}, nil
}
