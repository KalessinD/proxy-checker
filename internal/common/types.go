package common

import "strings"

type (
	Source string

	ProxyType string

	LogLevel string
)

const (
	ProxyHTTP   ProxyType = "http"
	ProxyHTTPS  ProxyType = "https"
	ProxySOCKS4 ProxyType = "socks4"
	ProxySOCKS5 ProxyType = "socks5"
	ProxyAll    ProxyType = "all"

	SourceProxyMania Source = "proxymania"
	SourceTheSpeedX  Source = "thespeedx"
	SourceProxifly   Source = "proxifly"

	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
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

func SourcesToStrings(sources []Source) []string {
	result := make([]string, len(sources))
	for i, src := range sources {
		result[i] = string(src)
	}
	return result
}

func JoinSources(sources []Source, separator string) string {
	return strings.Join(SourcesToStrings(sources), separator)
}
