package fetcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TheSpeedXFetcher реализует получение прокси из репозитория TheSpeedX/PROXY-List
type TheSpeedXFetcher struct{}

const theSpeedXBaseURL = "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/"

func (f *TheSpeedXFetcher) Fetch(ctx context.Context, settings Settings) ([]ProxyItem, error) {
	fileName := ""
	switch strings.ToLower(settings.Type) {
	case "socks5":
		fileName = "socks5.txt"
	case "socks4":
		fileName = "socks4.txt"
	case "http", "https":
		fileName = "http.txt"
	case "all":
		fileName = "http.txt" // Берем http по умолчанию для all
	default:
		fileName = "http.txt"
	}

	targetURL := theSpeedXBaseURL + fileName

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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
			Type:    settings.Type,
			Country: "N/A",
			RTT:     "N/A",
			RTTms:   0,
		})
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, err
	}

	return items, nil
}
