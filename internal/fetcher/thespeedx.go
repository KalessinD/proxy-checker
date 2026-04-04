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
	var fileNames []string

	switch proxyType {
	case common.ProxySOCKS5:
		fileNames = append(fileNames, ThespeedxSocks5FileName)
	case common.ProxySOCKS4:
		fileNames = append(fileNames, ThespeedxSocks4FileName)
	case common.ProxyHTTPS:
		fileNames = append(fileNames, ThespeedxHTTPSFileName)
	case common.ProxyHTTP:
		fileNames = append(fileNames, ThespeedxHTTPFileName)
	case common.ProxyAll:
		fileNames = append(fileNames, ThespeedxHTTPFileName, ThespeedxHTTPSFileName, ThespeedxSocks4FileName, ThespeedxSocks5FileName)
	default:
		fileNames = append(fileNames, ThespeedxSocks5FileName)
	}

	return fileNames
}
