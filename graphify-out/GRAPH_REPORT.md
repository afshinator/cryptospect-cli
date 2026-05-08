# Graph Report - /project/cryptospect-cli  (2026-05-08)

## Corpus Check
- 91 files · ~85,378 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 862 nodes · 1378 edges · 74 communities (42 shown, 32 thin omitted)
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 376 edges (avg confidence: 0.8)
- Token cost: 16,800 input · 4,050 output

## Community Hubs (Navigation)
- [[_COMMUNITY_CLI Commands & E2E Tests|CLI Commands & E2E Tests]]
- [[_COMMUNITY_Market Metrics Compute Core|Market Metrics Compute Core]]
- [[_COMMUNITY_DefiLlama Stablecoin Supply|DefiLlama Stablecoin Supply]]
- [[_COMMUNITY_CLI Scaffold & Global Hooks|CLI Scaffold & Global Hooks]]
- [[_COMMUNITY_Configuration System|Configuration System]]
- [[_COMMUNITY_Project Documentation & Design|Project Documentation & Design]]
- [[_COMMUNITY_Momentum Divergence Tests|Momentum Divergence Tests]]
- [[_COMMUNITY_MetricFloat & Provider Tests|MetricFloat & Provider Tests]]
- [[_COMMUNITY_HTTP Client & Error Handling|HTTP Client & Error Handling]]
- [[_COMMUNITY_CoinGecko CoinMarkets API|CoinGecko CoinMarkets API]]
- [[_COMMUNITY_Binance Klines API|Binance Klines API]]
- [[_COMMUNITY_Metric Registry Tests|Metric Registry Tests]]
- [[_COMMUNITY_API Fetcher Caching|API Fetcher Caching]]
- [[_COMMUNITY_Catalog & Endpoint Registration|Catalog & Endpoint Registration]]
- [[_COMMUNITY_Metric Provider Implementations|Metric Provider Implementations]]
- [[_COMMUNITY_File Cache Layer|File Cache Layer]]
- [[_COMMUNITY_Registry Core Logic|Registry Core Logic]]
- [[_COMMUNITY_CoinGecko GlobalStables Parsers|CoinGecko Global/Stables Parsers]]
- [[_COMMUNITY_Stablecoin Power Provider Tests|Stablecoin Power Provider Tests]]
- [[_COMMUNITY_CoinGecko Data Structures|CoinGecko Data Structures]]
- [[_COMMUNITY_Market Breadth Provider Tests|Market Breadth Provider Tests]]
- [[_COMMUNITY_Market Breadth Compute Tests|Market Breadth Compute Tests]]
- [[_COMMUNITY_API Fetcher Core|API Fetcher Core]]
- [[_COMMUNITY_CoinGecko Derivatives API|CoinGecko Derivatives API]]
- [[_COMMUNITY_CoinGecko Shared Parsers|CoinGecko Shared Parsers]]
- [[_COMMUNITY_Output EnvelopeMeta Tests|Output Envelope/Meta Tests]]
- [[_COMMUNITY_API URL Resolution|API URL Resolution]]
- [[_COMMUNITY_Flow Tension Compute|Flow Tension Compute]]
- [[_COMMUNITY_Binance API Client|Binance API Client]]
- [[_COMMUNITY_Output Envelope Types|Output Envelope Types]]
- [[_COMMUNITY_MetricFloat Serialization|MetricFloat Serialization]]
- [[_COMMUNITY_Output Metadata|Output Metadata]]
- [[_COMMUNITY_API Concurrent Fetching|API Concurrent Fetching]]
- [[_COMMUNITY_Go 1.25 Caching Patterns|Go 1.25 Caching Patterns]]
- [[_COMMUNITY_MetricProvider Interface|MetricProvider Interface]]
- [[_COMMUNITY_Config Loader|Config Loader]]
- [[_COMMUNITY_CoinMetrics Package|CoinMetrics Package]]
- [[_COMMUNITY_Flow Tension & MD Constants|Flow Tension & MD Constants]]
- [[_COMMUNITY_DefiLlama Trend Classification|DefiLlama Trend Classification]]
- [[_COMMUNITY_Config Context Operations|Config Context Operations]]
- [[_COMMUNITY_Config Detail Context|Config Detail Context]]
- [[_COMMUNITY_Config TopN Context|Config TopN Context]]
- [[_COMMUNITY_Config Segments Context|Config Segments Context]]
- [[_COMMUNITY_Cache Existence Check|Cache Existence Check]]
- [[_COMMUNITY_Market Breadth Constants|Market Breadth Constants]]
- [[_COMMUNITY_Liquidity Pulse Constants|Liquidity Pulse Constants]]
- [[_COMMUNITY_Stablecoin Power Constants|Stablecoin Power Constants]]
- [[_COMMUNITY_API Endpoints List|API Endpoints List]]
- [[_COMMUNITY_CoinGecko Dominance Parser|CoinGecko Dominance Parser]]
- [[_COMMUNITY_CoinGecko Stablecoin IDs|CoinGecko Stablecoin IDs]]
- [[_COMMUNITY_CoinGecko Top Gainers Parser|CoinGecko Top Gainers Parser]]
- [[_COMMUNITY_CoinDesk Client Instance|CoinDesk Client Instance]]
- [[_COMMUNITY_DefiLlama Base URL|DefiLlama Base URL]]
- [[_COMMUNITY_Output Writer Setter|Output Writer Setter]]
- [[_COMMUNITY_Config Write|Config Write]]
- [[_COMMUNITY_Config Source Resolution|Config Source Resolution]]
- [[_COMMUNITY_Config Cache Directory|Config Cache Directory]]
- [[_COMMUNITY_HTTP Client Package Node|HTTP Client Package Node]]
- [[_COMMUNITY_Binance Package Node|Binance Package Node]]
- [[_COMMUNITY_CoinMetrics Base URL|CoinMetrics Base URL]]
- [[_COMMUNITY_Registry Constructor|Registry Constructor]]
- [[_COMMUNITY_Status Detection Helper|Status Detection Helper]]
- [[_COMMUNITY_MetricFloat Value Accessor|MetricFloat Value Accessor]]
- [[_COMMUNITY_MetricFloat MarshalJSON|MetricFloat MarshalJSON]]
- [[_COMMUNITY_MetricFloat UnmarshalJSON|MetricFloat UnmarshalJSON]]
- [[_COMMUNITY_Version Variable|Version Variable]]
- [[_COMMUNITY_Root Command Instance|Root Command Instance]]

