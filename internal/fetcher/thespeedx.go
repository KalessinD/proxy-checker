package fetcher

import (
	"proxy-checker/internal/common"
	"strings"
)

const (
	TheSpeedXBaseURL = "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/"

	ThespeedxHTTPFileName   = "http.txt"
	ThespeedxHTTPSFileName  = "https.txt"
	ThespeedxSocks4FileName = "socks4.txt"
	ThespeedxSocks5FileName = "socks5.txt"
)

type (
	TheSpeedXProvider struct {
		BaseURL string
	}
)

func NewTheSpeedXProvider() TextProviderInterface {
	return &TheSpeedXProvider{
		BaseURL: TheSpeedXBaseURL,
	}
}

func (p *TheSpeedXProvider) ParseString(line string) []string {
	return strings.Split(line, ":")
}

func (p *TheSpeedXProvider) GetTypeFromFilename(filename string) common.ProxyType {
	return common.ProxyType(strings.ToLower(strings.Split(filename, ".")[0]))
}

func (p *TheSpeedXProvider) GetFilesByType(proxyType common.ProxyType) []string {
	return MapProxyTypeToFilenames(
		proxyType, ThespeedxHTTPFileName, ThespeedxHTTPSFileName, ThespeedxSocks4FileName, ThespeedxSocks5FileName,
	)
}
