package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"proxy-checker/internal/proxies"
	"proxy-checker/internal/services/fetcher"
)

type Result struct {
	ProxyLatency    time.Duration
	ReqLatency      time.Duration
	StatusCode      int
	Error           error
	ProxyLatencyStr string
	ReqLatencyStr   string
}

// ProxyItem теперь алиас к структуре из fetcher
type ProxyItem = fetcher.ProxyItem

type ProxyItemFull struct {
	ProxyItem
	CheckResult Result
}

func CheckBatch(
	ctx context.Context,
	proxiesList []ProxyItem,
	dest string,
	mode string,
	timeout time.Duration,
	workers int,
	progressCallback func(current, total int32),
) []ProxyItemFull {
	jobs := make(chan ProxyItem, len(proxiesList))
	results := make(chan ProxyItemFull, len(proxiesList))

	var wg sync.WaitGroup
	var processedCount int32
	totalCount := int32(len(proxiesList))

	// Worker logic
	worker := func() {
		defer wg.Done()
		for p := range jobs {
			// Проверяем контекст перед началом новой задачи
			select {
			case <-ctx.Done():
				return
			default:
			}

			currentMode := mode
			if mode == "all" {
				currentMode = strings.ToLower(p.Type)
			}

			addr := fmt.Sprintf("%s:%s", p.Host, p.Port)
			// Используем ctx для таймаута, но оборачиваем в новый context для жесткого лимита
			// Важно: ctx передается в NewRequestWithContext внутри CheckProxy
			ctxCheck, cancel := context.WithTimeout(ctx, timeout)
			res := CheckProxy(ctxCheck, addr, dest, currentMode)
			cancel()

			// Если контекст отменен, не отправляем результат в канал, чтобы не блокировать
			if ctx.Err() != nil {
				return
			}

			results <- ProxyItemFull{ProxyItem: p, CheckResult: res}

			if progressCallback != nil {
				current := atomic.AddInt32(&processedCount, 1)
				progressCallback(current, totalCount)
			}
		}
	}

	for w := 1; w <= workers; w++ {
		wg.Add(1)
		go worker()
	}

	// Dispatcher: отправляем задачи, но слушаем контекст
	go func() {
		defer close(jobs)
		for _, p := range proxiesList {
			select {
			case jobs <- p:
				// отправлено
			case <-ctx.Done():
				// контекст отменен, прекращаем отправку
				return
			}
		}
	}()

	// Waiter: закрывает канал результатов
	go func() {
		wg.Wait()
		close(results)
	}()

	var validProxies []ProxyItemFull
	for res := range results {
		if res.CheckResult.Error == nil {
			validProxies = append(validProxies, res)
		}
	}

	// Сортировка только если мы не были прерваны (или сортируем что успели)
	sort.Slice(validProxies, func(i, j int) bool {
		return validProxies[i].CheckResult.ReqLatency < validProxies[j].CheckResult.ReqLatency
	})

	return validProxies
}

func CheckProxy(ctx context.Context, proxyAddr, destAddr, mode string) Result {
	var res Result

	start := time.Now()
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		res.Error = fmt.Errorf("TCP: %w", err)
		return res
	}
	conn.Close()
	res.ProxyLatency = time.Since(start)
	res.ProxyLatencyStr = res.ProxyLatency.String()

	client, err := proxies.NewClient(proxyAddr, mode)
	if err != nil {
		res.Error = err
		return res
	}

	target := destAddr
	if target == "" {
		target = "http://google.com"
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", target, nil)

	start = time.Now()
	resp, err := client.Do(req)
	res.ReqLatency = time.Since(start)
	res.ReqLatencyStr = res.ReqLatency.String()

	if err != nil {
		res.Error = err
		return res
	}
	resp.Body.Close()
	res.StatusCode = resp.StatusCode
	return res
}