## God Nodes (most connected - your core abstractions)
1. `NewRootCommand()` - 44 edges
2. `NewRootCommand` - 41 edges
3. `SetWriter()` - 33 edges
4. `Writer()` - 33 edges
5. `c()` - 28 edges
6. `NewRegistry()` - 19 edges
7. `makeProvider()` - 17 edges
8. `Compute()` - 16 edges
9. `Open()` - 15 edges
10. `Cache` - 13 edges

## Surprising Connections (you probably didn't know these)
- `newCacheClearCommand()` --calls--> `Exists()`  [INFERRED]
  cmd/cryptospect-cli/cache.go → internal/cache/cache.go
- `newCacheClearCommand()` --calls--> `Open()`  [INFERRED]
  cmd/cryptospect-cli/cache.go → internal/cache/cache.go
- `newListCommand()` --calls--> `GlobalRegistry()`  [INFERRED]
  cmd/cryptospect-cli/list.go → internal/metrics/registry.go
- `NewRootCommand()` --calls--> `GlobalRegistry()`  [INFERRED]
  cmd/cryptospect-cli/root.go → internal/metrics/registry.go
- `TestCatalog_AllProvidersRegistered()` --calls--> `GlobalRegistry()`  [INFERRED]
  cmd/cryptospect-cli/catalog_test.go → internal/metrics/registry.go

