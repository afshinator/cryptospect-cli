package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New(3)
	if c == nil {
		t.Fatal("New returned nil")
	}
}

func TestGetSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"data":"ok"}`))
	}))
	defer server.Close()

	c := New(0)
	body, err := c.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(body) != `{"data":"ok"}` {
		t.Errorf("body = %q, want %q", body, `{"data":"ok"}`)
	}
}

func TestGet404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`not found`))
	}))
	defer server.Close()

	c := New(2)
	_, err := c.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("Get should fail on 404")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %v, want 404", apiErr.StatusCode)
	}
	if apiErr.Body != "not found" {
		t.Errorf("Body = %q, want 'not found'", apiErr.Body)
	}
}

func TestGet429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(429)
		w.Write([]byte(`rate limited`))
	}))
	defer server.Close()

	c := New(0)
	_, err := c.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("Get should fail on 429")
	}
	rateErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("error type = %T, want *RateLimitError", err)
	}
	if rateErr.StatusCode != 429 {
		t.Errorf("StatusCode = %v, want 429", rateErr.StatusCode)
	}
	if rateErr.RetryAfter != 60*time.Second {
		t.Errorf("RetryAfter = %v, want 60s", rateErr.RetryAfter)
	}
}

func TestGet500WithRetry(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(500)
			w.Write([]byte(`internal error`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`success`))
	}))
	defer server.Close()

	c := New(3) // max retries = 3, so total attempts = 4
	body, err := c.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get failed after retries: %v", err)
	}
	if attempt != 3 {
		t.Errorf("attempts = %v, want 3", attempt)
	}
	if string(body) != "success" {
		t.Errorf("body = %q, want 'success'", body)
	}
}

func TestGetNetworkErrorRetry(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 2 {
			// Simulate network error by closing connection
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	c := New(2)
	body, err := c.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get failed after retry: %v", err)
	}
	if attempt != 2 {
		t.Errorf("attempts = %v, want 2", attempt)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want 'ok'", body)
	}
}

func TestGetExhaustRetries(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(503)
		w.Write([]byte(`service unavailable`))
	}))
	defer server.Close()

	c := New(2) // will try 3 times total
	_, err := c.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("Get should fail after exhausting retries")
	}
	if attempt != 3 {
		t.Errorf("attempts = %v, want 3", attempt)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 503 {
		t.Errorf("StatusCode = %v, want 503", apiErr.StatusCode)
	}
}

func TestGetWithKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("x_cg_demo_api_key") == "test-key" {
			w.WriteHeader(200)
			w.Write([]byte(`with key`))
		} else {
			w.WriteHeader(200)
			w.Write([]byte(`without key`))
		}
	}))
	defer server.Close()

	c := New(0)
	// With key
	body, err := c.GetWithKey(context.Background(), server.URL, "test-key")
	if err != nil {
		t.Fatalf("GetWithKey failed: %v", err)
	}
	if string(body) != "with key" {
		t.Errorf("body with key = %q, want 'with key'", body)
	}
	// Without key
	body, err = c.GetWithKey(context.Background(), server.URL, "")
	if err != nil {
		t.Fatalf("GetWithKey empty key failed: %v", err)
	}
	if string(body) != "without key" {
		t.Errorf("body empty key = %q, want 'without key'", body)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
	}{
		{"", 0},
		{"60", 60 * time.Second},
		{"120", 120 * time.Second},
		{"invalid", 0},
		{"Wed, 21 Oct 2015 07:28:00 GMT", 0}, // past date -> 0
	}
	for _, tt := range tests {
		got := parseRetryAfter(tt.header)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestBackoffDuration(t *testing.T) {
	// Just ensure it returns positive durations
	for attempt := 1; attempt <= 5; attempt++ {
		d := backoffDuration(attempt)
		if d <= 0 {
			t.Errorf("backoffDuration(%v) = %v, want positive", attempt, d)
		}
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Endpoint:   "https://api.example.com/data",
		Body:       "Not found",
	}
	msg := err.Error()
	expected := "https://api.example.com/data: HTTP 404: Not found"
	if msg != expected {
		t.Errorf("Error() = %q, want %q", msg, expected)
	}
}
