package fetcher

import (
	"proxy-checker/internal/common"
	"strings"
)

const (
	ProxiFlyURL = "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/"

	ProxiflyHTTPFileName   = "http/data.txt"
	ProxiflyHTTPSFileName  = "https/data.txt"
	ProxiflySocks4FileName = "socks4/data.txt"
	ProxiflySocks5FileName = "socks5/data.txt"
)

type (
	ProxiflyProvider struct {
		BaseURL string
	}
)

func NewProxiflyProvider() TextProviderInterface {
	return &ProxiflyProvider{
		BaseURL: ProxiFlyURL,
	}
}

func (p *ProxiflyProvider) ParseString(line string) []string {
	_, addr, _ := strings.Cut(line, "://")
	return strings.Split(addr, ":")
}

func (p *ProxiflyProvider) GetTypeFromFilename(filename string) common.ProxyType {
	return common.ProxyType(strings.ToLower(strings.Split(filename, "/")[0]))
}

func (p *ProxiflyProvider) GetFilesByType(proxyType common.ProxyType) []string {
	return MapProxyTypeToFilenames(
		proxyType,
		ProxiflyHTTPFileName,
		ProxiflyHTTPSFileName,
		ProxiflySocks4FileName,
		ProxiflySocks5FileName,
	)
}