## Hyperedges (group relationships)
- **Metric Provider Suite** — lp_provider, sp_provider, ft_provider, mb_provider, md_provider, mr_provider [EXTRACTED 1.00]
- **Binance Klines Validator Pattern** — lp_provider, mb_provider, ft_provider [INFERRED 0.85]
- **Duplicate confidenceToFloat Helper** — lp_confidencetofloat, sp_confidencetofloat, lp_provider, sp_provider [INFERRED 0.95]
- **Plugin Registry Pattern** —  [INFERRED 0.85]
- **Custom JSON Serialization Pattern** —  [INFERRED 0.85]
- **Metric Status Detection Pattern** —  [INFERRED 0.75]
- **MetricE2ETestPattern** — cryptospect_cli_test_liquidity_pulse_command, cryptospect_cli_test_stablecoin_power_command, cryptospect_cli_test_flow_tension_command, cryptospect_cli_test_market_breadth_command, cryptospect_cli_test_momentum_divergence_command [INFERRED 0.95]
- **MetricAliasTestPattern** — cryptospect_cli_test_liquidity_pulse_alias, cryptospect_cli_test_stablecoin_power_alias, cryptospect_cli_test_flow_tension_alias, cryptospect_cli_test_market_breadth_alias, cryptospect_cli_test_momentum_divergence_alias [INFERRED 0.95]
- **MetricDetailExtendedTestPattern** — cryptospect_cli_test_liquidity_pulse_detail_extended, cryptospect_cli_test_stablecoin_power_detail_extended, cryptospect_cli_test_flow_tension_detail_extended, cryptospect_cli_test_market_breadth_detail_extended, cryptospect_cli_test_momentum_divergence_detail_extended [INFERRED 0.95]
- **MetricDetailFullTestPattern** — cryptospect_cli_test_liquidity_pulse_detail_full, cryptospect_cli_test_stablecoin_power_detail_full, cryptospect_cli_test_flow_tension_detail_full, cryptospect_cli_test_market_breadth_detail_full, cryptospect_cli_test_momentum_divergence_detail_full [INFERRED 0.95]
- **CLIArchitecturePattern** — cryptospect_cli_main_func, cryptospect_cli_new_root_command, cryptospect_cli_build_metric_run_e, cryptospect_cli_new_list_command, cryptospect_cli_new_cache_clear_command, cryptospect_cli_persistent_pre_run_e [INFERRED 0.95]
- **CatalogTestSuite** — cryptospect_cli_test_catalog_all_registered, cryptospect_cli_test_catalog_aliases_resolvable, cryptospect_cli_test_catalog_endpoints_known [EXTRACTED 1.00]
- **TopFlagTestPattern** — cryptospect_cli_test_market_breadth_top_flag, cryptospect_cli_test_stablecoin_power_top_flag [INFERRED 0.85]
- **TopFlagClampedTestPattern** — cryptospect_cli_test_market_breadth_top_flag_clamped, cryptospect_cli_test_stablecoin_power_top_flag_clamped [INFERRED 0.85]

## Communities (74 total, 32 thin omitted)

### Community 0 - "CLI Commands & E2E Tests"
Cohesion: 0.08
Nodes (48): FromContext(), TestCacheClearCommand(), TestCacheClearOutputJSON(), newCacheClearCommand(), flagRegistrar, TestFlowTensionAlias(), TestFlowTensionCommand(), TestFlowTensionDetailExtended() (+40 more)

### Community 1 - "Market Metrics Compute Core"
Cohesion: 0.06
Nodes (52): buildSummary(), buildTierDetail(), Compute(), ComputeExchangeNetFlow(), ComputeFlowHook(), ComputeFundingRateHook(), ComputeOIChange24h(), ComputeOIChangeHook() (+44 more)

### Community 2 - "DefiLlama Stablecoin Supply"
Cohesion: 0.05
Nodes (43): AggregateSupply() — sums peggedUSD across all assets (current+prev), Circulating, AggregateSupply(), ClassifyTrend(), ParseStablecoinsResponse(), StablecoinsURL(), buildBody(), containsAt() (+35 more)

### Community 3 - "CLI Scaffold & Global Hooks"
Cohesion: 0.04
Nodes (48): buildMetricRunE, catalog.go (metric provider blank imports), defaultConfigPath, flagRegistrar interface, main (entry point), newCacheClearCommand (cache-clear), newListCommand (list-metrics), NewRootCommand (+40 more)

