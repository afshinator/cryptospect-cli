package main

// catalog triggers init() registration for all built-in metric providers.
import (
	_ "github.com/afshinator/cryptospect-cli/internal/metrics/flowtension/v1"
	_ "github.com/afshinator/cryptospect-cli/internal/metrics/liquiditypulse/v1"
	_ "github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1"
	_ "github.com/afshinator/cryptospect-cli/internal/metrics/marketregime/v1"
	_ "github.com/afshinator/cryptospect-cli/internal/metrics/momentumdivergence/v1"
	_ "github.com/afshinator/cryptospect-cli/internal/metrics/stablecoinpower/v1"
)
