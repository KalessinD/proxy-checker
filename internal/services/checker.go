package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"proxy-checker/internal/common"
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

type ProxyItem = fetcher.ProxyItem

type ProxyItemFull struct {
	ProxyItem
	CheckResult Result
}

func CheckBatch(
	ctx context.Context,
	proxiesList []ProxyItem,
	dest string,
	mode common.ProxyType,
	timeout time.Duration,
	workers int,
	progressCallback func(current, total int32),
) []ProxyItemFull {
	jobs := make(chan ProxyItem, len(proxiesList))
	results := make(chan ProxyItemFull, len(proxiesList))

	var wg sync.WaitGroup
	var processedCount int32
	totalCount := int32(len(proxiesList))

	worker := func() {
		defer wg.Done()
		for p := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			currentMode := mode
			if mode == common.ProxyAll {
				currentMode = p.Type
			}

			addr := fmt.Sprintf("%s:%s", p.Host, p.Port)
			ctxCheck, cancel := context.WithTimeout(ctx, timeout)
			res := CheckProxy(ctxCheck, addr, dest, string(currentMode))
			cancel()

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

	go func() {
		defer close(jobs)
		for _, p := range proxiesList {
			select {
			case jobs <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

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

	sort.Slice(validProxies, func(i, j int) bool {
		return validProxies[i].CheckResult.ReqLatency < validProxies[j].CheckResult.ReqLatency
	})

	return validProxies
}

func CheckProxy(ctx context.Context, proxyAddr, destAddr, mode string) Result {
	var res Result

	// ИСПРАВЛЕНО: Вычисляем таймаут из контекста, а не берем захардкоженный 5 секунд
	dialTimeout := 10 * time.Second // Максимальный дефолт
	if deadline, ok := ctx.Deadline(); ok {
		remain := time.Until(deadline)
		if remain > 0 && remain < dialTimeout {
			dialTimeout = remain
		}
	}

	start := time.Now()
	dialer := net.Dialer{Timeout: dialTimeout}
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

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		res.Error = fmt.Errorf("создание запроса: %w", err)
		return res
	}

	start = time.Now()
	resp, err := client.Do(req)
	res.ReqLatency = time.Since(start)
	res.ReqLatencyStr = res.ReqLatency.String()

	if err != nil {
		res.Error = err
		return res
	}

	// ИСПРАВЛЕНО: Обязательно сбрасываем тело, чтобы соединение вернулось в пул (Keep-Alive)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	res.StatusCode = resp.StatusCode
	return res
}