### Community 4 - "Configuration System"
Cohesion: 0.07
Nodes (40): APIsConfig struct (CoinGecko + Binance APIKeyConfig), APIKeyConfig, APIsConfig, CacheConfig, Config, defaults(), DetailFromContext(), Load() (+32 more)

### Community 5 - "Project Documentation & Design"
Cohesion: 0.07
Nodes (38): Agent orchestration playbook, Barbell modifier (neutral + tail_extension), CLIResponse JSON envelope, OI 24h change via cache (cold-start), Concentration dead band (±0.5% large_avg), cryptospect-cli, Detail level standard (basic/extended/full), DetectStatus helper (confidence/thin-data) (+30 more)

### Community 6 - "Momentum Divergence Tests"
Cohesion: 0.1
Nodes (35): c(), cn(), TestCompute_Confidence_High(), TestCompute_Confidence_Low(), TestCompute_CustomSegments_TighterLarge(), TestCompute_DeadBand_NegativeBoundary(), TestCompute_DeadBand_PositiveBoundary(), TestCompute_FlightToSafety() (+27 more)

### Community 7 - "MetricFloat & Provider Tests"
Cohesion: 0.1
Nodes (18): Currency(), Ratio(), MetricFloat, cgDerivativesFixture(), dataMap(), makeKlinesFixture(), TestCompute_CgDataMissing_Degraded(), TestCompute_FullSignals_Ok() (+10 more)

### Community 8 - "HTTP Client & Error Handling"
Cohesion: 0.09
Nodes (14): APIError, backoffDuration function, Client, backoffDuration(), parseRetryAfter(), TestBackoffDuration(), TestParseRetryAfter(), Get method (+6 more)

### Community 9 - "CoinGecko CoinMarkets API"
Cohesion: 0.13
Nodes (26): CoinMarketsBreadthURL(), ParseCoinMarketsBreadthResponse(), ParseCoinMarketsMomentumResponse(), ParseCoinMarketsRankedResponse(), TestCoinMarketsBreadthURL(), TestCoinMarketsBreadthURL_CustomCount(), TestParseCoinMarketsBreadthResponse_BTCReference(), TestParseCoinMarketsBreadthResponse_BTCReferenceAbsent() (+18 more)

### Community 10 - "Binance Klines API"
Cohesion: 0.14
Nodes (24): KlinesURL(), parseInt(), ParseKlinesResponse(), ParseKlinesVolumesResponse(), parseStringFloat(), abs(), contains(), containsStr() (+16 more)

### Community 11 - "Metric Registry Tests"
Cohesion: 0.22
Nodes (22): NewRegistry(), makeProvider(), mustRegisterTo(), TestMustRegister_Panics(), TestRegistry_BestProviders_CoreNamespacePriority(), TestRegistry_BestProviders_Deterministic(), TestRegistry_BestProviders_Empty(), TestRegistry_BestProviders_HighestVersionWins() (+14 more)

### Community 12 - "API Fetcher Caching"
Cohesion: 0.14
Nodes (15): countingDoer, TestFileCacheStaleAPIFailFallback(), TestFileCacheStaleAPISuccess(), TestNoCacheAPISuccess(), TestStaleMemorySkippedOnRecovery(), mockReadCloser, switchingDoer, Open() (+7 more)

### Community 13 - "Catalog & Endpoint Registration"
Cohesion: 0.13
Nodes (13): AllEndpoints(), TestCatalog_AliasesResolvable(), TestCatalog_AllProvidersRegistered(), TestCatalog_EndpointsAreKnown(), Registry, fullKey(), GlobalRegistry(), MustRegister() (+5 more)

### Community 14 - "Metric Provider Implementations"
Cohesion: 0.1
Nodes (22): LP classify, LP confidenceToFloat, Liquidity Pulse Provider, Market Breadth Compute, Market Breadth Provider, sign helper, buildSummary, buildTierDetail (+14 more)

### Community 15 - "File Cache Layer"
Cohesion: 0.17
Nodes (13): Cache, Exists(), checkClosed internal method, Clear method, Close method, Entry, filePath internal method, FilePath exported method (+5 more)

