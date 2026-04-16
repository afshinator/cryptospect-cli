package output

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// stdout is the writer for JSON output. Defaults to os.Stdout.
var stdout io.Writer = os.Stdout

// WriteSuccess writes a successful CLIResponse envelope containing the given results.
func WriteSuccess(results []MetricResult) error {
	resp := CLIResponse{
		Status:  "ok",
		TS:      time.Now().Unix(),
		Results: results,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

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

	data, err2 := json.Marshal(resp)
	if err2 != nil {
		return err2
	}

	_, err2 = stdout.Write(data)
	return err2
}
