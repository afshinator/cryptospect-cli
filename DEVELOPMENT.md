# Development

Build, lint, test, and contribution notes for `cryptospect-cli`.

## Requirements

- Go 1.25+ (see `.go-version` for exact patch)
- `golangci-lint` v2.11+ (linting only — not required to build)

## Build Commands

```bash
make build    # compile to bin/cryptospect-cli
make fmt      # format with goimports + gofumpt
make lint     # run golangci-lint v2
make vet      # run go vet
make test     # run tests with race detector and coverage
make clean    # remove build artifacts
```

### `make build` vs `go build`

`make build` produces a stripped static binary with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` and injects a version string from the nearest git tag via ldflags.

`go build -o ./cryptospect-cli ./cmd/cryptospect-cli/` is faster for local development — it skips cross-compilation and produces a debug binary.

### Version Strings

| Build method | Example output |
|---|---|
| `make build` with a git tag | `v1.0.0` |
| `make build` without a tag | `v1.0.0 (0fedbf2-dirty)` |
| `go build` (dev) | `v1.0.0 (0fedbf2-dirty)` |

The source default is always `v1.0.0`. `make build` overrides it via ldflags when a tag exists.

> No git tags exist yet — both `make build` and `go build` currently produce the dev format.

## API Keys — Implementation Status

| Key | Env var | Status |
|-----|---------|--------|
| CoinGecko | `CRYPTOSPECT_COINGECKO_KEY` | Implemented — appended as `x_cg_demo_api_key` on the `/derivatives` endpoint |
| Binance | `CRYPTOSPECT_BINANCE_KEY` | Reserved — read from config but not yet passed to any request (only Binance US spot is implemented) |

## Cache TTLs

Default TTLs per data source:

| Source | Endpoint | TTL |
|--------|----------|-----|
| CoinGecko | Global market, stablecoins | 300 s |
| CoinGecko | Coin markets (breadth, momentum) | 300 s |
| Binance US | Spot klines (CVD) | 60 s |
| DefiLlama | Stablecoins | 300 s |

Override any TTL in `~/.cryptospect.yaml` under `cache.ttl`.

## Releasing

Automated binary releases use GoReleaser with GitHub Actions.

### Cut a release

```bash
git tag v1.0.0 && git push origin v1.0.0
```

This triggers `.github/workflows/release.yml`: runs tests, then cross-compiles 5 targets (linux/amd64, linux/arm64, windows/amd64, darwin/amd64, darwin/arm64), packages them as archives, and attaches them to the GitHub Release.

### Verify before pushing a tag

Run a local snapshot build to check binaries, archive names, and checksums without publishing:

```bash
goreleaser release --snapshot --clean
# inspect dist/ for artifacts
```

## License

MIT — see [LICENSE](LICENSE).