### Community 16 - "Registry Core Logic"
Cohesion: 0.1
Nodes (20): Registry.bestProvider method, Registry.BestProviders method, CompareSemVer function, CoreNamespace constant, ErrDuplicateMetric sentinel error, ErrInvalidProvider sentinel error, ErrMetricNotFound sentinel error, fullKey helper function (+12 more)

### Community 17 - "CoinGecko Global/Stables Parsers"
Cohesion: 0.16
Nodes (18): ParseGlobalDominance(), ParseGlobalResponse(), ParseStablesMarketsResponse(), ParseTopGainers(), TestParseGlobalDominance_EmptyBody(), TestParseGlobalDominance_InvalidJSON(), TestParseGlobalDominance_NoBTC(), TestParseGlobalDominance_Valid() (+10 more)

### Community 18 - "Stablecoin Power Provider Tests"
Cohesion: 0.37
Nodes (16): cgGlobalJSON(), cgStablesJSON(), dataMap3(), dlJSON(), TestCompute_DataFields(), TestCompute_DefiLlamaMissing_StillComputes(), TestCompute_DiscrepancyLow(), TestCompute_DiscrepancyMedium() (+8 more)

### Community 19 - "CoinGecko Data Structures"
Cohesion: 0.12
Nodes (16): BTCReference, GlobalURL(), StablesMarketsURL(), CoinMarketsBreadthData, CoinMarketsBreadthEntry, CoinMarketsMomentumData, CoinMarketsMomentumEntry, CoinMarketsRankedCoin (+8 more)

### Community 20 - "Market Breadth Provider Tests"
Cohesion: 0.15
Nodes (8): StoreSegmentsInContext(), TestProvider_Compute_CGUnavailable(), TestProvider_Compute_HappyPath(), TestProvider_Compute_SegmentsClamping(), TestProvider_InvalidSegments_NonAscending(), TestProvider_InvalidSegments_WrongCount(), TestProvider_InvalidSegments_WrongFormat(), TestProvider_RegisterFlags()

### Community 21 - "Market Breadth Compute Tests"
Cohesion: 0.31
Nodes (13): freshKline(), makeCounts(), TestCompute_AllTimeframesAbsent(), TestCompute_BroadScore(), TestCompute_BTCUnavailable(), TestCompute_DiscrepancyDirectionalDisagreement(), TestCompute_ExactBoundaries(), TestCompute_GhostRally() (+5 more)

### Community 22 - "API Fetcher Core"
Cohesion: 0.22
Nodes (8): Fetcher, getHandle(), New(), FetchMeta, New() constructor — creates Fetcher with cache, http client, shards, shard, CoinMarketsMomentumURL(), TestCoinMarketsMomentumURL()

### Community 23 - "CoinGecko Derivatives API"
Cohesion: 0.23
Nodes (13): DerivativesURL(), median(), ParseDerivativesResponse(), TestDerivativesURL_NoKey(), TestDerivativesURL_WithKey(), TestParseDerivativesResponse_EmptyBody(), TestParseDerivativesResponse_EvenCountMedian(), TestParseDerivativesResponse_FiltersBTCOnly() (+5 more)

### Community 24 - "CoinGecko Shared Parsers"
Cohesion: 0.15
Nodes (11): CoinMarketsBreadthEntry struct (multi-tf price changes per coin), GlobalData, GlobalResponse, median() — median of float64 slice (avg of 2 middle for even len), ParseCoinMarketsBreadthResponse() — computes per-tf green counts, excludes nulls, ParseDerivativesResponse() — aggregates BTC perpetual OI + median funding rate, ParseGlobalResponse() — unmarshals /global JSON, ParseCoinMarketsMomentumResponse() — extracts momentum data, null→0 (+3 more)

### Community 25 - "Output Envelope/Meta Tests"
Cohesion: 0.24
Nodes (7): mapsEqual(), TestCLIError_MarshalJSON(), TestMetricResult_MarshalJSON(), TestMetaBasic_MarshalJSON(), TestMetaExtended_MarshalJSON(), TestMetaFull_MarshalJSON(), TestSourceMeta_MarshalJSON()

