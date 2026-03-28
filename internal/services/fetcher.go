package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ProxyItem struct {
    Host    string
    Port    string
    Country string
    Type    string
    RTT     string
    RTTms   int
}

// FetchAllPages обходит страницы с пейджинацией
func FetchAllPages(ctx context.Context, startURL string, maxPages int) ([]ProxyItem, error) {
    visitedURLs := make(map[string]bool)
    var allProxies []ProxyItem

    queue := []string{startURL}
    pagesFetched := 0

    baseParsedURL, err := url.Parse(startURL)
    if err != nil {
        return nil, err
    }

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

        // Сообщаем о прогрессии через fmt (или можно вернуть callback)
        fmt.Printf("\rПарсинг страницы %d...", pagesFetched+1)

        proxies, pageLinks, err := fetchSinglePage(ctx, currentURL)
        if err != nil {
            // Выводим ошибку, но продолжаем
            fmt.Printf("\nОшибка парсинга %s: %v\n", currentURL, err)
            continue
        }

        allProxies = append(allProxies, proxies...)
        pagesFetched++

        // Добавляем ссылки на следующие страницы
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
    
    fmt.Println() // Перенос строки после завершения парсинга

    // Сортируем итоговый список
    sort.Slice(allProxies, func(i, j int) bool {
        return allProxies[i].RTTms < allProxies[j].RTTms
    })

    return allProxies, nil
}

// fetchSinglePage парсит одну страницу ( приватная)
func fetchSinglePage(ctx context.Context, urlStr string) ([]ProxyItem, []string, error) {
    client := &http.Client{Timeout: 20 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
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

    if res.StatusCode != 200 {
        return nil, nil, fmt.Errorf("status %d", res.StatusCode)
    }

    doc, err := goquery.NewDocumentFromReader(res.Body)
    if err != nil {
        return nil, nil, err
    }

    var proxies []ProxyItem
    var pageLinks []string

    // Парсинг таблицы
    doc.Find("table.table_proxychecker tbody tr").Each(func(i int, row *goquery.Selection) {
        addressCell := row.Find("td.proxy-cell")
        fullAddress := strings.TrimSpace(addressCell.Text())
        if fullAddress == "" {
            return
        }

        parts := strings.Split(fullAddress, ":")
        host, port := "", ""
        if len(parts) == 2 {
            host, port = parts[0], parts[1]
        } else {
            host = fullAddress
        }

        countryCell := row.Find("td.country-cell")
        country := strings.TrimSpace(countryCell.Text())
        typeCell := countryCell.Next()
        proxyType := strings.TrimSpace(typeCell.Text())
        speedCell := row.Find("td.speed-fast")
        rttText := strings.TrimSpace(speedCell.Text())

        rttMs := 99999
        if p := strings.Fields(rttText); len(p) > 0 {
            if val, err := strconv.Atoi(p[0]); err == nil {
                rttMs = val
            }
        }

        proxies = append(proxies, ProxyItem{
            Host: host, Port: port, Country: country, Type: proxyType, RTT: rttText, RTTms: rttMs,
        })
    })

    // Парсинг пейджинации
    doc.Find("ul.pagination li.page-item").Each(func(i int, s *goquery.Selection) {
        if s.HasClass("active") {
            return
        }
        a := s.Find("a.page-link")
        if _, exists := a.Attr("rel"); exists {
            return // Skip "Next"
        }
        if href, exists := a.Attr("href"); exists {
            pageLinks = append(pageLinks, href)
        }
    })

    return proxies, pageLinks, nil
}
