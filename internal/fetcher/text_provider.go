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

type (
	TextListFetcher struct {
		BaseURL  string
		provider TextProviderInterface
	}

	TextProviderInterface interface {
		GetFilesByType(proxyType common.ProxyType) []string
		GetTypeFromFilename(filename string) common.ProxyType
		ParseString(str string) []string
	}
)

func NewTextListFetcher(baseURL string, provider TextProviderInterface) *TextListFetcher {
	return &TextListFetcher{
		BaseURL:  baseURL,
		provider: provider,
	}
}

func (f *TextListFetcher) Fetch(ctx context.Context, settings Settings) ([]*ProxyItem, error) {
	fileNames := f.provider.GetFilesByType(settings.Type)

	var items []*ProxyItem

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
			return nil, fmt.Errorf("%s %d", i18n.T("fetcher.err_fetch_list_status"), resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := f.provider.ParseString(line)
			if len(parts) != 2 {
				continue
			}
			host := parts[0]
			port := parts[1]

			proxyType := settings.Type
			if proxyType == common.ProxyAll {
				proxyType = f.provider.GetTypeFromFilename(fileName)
			}

			countryCode := i18n.T("common.na")
			if settings.Resolver != nil {
				countryCode = settings.Resolver.ResolveCountry(host, settings.Lang)
			}

			items = append(items, &ProxyItem{
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
