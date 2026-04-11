//go:build generate

// run: go generate ./internal/mocks/...

package mocks

import (
    "proxy-checker/internal/cache"
    "proxy-checker/internal/common"
    "proxy-checker/internal/services"
    "proxy-checker/internal/sysproxy"

    "go.uber.org/mock/mockgen"
)

//go:generate mockgen -destination=mock_cache.go -package=mocks proxy-checker/internal/cache StorageInterface
//go:generate mockgen -destination=mock_services.go -package=mocks proxy-checker/internal/services PipelineRunner
//go:generate mockgen -destination=mock_services_verifier.go -package=mocks proxy-checker/internal/services ProxyVerifier
//go:generate mockgen -destination=mock_sysproxy.go -package=mocks proxy-checker/internal/sysproxy SystemProxyManager,CommandExecutor
//go:generate mockgen -destination=mock_geoip.go -package=mocks proxy-checker/internal/common GeoIPResolver

