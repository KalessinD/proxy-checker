package fetcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"strings"
	"time"
)

type TheSpeedXFetcher struct {
	BaseURL string
}

const (
	TheSpeedXBaseURL = "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/"

	httpFileName   = "http.txt"
	socks4FileName = "socks4.txt"
	socks5FileName = "socks5.txt"
)

func NewTheSpeedXFetcher() *TheSpeedXFetcher {
	return &TheSpeedXFetcher{
		BaseURL: TheSpeedXBaseURL,
	}
}

func (f *TheSpeedXFetcher) Fetch(ctx context.Context, settings Settings) ([]ProxyItem, error) {
	var fileNames []string
	haveToGetTypeFromFileName := false

	// Используем типизированные константы
	switch settings.Type {
	case common.ProxySOCKS5:
		fileNames = append(fileNames, socks5FileName)
	case common.ProxySOCKS4:
		fileNames = append(fileNames, socks4FileName)
	case common.ProxyHTTP, common.ProxyHTTPS:
		fileNames = append(fileNames, httpFileName)
	case common.ProxyAll:
		fileNames = append(fileNames, httpFileName, socks4FileName, socks5FileName)
		haveToGetTypeFromFileName = true
	default:
		fileNames = append(fileNames, socks5FileName)
	}

	var items []ProxyItem

	for _, fileName := range fileNames {
		targetURL := f.BaseURL + fileName

		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(i18n.T("fetcher.err_fetch_list_status"), resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				continue
			}
			host := parts[0]
			port := parts[1]

			proxyType := settings.Type
			if haveToGetTypeFromFileName {
				proxyType = common.ProxyType(strings.ToLower(strings.Split(fileName, ".")[0]))
			}

			countryCode := i18n.T("common.na")
			if settings.Resolver != nil {
				countryCode = settings.Resolver.ResolveCountry(host)
			}

			items = append(items, ProxyItem{
				Host:    host,
				Port:    port,
				Type:    proxyType,
				Country: countryCode,
				RTT:     i18n.T("common.na"),
				RTTms:   0,
			})
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			return nil, err
		}
	}
	return items, nil
}
