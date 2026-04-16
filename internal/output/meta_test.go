package output

import (
	"encoding/json"
	"testing"
)

func TestMetaBasic_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		meta MetaBasic
		want string
	}{
		{
			name: "cache hit with ttl",
			meta: MetaBasic{
				CacheHit:     true,
				TTLRemaining: 300,
			},
			want: `{"cache_hit":true,"ttl_remaining":300}`,
		},
		{
			name: "cache miss",
			meta: MetaBasic{
				CacheHit:     false,
				TTLRemaining: 0,
			},
			want: `{"cache_hit":false,"ttl_remaining":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.meta)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var gotObj, wantObj map[string]interface{}
			if err := json.Unmarshal(got, &gotObj); err != nil {
				t.Fatalf("Unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
				t.Fatalf("Unmarshal want: %v", err)
			}

			if !mapsEqual(gotObj, wantObj) {
				t.Errorf("got %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestSourceMeta_MarshalJSON(t *testing.T) {
	meta := SourceMeta{
		Endpoint:  "coingecko.global",
		Timestamp: 1744444800,
		CacheHit:  true,
	}

	got, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	want := `{"endpoint":"coingecko.global","timestamp":1744444800,"cache_hit":true}`

	var gotObj, wantObj map[string]interface{}
	if err := json.Unmarshal(got, &gotObj); err != nil {
		t.Fatalf("Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantObj); err != nil {
		t.Fatalf("Unmarshal want: %v", err)
	}

	if !mapsEqual(gotObj, wantObj) {
		t.Errorf("got %s, want %s", string(got), want)
	}
}

func TestMetaExtended_MarshalJSON(t *testing.T) {
	meta := MetaExtended{
		MetaBasic: MetaBasic{
			CacheHit:     true,
			TTLRemaining: 300,
		},
		Sources: map[string]SourceMeta{
			"global_data": {
				Endpoint:  "coingecko.global",
				Timestamp: 1744444800,
				CacheHit:  true,
			},
		},
	}

	got, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	want := `{"cache_hit":true,"ttl_remaining":300,"sources":{"global_data":{"endpoint":"coingecko.global","timestamp":1744444800,"cache_hit":true}}}`

	var gotObj, wantObj map[string]interface{}
	if err := json.Unmarshal(got, &gotObj); err != nil {
		t.Fatalf("Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantObj); err != nil {
		t.Fatalf("Unmarshal want: %v", err)
	}

	if !mapsEqual(gotObj, wantObj) {
		t.Errorf("got %s, want %s", string(got), want)
	}
}

func TestMetaFull_MarshalJSON(t *testing.T) {
	meta := MetaFull{
		MetaExtended: MetaExtended{
			MetaBasic: MetaBasic{
				CacheHit:     true,
				TTLRemaining: 300,
			},
			Sources: map[string]SourceMeta{
				"global_data": {
					Endpoint:  "coingecko.global",
					Timestamp: 1744444800,
					CacheHit:  true,
				},
			},
		},
		Thresholds: map[string]float64{
			"high": 0.8,
			"low":  0.2,
		},
		Description: "Liquidity pulse measures global 24h volume relative to market cap.",
	}

	got, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	want := `{"cache_hit":true,"ttl_remaining":300,"sources":{"global_data":{"endpoint":"coingecko.global","timestamp":1744444800,"cache_hit":true}},"thresholds":{"high":0.8,"low":0.2},"description":"Liquidity pulse measures global 24h volume relative to market cap."}`

	var gotObj, wantObj map[string]interface{}
	if err := json.Unmarshal(got, &gotObj); err != nil {
		t.Fatalf("Unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantObj); err != nil {
		t.Fatalf("Unmarshal want: %v", err)
	}

	if !mapsEqual(gotObj, wantObj) {
		t.Errorf("got %s, want %s", string(got), want)
	}
}
