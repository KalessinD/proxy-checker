// internal/common/types.go
package common

const (
	ProxyHTTP        ProxyType = "http"
	ProxyHTTPS       ProxyType = "https"
	ProxySOCKS4      ProxyType = "socks4"
	ProxySOCKS5      ProxyType = "socks5"
	ProxyAll         ProxyType = "all"
	SourceProxyMania Source    = "proxymania"
	SourceTheSpeedX  Source    = "thespeedx"
	SourceProxifly   Source    = "proxifly"
)

type (
	Source string

	ProxyType string
)

var allowedSources = map[Source]struct{}{
	SourceProxyMania: {},
	SourceTheSpeedX:  {},
	SourceProxifly:   {},
}

func IsKnownSource(source Source) bool {
	_, ok := allowedSources[source]
	return ok
}
