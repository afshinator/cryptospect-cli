package metrics

import (
	"encoding/json"
	"strconv"
)

// FloatPrecision is the precision for ratio fields (more decimal places).
const FloatPrecision = 4

// CurrencyPrecision is the precision for currency fields.
const CurrencyPrecision = 2

// MetricFloat is a float64 that serializes to JSON with configurable precision.
type MetricFloat struct {
	value     float64
	precision int
}

// NewMetricFloat creates a MetricFloat with specified precision.
func NewMetricFloat(v float64, precision int) MetricFloat {
	return MetricFloat{value: v, precision: precision}
}

// Value returns the underlying float64 value.
func (f MetricFloat) Value() float64 {
	return f.value
}

// MarshalJSON implements json.Marshaler.
func (f MetricFloat) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(f.value, 'f', f.precision, 64)), nil
}

// UnmarshalJSON implements json.Unmarshaler, accepting both numbers and strings.
func (f *MetricFloat) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		f.value = v
		return nil
	}
	var v float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	f.value = v
	return nil
}

// Ratio creates a MetricFloat with ratio precision (4 decimals).
func Ratio(v float64) MetricFloat {
	return MetricFloat{value: v, precision: FloatPrecision}
}

// Currency creates a MetricFloat with currency precision (2 decimals).
func Currency(v float64) MetricFloat {
	return MetricFloat{value: v, precision: CurrencyPrecision}
}
