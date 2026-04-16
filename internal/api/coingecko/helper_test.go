package coingecko

// abs returns the absolute value of a float64.
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
