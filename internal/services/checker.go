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
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/proxies"
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
		ProxyLatency    time.Duration `json:"proxy_latency"`
		ProxyLatencyStr string        `json:"proxy_latency_str"`

		ReqLatency    time.Duration `json:"req_latency"`
		ReqLatencyStr string        `json:"req_latency_str"`

		StatusCode    int   `json:"status_code"`
		Error         error `json:"error,omitempty"`
		SupportsHTTP  bool  `json:"supports_http"`
		SupportsHTTP2 bool  `json:"supports_http2"`
	}

	ProxyItem = fetcher.ProxyItem

	ProxyItemFull struct {
		ProxyItem   `json:",inline"`
		CheckResult Result `json:",inline"`
	}

	// CheckParams is the narrow DTO of the network-validation settings shared
	// by the checker and the pipeline configuration.
	CheckParams struct {
		DestAddr    string
		Type        common.ProxyType
		Timeout     time.Duration
		DialTimeout time.Duration
		Workers     int
		CheckHTTP2  bool
	}

	defaultChecker struct{}

	defaultVerifier struct {
		checker *defaultChecker
	}
)

// ProxyChecker defines the port for network validation of proxies
// (single and batch). It knows nothing about where proxies came from.
type ProxyChecker interface {
	CheckProxy(ctx context.Context, proxyAddr string, params *CheckParams) Result
	CheckBatch(
		ctx context.Context,
		proxiesList []*ProxyItem,
		params *CheckParams,
		verifier ProxyVerifier,
		progressCallback func(current, total int),
	) []*ProxyItemFull
}

func NewProxyChecker() ProxyChecker {
	return &defaultChecker{}
}

// NewCheckParams projects the application configuration onto check settings.
func NewCheckParams(cfg *config.Config) *CheckParams {
	return &CheckParams{
		DestAddr:    cfg.DestAddr,
		Type:        cfg.Type,
		Timeout:     cfg.Timeout,
		DialTimeout: cfg.DialTimeout,
		Workers:     cfg.Workers,
		CheckHTTP2:  cfg.CheckHTTP2,
	}
}

func NewDefaultVerifier() ProxyVerifier {
	return &defaultVerifier{checker: &defaultChecker{}}
}

func (v *defaultVerifier) Verify(ctx context.Context, proxyAddr string, params *CheckParams) Result {
	return v.checker.checkProxy(ctx, proxyAddr, params)
}

func (pc *defaultChecker) CheckProxy(ctx context.Context, proxyAddr string, params *CheckParams) Result {
	return pc.checkProxy(ctx, proxyAddr, params)
}

func (pc *defaultChecker) CheckBatch(
	ctx context.Context,
	proxiesList []*ProxyItem,
	params *CheckParams,
	verifier ProxyVerifier,
	progressCallback func(current, total int),
) []*ProxyItemFull {
	timeout := params.Timeout
	workers := params.Workers

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

			// The proxy type may override the configured one when checking "all" types.
			checkParams := *params
			if params.Type == common.ProxyAll {
				checkParams.Type = proxy.Type
			}

			addr := fmt.Sprintf("%s:%s", proxy.Host, proxy.Port)
			ctxCheck, cancel := context.WithTimeout(ctx, timeout)

			res := verifier.Verify(ctxCheck, addr, &checkParams)
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

// ResolveSchema maps the proxy mode to the schema used for the validation request.
func ResolveSchema(mode string, forceHTTP2 bool) string {
	if forceHTTP2 {
		return httpsSchema
	}

	switch mode {
	case "http":
		return httpSchema
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

func (pc *defaultChecker) checkProxy(ctx context.Context, proxyAddr string, params *CheckParams) Result {
	var res Result

	dialTimeout := params.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = config.DefaultDialTimeout
	}
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

	target := params.DestAddr
	if target == "" {
		target = i18n.T("checker.default_target")
	}

	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")

	mode := string(params.Type)
	schema := ResolveSchema(mode, params.CheckHTTP2)
	target = schema + target

	if mode == "socks4" && params.CheckHTTP2 {
		res.Error = errors.New(i18n.T("checker.err_socks4_no_http2"))
		return res
	}

	client, err := proxies.NewClient(proxyAddr, mode, params.CheckHTTP2)
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

	if params.CheckHTTP2 && !res.SupportsHTTP2 {
		res.Error = fmt.Errorf("%s: %s", i18n.T("checker.err_http2_failed"), resp.Proto)
	}

	return res
}
