package fetcher

import (
	"proxy-checker/internal/common"
	"strings"
)

const (
	TheSpeedXBaseURL = "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/"

	thespeedxHTTPFileName   = "http.txt"
	thespeedxHTTPSFileName  = "http.txt"
	thespeedxSocks4FileName = "socks4.txt"
	thespeedxSocks5FileName = "socks5.txt"
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
		fileNames = append(fileNames, thespeedxSocks5FileName)
	case common.ProxySOCKS4:
		fileNames = append(fileNames, thespeedxSocks4FileName)
	case common.ProxyHTTPS:
		fileNames = append(fileNames, thespeedxHTTPSFileName)
	case common.ProxyHTTP:
		fileNames = append(fileNames, thespeedxHTTPFileName)
	case common.ProxyAll:
		fileNames = append(fileNames, thespeedxHTTPFileName, thespeedxHTTPSFileName, thespeedxSocks5FileName)
	default:
		fileNames = append(fileNames, thespeedxSocks5FileName)
	}

	return fileNames
}
