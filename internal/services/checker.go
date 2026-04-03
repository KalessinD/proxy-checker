package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/proxies"
	"proxy-checker/internal/services/fetcher"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	httpSchema  = "http://"
	httpsSchema = "https://"
)

type (
	Result struct {
		ProxyLatency    time.Duration
		ProxyLatencyStr string

		ReqLatency    time.Duration
		ReqLatencyStr string

		StatusCode int
		Error      error

		SupportsHTTP  bool
		SupportsHTTP2 bool
	}

	ProxyItem = fetcher.ProxyItem

	ProxyItemFull struct {
		ProxyItem
		CheckResult Result
	}

	// defaultVerifier implements ProxyVerifier with reale network usage
	defaultVerifier struct{}
)

func (v *defaultVerifier) Verify(ctx context.Context, proxyAddr, destAddr, mode string, checkHTTP2 bool) Result {
	return checkProxy(ctx, proxyAddr, destAddr, mode, checkHTTP2)
}

func CheckProxy(ctx context.Context, proxyAddr, destAddr, mode string, checkHTTP2 bool) Result {
	return (&defaultVerifier{}).Verify(ctx, proxyAddr, destAddr, mode, checkHTTP2)
}

func CheckBatch(
	ctx context.Context,
	proxiesList []*ProxyItem,
	dest string,
	mode common.ProxyType,
	timeout time.Duration,
	workers int,
	checkHTTP2 bool,
	progressCallback func(current, total int),
	verifier ProxyVerifier,
) []*ProxyItemFull {
	jobs := make(chan *ProxyItem, len(proxiesList))
	results := make(chan *ProxyItemFull, len(proxiesList))

	var wg sync.WaitGroup
	var processedCount int32
	totalCount := len(proxiesList)

	worker := func() {
		defer wg.Done()
		for proxy := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			currentMode := mode
			if mode == common.ProxyAll {
				currentMode = proxy.Type
			}

			addr := fmt.Sprintf("%s:%s", proxy.Host, proxy.Port)
			ctxCheck, cancel := context.WithTimeout(ctx, timeout)

			res := verifier.Verify(ctxCheck, addr, dest, string(currentMode), checkHTTP2)
			cancel()

			if ctx.Err() != nil {
				return
			}

			results <- &ProxyItemFull{ProxyItem: *proxy, CheckResult: res}

			if progressCallback != nil {
				current := int(atomic.AddInt32(&processedCount, 1))
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

	var validProxies []*ProxyItemFull
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

func ResolveSchema(mode string, forceHTTP2 bool) string {
	if forceHTTP2 {
		return httpsSchema
	}

	switch mode {
	case "http":
		return "http://"
	case "https":
		return httpsSchema
	case "socks4":
		return httpSchema
	case "socks5":
		return httpsSchema
	default:
		return httpSchema
	}
}

func checkProxy(ctx context.Context, proxyAddr, destAddr, mode string, checkHTTP2 bool) Result {
	var res Result

	dialTimeout := 10 * time.Second
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
		res.Error = fmt.Errorf("%s: %w", i18n.T("checker.err_tcp"), err)
		return res
	}
	conn.Close()

	res.ProxyLatency = time.Since(start)
	res.ProxyLatencyStr = res.ProxyLatency.String()

	target := destAddr
	if target == "" {
		target = i18n.T("checker.default_target")
	}

	// Защита от случая, если пользователь вручную ввел схему в настройках/GUI
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")

	schema := ResolveSchema(mode, checkHTTP2)
	target = schema + target

	if mode == "socks4" && checkHTTP2 {
		res.Error = errors.New(i18n.T("checker.err_socks4_no_http2"))
		return res
	}

	client, err := proxies.NewClient(proxyAddr, mode, checkHTTP2)
	if err != nil {
		res.Error = err
		return res
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		res.Error = fmt.Errorf("%s: %w", i18n.T("checker.err_create_req"), err)
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
	defer resp.Body.Close()

	_, _ = io.CopyN(io.Discard, resp.Body, 512)

	res.StatusCode = resp.StatusCode
	res.SupportsHTTP = true
	res.SupportsHTTP2 = resp.ProtoMajor == 2

	if checkHTTP2 && !res.SupportsHTTP2 {
		res.Error = fmt.Errorf("%s: %s", i18n.T("checker.err_http2_failed"), resp.Proto)
	}

	return res
}
