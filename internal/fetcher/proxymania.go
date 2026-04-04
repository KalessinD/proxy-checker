package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ProxyManiaFetcher struct {
	BaseURL string
	Logger  common.LoggerInterface
}

const (
	ProxyManiaBaseURL = "https://proxymania.su/en/free-proxy?speed=100&type=SOCKS5"
	DefaultUnknownRTT = 99999
)

func NewProxyManiaFetcher(logger common.LoggerInterface) *ProxyManiaFetcher {
	return &ProxyManiaFetcher{
		BaseURL: ProxyManiaBaseURL,
		Logger:  logger,
	}
}

func (f *ProxyManiaFetcher) Fetch(ctx context.Context, settings Settings) ([]*ProxyItem, error) {
	parsedURL, err := url.Parse(f.BaseURL)
	if err != nil {
		return nil, err
	}
	query := parsedURL.Query()

	if settings.Type == common.ProxyAll {
		query.Del("type")
	} else {
		typeMap := map[common.ProxyType]string{
			common.ProxySOCKS5: "SOCKS5", common.ProxySOCKS4: "SOCKS4",
			common.ProxyHTTP: "HTTP", common.ProxyHTTPS: "HTTPS",
		}
		if t, ok := typeMap[settings.Type]; ok {
			query.Set("type", t)
		}
	}

	query.Set("speed", strconv.Itoa(settings.MaxRTT))
	parsedURL.RawQuery = query.Encode()
	startURL := parsedURL.String()

	visitedURLs := make(map[string]bool)
	var allProxies []*ProxyItem

	queue := []string{startURL}
	pagesFetched := 0
	maxPages := settings.Pages

	baseParsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: fetcherClientTimeout}

	for len(queue) > 0 {
		if maxPages > 0 && pagesFetched >= maxPages {
			break
		}

		currentURL := queue[0]
		queue = queue[1:]

		if visitedURLs[currentURL] {
			continue
		}
		visitedURLs[currentURL] = true

		proxies, pageLinks, err := f.fetchSinglePage(ctx, client, currentURL, baseParsedURL)
		if err != nil {
			f.Logger.Warnf("failed to fetch page %s: %v", strings.TrimSpace(currentURL), err)
			continue
		}

		allProxies = append(allProxies, proxies...)
		pagesFetched++

		for _, link := range pageLinks {
			absURL, err := baseParsedURL.Parse(link)
			if err != nil {
				continue
			}
			normalized := absURL.String()
			if !visitedURLs[normalized] {
				queue = append(queue, normalized)
			}
		}
	}

	sort.Slice(allProxies, func(i, j int) bool {
		return allProxies[i].RTTms < allProxies[j].RTTms
	})

	return allProxies, nil
}

func (f *ProxyManiaFetcher) fetchSinglePage(ctx context.Context, client *http.Client, urlStr string, _ *url.URL) ([]*ProxyItem, []string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%s %d", i18n.T("fetcher.err_status"), res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var proxies []*ProxyItem
	var pageLinks []string

	siteToInternalType := map[string]common.ProxyType{
		"SOCKS5": common.ProxySOCKS5,
		"SOCKS4": common.ProxySOCKS4,
		"HTTP":   common.ProxyHTTP,
		"HTTPS":  common.ProxyHTTPS,
	}

	doc.Find("table.table_proxychecker tbody tr").Each(func(_ int, row *goquery.Selection) {
		addressCell := row.Find("td.proxy-cell")
		fullAddress := strings.TrimSpace(addressCell.Text())
		if fullAddress == "" {
			return
		}

		parts := strings.Split(fullAddress, ":")
		host, port := fullAddress, ""
		if len(parts) == 2 {
			host, port = parts[0], parts[1]
		}

		countryCell := row.Find("td.country-cell")
		country := strings.TrimSpace(countryCell.Text())
		typeCell := countryCell.Next()

		proxyTypeStr := strings.TrimSpace(typeCell.Text())
		proxyType := siteToInternalType[proxyTypeStr]

		speedCell := row.Find("td.speed-fast")
		rttText := strings.TrimSpace(speedCell.Text())

		rttMs := DefaultUnknownRTT
		if p := strings.Fields(rttText); len(p) > 0 {
			if val, err := strconv.Atoi(p[0]); err == nil {
				rttMs = val
			}
		}

		proxies = append(proxies, &ProxyItem{
			Host: host, Port: port, Country: country, Type: proxyType, RTT: rttText, RTTms: rttMs,
		})
	})

	doc.Find("ul.pagination li.page-item").Each(func(_ int, s *goquery.Selection) {
		if s.HasClass("active") {
			return
		}
		a := s.Find("a.page-link")
		if _, exists := a.Attr("rel"); exists {
			return
		}
		if href, exists := a.Attr("href"); exists {
			pageLinks = append(pageLinks, href)
		}
	})

	return proxies, pageLinks, nil
}
