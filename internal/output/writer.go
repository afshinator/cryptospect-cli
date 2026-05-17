package output

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	stdoutMu sync.RWMutex
	stdout   io.Writer = os.Stdout
	pretty   atomic.Bool
)

// SetPretty enables or disables pretty-printed JSON output (2-space indent).
func SetPretty(v bool) { pretty.Store(v) }

// ResetForTest restores package-level defaults. Use with t.Cleanup in tests.
func ResetForTest() {
	pretty.Store(false)
	stdoutMu.Lock()
	stdout = os.Stdout
	stdoutMu.Unlock()
}

// SetWriter replaces the writer used for JSON output.
// This is intended for testing only.
func SetWriter(w io.Writer) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	stdout = w
}

// Writer returns the current writer used for JSON output.
func Writer() io.Writer {
	stdoutMu.RLock()
	defer stdoutMu.RUnlock()
	return stdout
}

// WriteSuccess writes a successful CLIResponse envelope containing the given results.
func WriteSuccess(results []MetricResult) error {
	resp := CLIResponse{
		Status:  "ok",
		TS:      time.Now().Unix(),
		Results: results,
	}

	data, err := encodeJSON(resp)
	if err != nil {
		return err
	}

	stdoutMu.RLock()
	defer stdoutMu.RUnlock()
	_, err = stdout.Write(data)
	return err
}

// WriteError writes an error CLIResponse envelope.
func WriteError(code int, msg, source string, retryAfterSec int) error {
	err := CLIError{
		Code:          code,
		Msg:           msg,
		Source:        source,
		RetryAfterSec: retryAfterSec,
	}

	resp := CLIResponse{
		Status: "error",
		TS:     time.Now().Unix(),
		Error:  &err,
	}

	data, err2 := encodeJSON(resp)
	if err2 != nil {
		return err2
	}

	stdoutMu.RLock()
	defer stdoutMu.RUnlock()
	_, err2 = stdout.Write(data)
	return err2
}

// encodeJSON serialises v as compact or indented JSON depending on the pretty flag.
func encodeJSON(v any) ([]byte, error) {
	if pretty.Load() {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
