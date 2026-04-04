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
)

type (
	TextListFetcher struct {
		BaseURL  string
		provider TextProviderInterface
		Logger   common.LoggerInterface
	}

	TextProviderInterface interface {
		GetFilesByType(proxyType common.ProxyType) []string
		GetTypeFromFilename(filename string) common.ProxyType
		ParseString(str string) []string
	}
)

func NewTextListFetcher(baseURL string, provider TextProviderInterface, logger common.LoggerInterface) *TextListFetcher {
	return &TextListFetcher{
		BaseURL:  baseURL,
		provider: provider,
		Logger:   logger,
	}
}

func (f *TextListFetcher) Fetch(ctx context.Context, settings Settings) ([]*ProxyItem, error) {
	fileNames := f.provider.GetFilesByType(settings.Type)

	var items []*ProxyItem
	var fetchErrors []string

	client := &http.Client{Timeout: fetcherClientTimeout}

	for _, fileName := range fileNames {
		targetURL := f.BaseURL + fileName

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Sprintf("%s: %v", fileName, err))
			f.Logger.Warnf("failed to fetch %s: %v", fileName, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fetchErrors = append(fetchErrors, fmt.Sprintf("%s: status %d", fileName, resp.StatusCode))
			f.Logger.Warnf("failed to fetch %s: status %d", fileName, resp.StatusCode)
			resp.Body.Close()
			continue
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
			fetchErrors = append(fetchErrors, fmt.Sprintf("%s: read error %v", fileName, err))
			f.Logger.Warnf("error reading %s: %v", fileName, err)
		}

		resp.Body.Close()
	}

	if len(items) == 0 && len(fetchErrors) > 0 {
		return nil, fmt.Errorf("failed to fetch all sources: %s", strings.Join(fetchErrors, "; "))
	}

	return items, nil
}
