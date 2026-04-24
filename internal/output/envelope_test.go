package output

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCLIResponse_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		response CLIResponse
		wantKeys []string
	}{
		{
			name: "success response with one metric result",
			response: CLIResponse{
				Status: "ok",
				TS:     1744444800,
				Results: []MetricResult{
					{
						Metric: "liquidity-pulse",
						Status: "ok",
						Data:   json.RawMessage(`{"score":0.85}`),
					},
				},
			},
			wantKeys: []string{"status", "ts", "results"},
		},
		{
			name: "error response",
			response: CLIResponse{
				Status: "error",
				TS:     1744444800,
				Error: &CLIError{
					Code:          429,
					Msg:           "rate_limited",
					RetryAfterSec: 60,
					Source:        "coingecko",
				},
			},
			wantKeys: []string{"status", "ts", "error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var obj map[string]interface{}
			if err := json.Unmarshal(got, &obj); err != nil {
				t.Fatalf("Unmarshal got: %v", err)
			}

			// Check expected keys are present
			for _, key := range tt.wantKeys {
				if _, ok := obj[key]; !ok {
					t.Errorf("key %q missing from marshaled JSON", key)
				}
			}

			// Verify status field matches
			if status, ok := obj["status"].(string); !ok || status != tt.response.Status {
				t.Errorf("status = %v, want %v", status, tt.response.Status)
			}

			// Verify ts field matches (JSON numbers become float64)
			if ts, ok := obj["ts"].(float64); !ok || int64(ts) != tt.response.TS {
				t.Errorf("ts = %v, want %v", ts, tt.response.TS)
			}
		})
	}
}

func TestCLIResponse_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    CLIResponse
	}{
		{
			name:    "success response",
			jsonStr: `{"status":"ok","ts":1744444800,"results":[{"metric":"liquidity-pulse","status":"ok","data":{"score":0.85}}]}`,
			want: CLIResponse{
				Status: "ok",
				TS:     1744444800,
				Results: []MetricResult{
					{
						Metric: "liquidity-pulse",
						Status: "ok",
						Data:   json.RawMessage(`{"score":0.85}`),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got CLIResponse
			err := json.Unmarshal([]byte(tt.jsonStr), &got)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.TS != tt.want.TS {
				t.Errorf("TS = %v, want %v", got.TS, tt.want.TS)
			}
			if len(got.Results) != len(tt.want.Results) {
				t.Errorf("Results length = %v, want %v", len(got.Results), len(tt.want.Results))
			}
			if len(got.Results) > 0 {
				if got.Results[0].Metric != tt.want.Results[0].Metric {
					t.Errorf("Metric = %v, want %v", got.Results[0].Metric, tt.want.Results[0].Metric)
				}
				// Compare Data as strings (json.RawMessage is []byte)
				if string(got.Results[0].Data) != string(tt.want.Results[0].Data) {
					t.Errorf("Data = %s, want %s", got.Results[0].Data, tt.want.Results[0].Data)
				}
			}
		})
	}
}

func TestMetricResult_MarshalJSON(t *testing.T) {
tests := []struct {
		name   string
		result MetricResult
		want   string
	}{
		{
			name: "without meta",
			result: MetricResult{
				Metric:  "liquidity-pulse",
				Version: "v1.0.0",
				Status:  "ok",
				Data:    json.RawMessage(`{"score":0.85}`),
			},
			want: `{"metric":"liquidity-pulse","version":"v1.0.0","status":"ok","data":{"score":0.85}}`,
		},
		{
			name: "with meta",
			result: MetricResult{
				Metric:  "market-breadth",
				Version: "v1.0.0",
				Status:  "degraded",
				Data:    json.RawMessage(`{"participation":0.65}`),
				Meta:    json.RawMessage(`{"cache_hit":true}`),
			},
			want: `{"metric":"market-breadth","version":"v1.0.0","status":"degraded","data":{"participation":0.65},"meta":{"cache_hit":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.result)
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

			// Compare objects
			if !mapsEqual(gotObj, wantObj) {
				t.Errorf("got %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestCLIError_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		err  CLIError
		want string
	}{
		{
			name: "full error",
			err: CLIError{
				Code:          429,
				Msg:           "rate_limited",
				RetryAfterSec: 60,
				Source:        "coingecko",
			},
			want: `{"code":429,"msg":"rate_limited","retry_after_sec":60,"source":"coingecko"}`,
		},
		{
			name: "minimal error",
			err: CLIError{
				Code: 500,
				Msg:  "internal_error",
			},
			want: `{"code":500,"msg":"internal_error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.err)
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

// mapsEqual compares two map[string]interface{} using reflect.DeepEqual.
func mapsEqual(a, b map[string]interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func TestJSONFieldNames(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{
			name: "CLIResponse",
			typ:  reflect.TypeOf(CLIResponse{}),
		},
		{
			name: "MetricResult",
			typ:  reflect.TypeOf(MetricResult{}),
		},
		{
			name: "CLIError",
			typ:  reflect.TypeOf(CLIError{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.typ.NumField(); i++ {
				field := tt.typ.Field(i)
				tag := field.Tag.Get("json")
				if tag == "" {
					t.Errorf("%s.%s missing json tag", tt.name, field.Name)
					continue
				}

				// Parse tag (could have "omitempty")
				parts := strings.Split(tag, ",")
				jsonName := parts[0]

				// Check snake_case (lowercase with underscores)
				if jsonName == "-" {
					continue // intentionally omitted
				}
				for _, ch := range jsonName {
					if ch >= 'A' && ch <= 'Z' {
						t.Errorf("%s.%s json tag %q contains uppercase letters; use snake_case", tt.name, field.Name, jsonName)
						break
					}
				}
				// Ensure no hyphens (use underscores)
				if strings.Contains(jsonName, "-") {
					t.Errorf("%s.%s json tag %q contains hyphens; use underscores", tt.name, field.Name, jsonName)
				}
			}
		})
	}
}
