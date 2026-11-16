# Changelog

Format based on [Keep a Changelog](https://keepachangelog.com/), project adheres to [Semantic Versioning](https://semver.org/).
History before 0.0.20 is migrated from the old `Changelog` file (kept as-is, grouped by changelog categories).

## [0.0.21] - Unreleased

### Added
- Proxy monitoring with auto-failover (TODO item 1): a background health-check worker probes the OS-level active proxy on a timer and, on failure, switches to the best candidate from the results table (top N=5 by TCP RTT, batch-checked in one pass) or only notifies, per the configured mode: `any` / `same_country` / `none` (config `monitor_mode`, default `any`). Country-less candidates can be excluded (`monitor_require_country`); a stale pool (expired results-cache TTL) is never used for switching. Settings live in the settings screen (`monitor_mode`, `monitor_interval_min` 1/2/5/10) and hot-reload into a running monitor. The failed row is marked with a red tint, its Apply button is disabled, and the event is reported in the log plus a desktop notification; the monitor is started/stopped by the "Monitor" checkbox in the main controls bar.
- TTL indicator in the GUI: a label next to the source/country filters shows how long the loaded cached results remain valid (the soonest expiring source), re-rendered once a minute; hidden when there is no cached data, switches to "Cache expired" after the deadline. The expiry is read from the results cache through the new `StorageInterface.ExpireAt` port and refreshed after every scan.

## [0.0.20] - 2026-10-29

### Added
- `make install-deps` (Linux): checks the GUI build dependencies the way glfw does (`pkg-config`) and offers to install the missing `apt-get` packages — confirmation prompt with the sudo password entered interactively by the user.
- Raw list cache: proxy lists fetched from sources are stored as plain text files (one per `source:type`) with a fixed 4-hour TTL and are reused by the pipeline before hitting the source; a scan stays alive on a fresh cache even when the source is unreachable. Managed by the new `services.ListCache` port, implemented by `cache.ListStorage`; disabled when `PipelineParams.ListCache` is nil.
- macOS support (alpha): system-proxy management via `networksetup` for every network service, deterministic active-service detection (default route → hardware port mapping); platform factories split into `factory_linux.go` / `factory_darwin.go` / `factory_other.go` with `go:build` tags in file headers.
- CI: dedicated macOS job running the full lint+test suite natively (darwin-tagged sysproxy tests now run on CI); darwin releases ship a single universal binary (`proxy-checker-macos`, arm64 + amd64 via `lipo`), built on a native macOS runner since the GUI requires cgo.
- `make install-macos` / `make uninstall-macos` (binary to `/usr/local/bin`), `make install-macos-app` (`.app` bundle in `/Applications` with generated `.icns` icon and Spotlight indexing disabled for `bin/`).
- Config option `dial_timeout` (TOML, default 30s): the TCP-dial phase gets its own timeout instead of the hardcoded 10s; still clamped by the per-check context deadline.
- Ports in `internal/services`: `ProxySource` (raw proxy retrieval) and `ProxyChecker` (network validation port with `CheckProxy`/`CheckBatch`); the concrete checker implementation is unexported.
- Composition root `internal/app/builder.go`: GUI dependencies are assembled via `gui.AppDependencies` (the cache is no longer created inside `NewAppGUI`).
- Test tooling: mockgen directives are colocated with their interface declarations; `make generate-mocks` target with a friendly guard for a missing `mockgen` binary.
- `sysproxy.CommandRunner` port: all `networksetup` orchestration (active-service detection, proxy state toggling, bypass domains) moved from the darwin-tagged `macos.go` into the platform-independent `networksetup.go` parameterized by the runner; darwin-specific code now only wires the real `os/exec` runner in the factory. The macOS proxy logic runs under tests on any OS (fake runner covers argument order, service resolution and error paths); `SetProxy` no longer resolves the active network service twice.

### Changed
- Pipeline configuration is passed as the narrow `services.PipelineConfig` DTO (with embedded `CheckParams`): the `CheckBatch` signature shrank from 9 positional arguments to `ctx, proxies, params, verifier, progress`; `RunPipeline` no longer sees the whole application config.
- `ProxyChecker.CheckProxy` and `ProxyVerifier.Verify` now take `proxyAddr` plus a single `*CheckParams` (settings bundle) instead of loose arguments; the `all`-type mode resolution moved into `CheckBatch`; new `services.NewCheckParams(cfg)` constructor is the single place projecting config onto check settings.
- `ShowMainScreen` decomposed into four builder helpers; the cache/notification part of `runBatchCheck` extracted into `persistScanResults`.
- `internal/sysproxy`: the runtime `GOOS` checks replaced by compile-time platform-specific factories; GVariant helpers moved to the linux-only manager file.
- `.golangci` config migrated to the linter v2 format (`.golangci.json`).
- Fyne upgraded from 2.7.3 to 2.8.2: text rendering and repaint performance (benefits the log view), macOS menu and fullscreen fixes, Wayland enabled by default on Linux. Go >= 1.22 required (already satisfied).
- Hand-written mocks replaced with `go.uber.org/mock` (mockgen); pipeline and checker tests rewritten on the mocks.

### Fixed
- The raw-list cache no longer persists empty fetch results and treats an empty cached list as a cache miss: a blocked source (ISP/RKN stub page) could otherwise pin zero proxies for the whole 4-hour TTL.
- Sources that returned zero proxies are no longer silent: the GUI log shows a warning and the CLI prints one to stderr (a silent zero made blocked sources hard to diagnose).
- proxymania: the experimental TLS certificate verification skip was reverted — the self-signed certificate belongs to the ISP/RKN block stub, not to the site; the blocker is documented in the README.
- GUI Defect #2: `GetActiveProxy` (external command) no longer blocks the UI on every filter click; the active system proxy is cached and refreshed only on rare user actions (startup, proxy apply, switch toggle).
- `networksetup` invocation on macOS: the network service was prepended to the argument list, so every set-call failed with exit status 5 ("command is not recognized"); `SetIgnoreHosts` no longer doubles the service, and errors now include stdout (where `networksetup` reports failures).
- sysproxy errors on macOS are localized (ru/en) via `common/i18n` instead of hardcoded English strings.
- GUI: all widget mutations in the background pipeline goroutine (`progress`/`progressBar`, log `RichText` segments) are now wrapped in `fyne.Do` — removes the "should have been called in fyne.Do[AndWait]" warnings and data races with the UI thread.
- GeoIP database path from the config is normalized on load: a lost leading slash (e.g. `Users/...` instead of `/Users/...`) is restored and `~` is expanded, so a valid `.mmdb` is found again.
- The missing Fyne ru translation for the `Select` placeholder ("(Select one)") is registered by the app, removing the "Translation failure" noise on every start.
- Fetch statistics in the GUI log and CLI output are broken down per source (`source: count` lines, one per configured source, in fetch order) before the total line; previously only the aggregated number was printed. The pipeline now reports fetch counts through `services.FetchStats` instead of a bare total.
- CI build dependencies for Fyne 2.8: `libwayland-dev` and `libxkbcommon-dev` added to the install step (glfw 3.4 compiles the Wayland backend and xkbcommon headers); on local Linux machines these packages must be installed manually too. Build prerequisites are now documented in the README (with the X11-only fallback `GOFLAGS="-tags=x11"`).

## [0.0.19] - 2026-04-06

### Added
- Filter by country.
- Filter by source.
- Ability to copy data into the clipboard.
- Sorting of the TCP/HTTP columns in numerical order.
- System notifications after a completed scan.

## [0.0.18] - 2026-04-06

### Changed
- Minor refactoring, added test cases.

## [0.0.17] - 2026-04-06

### Added
- Search via multiple sources.
- Sortable table columns.

### Changed
- README.md updates.

## [0.0.16] - 2026-04-05

### Added
- Embedded application icon (`go:embed`).

## [0.0.15] - 2026-04-04

### Added
- Row highlighting for the active system proxy.
- About dialog window.

## [0.0.14] - 2026-04-04

### Added
- Cache for each source + proxy type pair.

### Changed
- Logger has been reworked.

## [0.0.13] - 2026-04-04

### Changed
- Refactoring.

## [0.0.12] - 2026-04-03

### Added
- New source: proxifly.

## [0.0.11] - 2026-04-03

### Changed
- Refactoring, some tests.
- README.md and AGENTS.md updates.

## [0.0.10] - 2026-04-03

### Fixed
- Displaying of the check result.

## [0.0.9] - 2026-04-03

### Added
- Settings screen: "ignore-hosts" for proxies.
- Support for mmdb GeoIP formats to determine regions when it is not provided by the source.

### Changed
- Minor refactoring, some tests, AGENTS.md rules update.

## [0.0.8] - 2026-04-01

### Added
- Initial base version.
