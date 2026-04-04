package fetcher

import (
	"proxy-checker/internal/common"
	"strings"
)

const (
	ProxiFlyURL = "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/"

	ProxiflyHTTPFileName    = "http/data.txt"
	ProxiflyHTTPSFileName   = "https/data.txt"
	ProxiflySsocks4FileName = "socks4/data.txt"
	ProxiflyHSocks5FileName = "socks5/data.txt"
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
	var fileNames []string

	switch proxyType {
	case common.ProxySOCKS5:
		fileNames = append(fileNames, ProxiflyHSocks5FileName)
	case common.ProxySOCKS4:
		fileNames = append(fileNames, ProxiflySsocks4FileName)
	case common.ProxyHTTPS:
		fileNames = append(fileNames, ProxiflyHTTPSFileName)
	case common.ProxyHTTP:
		fileNames = append(fileNames, ProxiflyHTTPFileName)
	case common.ProxyAll:
		fileNames = append(fileNames, ProxiflyHTTPFileName, ProxiflyHTTPSFileName, ProxiflySsocks4FileName, ProxiflyHSocks5FileName)
	default:
		fileNames = append(fileNames, ProxiflyHSocks5FileName)
	}

	return fileNames
}