### Community 26 - "API URL Resolution"
Cohesion: 0.32
Nodes (12): apiKey() — returns provider API key from config, Endpoint key constants (coingecko/binance/defillama/coindesk/coinmetrics), Fetch() — cache-first fetch with stale fallback on API failure, resolveTTL() — looks up per-endpoint TTL from config with bounds clamping, resolveURL() — dispatches endpoint key to provider URL builder, BaseURL constant (https://api.coingecko.com/api/v3), CoinMarketsBreadthURL() — /coins/markets for 1h/24h/7d/30d breadth, CoinMarketsMomentumURL() — /coins/markets for 24h momentum (+4 more)

### Community 27 - "Flow Tension Compute"
Cohesion: 0.2
Nodes (10): Flow Tension Compute, ComputeFlowHook, ComputeFundingRateHook, Flow Tension computeErr, ComputeExchangeNetFlow, ComputeOIChange24h, ComputeOIChangeHook, ComputeSummary (+2 more)

### Community 28 - "Binance API Client"
Cohesion: 0.33
Nodes (7): BaseURL constant, KlinesURL function, parseInt internal function, ParseKlinesResponse function, ParseKlinesVolumesResponse function, parseStringFloat internal function, binance test suite

### Community 29 - "Output Envelope Types"
Cohesion: 0.53
Nodes (5): CLIError, CLIResponse, MetricResult, WriteError() — writes error CLIResponse with CLIError to stdout, WriteSuccess() — writes successful CLIResponse JSON to stdout

### Community 30 - "MetricFloat Serialization"
Cohesion: 0.33
Nodes (6): Currency helper function, CurrencyPrecision constant, FloatPrecision constant, MetricFloat struct, NewMetricFloat constructor, Ratio helper function

### Community 31 - "Output Metadata"
Cohesion: 0.7
Nodes (4): MetaBasic, MetaExtended, MetaFull, SourceMeta

### Community 34 - "Go 1.25 Caching Patterns"
Cohesion: 0.83
Nodes (4): Go 1.25 caching patterns doc, Sharded maps (16 shards), Structured Caching Patterns, Zero-allocation keying with unique.Handle

### Community 38 - "Config Loader"
Cohesion: 0.67
Nodes (3): Load() — loads config from file (delegates to LoadWithViper), LoadWithViper() — viper-based config load with env, flags, perms check, resolveConfigPath() — .yaml↔.yml extension fallback resolution

### Community 39 - "CoinMetrics Package"
Cohesion: 0.67
Nodes (3): CommunityURL stub function, coinmetrics package (stub), ParseCommunityResponse stub function

## Knowledge Gaps
- **184 isolated node(s):** `Entry`, `record`, `MetricDef`, `MetricProvider`, `tierCoin` (+179 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **32 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Compute()` connect `Market Metrics Compute Core` to `Momentum Divergence Tests`, `MetricFloat & Provider Tests`?**
  _High betweenness centrality (0.139) - this node is a cross-community bridge._
- **Why does `Open()` connect `API Fetcher Caching` to `CLI Commands & E2E Tests`, `DefiLlama Stablecoin Supply`, `API Fetcher Core`, `File Cache Layer`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Why does `NewRootCommand()` connect `CLI Commands & E2E Tests` to `Configuration System`, `Catalog & Endpoint Registration`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Are the 41 inferred relationships involving `NewRootCommand()` (e.g. with `LoadWithViper()` and `StoreInContext()`) actually correct?**
  _`NewRootCommand()` has 41 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `NewRootCommand` (e.g. with `flagRegistrar interface` and `catalog.go (metric provider blank imports)`) actually correct?**
  _`NewRootCommand` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 32 inferred relationships involving `SetWriter()` (e.g. with `TestWriteSuccess()` and `TestWriteError()`) actually correct?**
  _`SetWriter()` has 32 INFERRED edges - model-reasoned connections that need verification._
- **Are the 32 inferred relationships involving `Writer()` (e.g. with `TestWriteSuccess()` and `TestWriteError()`) actually correct?**
  _`Writer()` has 32 INFERRED edges - model-reasoned connections that need verification._