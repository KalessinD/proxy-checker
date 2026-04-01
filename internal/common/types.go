// internal/common/types.go
package common

type ProxyType string

const (
	ProxyHTTP   ProxyType = "http"
	ProxyHTTPS  ProxyType = "https"
	ProxySOCKS4 ProxyType = "socks4"
	ProxySOCKS5 ProxyType = "socks5"
	ProxyAll    ProxyType = "all"
)

type Source string

const (
	SourceProxyMania Source = "proxymania"
	SourceTheSpeedX  Source = "thespeedx"
)
