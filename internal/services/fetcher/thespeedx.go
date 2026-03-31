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

type TheSpeedXFetcher struct{}

const theSpeedXBaseURL = "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/"

func (f *TheSpeedXFetcher) Fetch(ctx context.Context, settings Settings) ([]ProxyItem, error) {
	var fileName string

	// Используем типизированные константы
	switch settings.Type {
	case common.ProxySOCKS5:
		fileName = "socks5.txt"
	case common.ProxySOCKS4:
		fileName = "socks4.txt"
	case common.ProxyHTTP, common.ProxyHTTPS:
		fileName = "http.txt"
	case common.ProxyAll:
		fileName = "http.txt"
	default:
		fileName = "http.txt"
	}

	targetURL := theSpeedXBaseURL + fileName

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
		return nil, fmt.Errorf("failed to fetch proxy list: status %d", resp.StatusCode)
	}

	var items []ProxyItem

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

		items = append(items, ProxyItem{
			Host:    host,
			Port:    port,
			Type:    settings.Type, // Присваиваем строго типизированный параметр
			Country: i18n.T("common.na"),
			RTT:     i18n.T("common.na"),
			RTTms:   0,
		})
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, err
	}

	return items, nil
}
