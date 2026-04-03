package fetcher

import (
	"proxy-checker/internal/common"
	"strings"
)

const (
	ProxiFlyURL = "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/"

	proxiflyHTTPFileName    = "http/data.txt"
	proxiflyHTTPSFileName   = "https/data.txt"
	proxiflySsocks4FileName = "socks4/data.txt"
	proxiflyHSocks5FileName = "socks5/data.txt"
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
		fileNames = append(fileNames, proxiflyHSocks5FileName)
	case common.ProxySOCKS4:
		fileNames = append(fileNames, proxiflySsocks4FileName)
	case common.ProxyHTTPS:
		fileNames = append(fileNames, proxiflyHTTPSFileName)
	case common.ProxyHTTP:
		fileNames = append(fileNames, proxiflyHTTPFileName)
	case common.ProxyAll:
		fileNames = append(fileNames, proxiflyHTTPFileName, proxiflyHTTPSFileName, proxiflySsocks4FileName, proxiflyHSocks5FileName)
	default:
		fileNames = append(fileNames, proxiflyHSocks5FileName)
	}

	return fileNames
}
