# E2E Tests for New Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CLI-level end-to-end tests for dominance, volatility, fear-greed-index, and china-m2 metrics consistent with existing metric E2E patterns.

**Architecture:** Mirror existing Cobra-driven E2E tests using httptest servers to mock upstream APIs, execute root command, and assert JSON envelope + key fields without brittle numeric checks.

**Tech Stack:** Go testing (stdlib), httptest, Cobra CLI, existing internal output structs

---

## File Structure

- Create: `cmd/cryptospect-cli/dominance_e2e_test.go`
- Create: `cmd/cryptospect-cli/volatility_e2e_test.go`
- Create: `cmd/cryptospect-cli/fear_greed_index_e2e_test.go`
- Create: `cmd/cryptospect-cli/china_m2_e2e_test.go`

Each file contains:
- 1 happy-path E2E test
- 1 degraded/unavailable-path test

---

### Task 1: Dominance E2E Test

**Files:**
- Create: `cmd/cryptospect-cli/dominance_e2e_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestDominance_E2E(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "/global") {
            w.Header().Set("Content-Type", "application/json")
            w.Write([]byte(`{"data":{"market_cap_percentage":{"btc":52.3,"eth":18.1}}}`))
            return
        }
        http.NotFound(w, r)
    }))
    defer srv.Close()

    os.Setenv("COINGECKO_BASE_URL", srv.URL)

    buf := new(bytes.Buffer)
    rootCmd := NewRootCmd()
    rootCmd.SetOut(buf)
    rootCmd.SetArgs([]string{"dom", "--detail", "full"})

    err := rootCmd.Execute()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var resp output.CLIResponse
    if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
        t.Fatalf("invalid json: %v", err)
    }

    if len(resp.Results) != 1 {
        t.Fatalf("expected 1 result")
    }

    res := resp.Results[0]

    if res.Metric != "dominance" {
        t.Fatalf("wrong metric")
    }

    if res.Status == "unavailable" {
        t.Fatalf("unexpected unavailable")
    }

    if res.Meta == nil {
        t.Fatalf("expected meta")
    }
}
```

- [ ] **Step 2: Run test (expect fail)**

Run: `go test ./cmd/cryptospect-cli -run TestDominance_E2E -v`

- [ ] **Step 3: Fix env wiring if needed**

Adjust env variable name to match existing CoinGecko override pattern used in other E2E tests.

- [ ] **Step 4: Run test (expect pass)**

Run same command

- [ ] **Step 5: Commit**

```bash
git add cmd/cryptospect-cli/dominance_e2e_test.go
git commit -m "test: add dominance e2e test"
```

---

### Task 2: Volatility E2E Test

**Files:**
- Create: `cmd/cryptospect-cli/volatility_e2e_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestVolatility_E2E(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "/klines") {
            w.Header().Set("Content-Type", "application/json")
            w.Write([]byte(`[[1,100,110,90,105,1000],[2,105,115,95,110,1200]]`))
            return
        }
        http.NotFound(w, r)
    }))
    defer srv.Close()

    os.Setenv("BINANCE_BASE_URL", srv.URL)

    buf := new(bytes.Buffer)
    cmd := NewRootCmd()
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"vol"})

    if err := cmd.Execute(); err != nil {
        t.Fatalf("error: %v", err)
    }

    var resp output.CLIResponse
    if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
        t.Fatalf("invalid json")
    }

    if resp.Results[0].Metric != "volatility" {
        t.Fatalf("wrong metric")
    }
}
```

- [ ] **Step 2: Run test (expect fail)**

Run: `go test ./cmd/cryptospect-cli -run TestVolatility_E2E -v`

- [ ] **Step 3: Fix base URL override to match repo pattern**

- [ ] **Step 4: Run test (expect pass)**

- [ ] **Step 5: Commit**

```bash
git add cmd/cryptospect-cli/volatility_e2e_test.go
git commit -m "test: add volatility e2e test"
```

---

### Task 3: Fear & Greed Index E2E Test

**Files:**
- Create: `cmd/cryptospect-cli/fear_greed_index_e2e_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestFGI_E2E(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"data":[{"value":"60","value_classification":"Greed"}]}`))
    }))
    defer srv.Close()

    os.Setenv("ALTERNATIVEME_BASE_URL", srv.URL)

    buf := new(bytes.Buffer)
    cmd := NewRootCmd()
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"fgi"})

    if err := cmd.Execute(); err != nil {
        t.Fatalf("error: %v", err)
    }

    var resp output.CLIResponse
    json.Unmarshal(buf.Bytes(), &resp)

    if resp.Results[0].Metric != "fear-greed-index" {
        t.Fatalf("wrong metric")
    }
}
```

- [ ] **Step 2: Run test (expect fail)**

- [ ] **Step 3: Adjust base URL override if needed**

- [ ] **Step 4: Run test (expect pass)**

- [ ] **Step 5: Commit**

```bash
git add cmd/cryptospect-cli/fear_greed_index_e2e_test.go
git commit -m "test: add fgi e2e test"
```

---

### Task 4: China M2 E2E Test

**Files:**
- Create: `cmd/cryptospect-cli/china_m2_e2e_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestChinaM2_E2E(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"series":{"docs":[{"period":"2026-01","value":3000000}]}}`))
    }))
    defer srv.Close()

    os.Setenv("DBNOMICS_BASE_URL", srv.URL)

    buf := new(bytes.Buffer)
    cmd := NewRootCmd()
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"cnm2"})

    if err := cmd.Execute(); err != nil {
        t.Fatalf("error: %v", err)
    }

    var resp output.CLIResponse
    json.Unmarshal(buf.Bytes(), &resp)

    if resp.Results[0].Metric != "china-m2" {
        t.Fatalf("wrong metric")
    }
}
```

- [ ] **Step 2: Run test (expect fail)**

- [ ] **Step 3: Adjust DBnomics mock shape to match parser expectations**

- [ ] **Step 4: Run test (expect pass)**

- [ ] **Step 5: Commit**

```bash
git add cmd/cryptospect-cli/china_m2_e2e_test.go
git commit -m "test: add china m2 e2e test"
```

---

## Final Verification

- [ ] Run full suite

```bash
go test ./... -race
```

Expected: all tests pass

---

## Notes

- Always match existing E2E env override patterns (critical)
- Avoid asserting exact numeric outputs
- Focus on envelope, status, and presence of fields
- Ensure JSON always parses (core contract)
