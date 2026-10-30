# Changelog

Format based on [Keep a Changelog](https://keepachangelog.com/), project adheres to [Semantic Versioning](https://semver.org/).
History before 0.0.20 is migrated from the old `Changelog` file (kept as-is, grouped by changelog categories).

## [0.0.20] - 2026-10-29

### Added
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
- GUI Defect #2: `GetActiveProxy` (external command) no longer blocks the UI on every filter click; the active system proxy is cached and refreshed only on rare user actions (startup, proxy apply, switch toggle).
- `networksetup` invocation on macOS: the network service was prepended to the argument list, so every set-call failed with exit status 5 ("command is not recognized"); `SetIgnoreHosts` no longer doubles the service, and errors now include stdout (where `networksetup` reports failures).
- sysproxy errors on macOS are localized (ru/en) via `common/i18n` instead of hardcoded English strings.
- GUI: all widget mutations in the background pipeline goroutine (`progress`/`progressBar`, log `RichText` segments) are now wrapped in `fyne.Do` — removes the "should have been called in fyne.Do[AndWait]" warnings and data races with the UI thread.

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
