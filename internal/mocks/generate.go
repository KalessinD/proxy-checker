//go:build generate

// Regeneration instructions for hand-maintained mock package.
//
// Run: go generate ./internal/mocks/...
// Requires the mockgen binary: go install go.uber.org/mock/mockgen@latest
//
// Only interfaces consumed by tests are mocked on purpose to avoid dead code:
// cache.StorageInterface, services.PipelineRunner, sysproxy.SystemProxyManager.
// ProxyVerifier and GeoIPResolver are intentionally NOT mocked here:
// ProxyVerifier has a package-local test double in internal/services tests,
// GeoIPResolver is not used by any test yet. Add a directive only together
// with the test that consumes it.
package mocks

//go:generate mockgen -destination=mock_cache.go -package=mocks proxy-checker/internal/cache StorageInterface
//go:generate mockgen -destination=mock_services.go -package=mocks proxy-checker/internal/services PipelineRunner
//go:generate mockgen -destination=mock_sys_proxy.go -package=mocks proxy-checker/internal/sysproxy SystemProxyManager
