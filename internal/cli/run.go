package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"strings"
	"syscall"
)

func Run(cfg *config.Config, opts *Options, logger common.LoggerInterface) {
	switch {
	case opts.ProxiesStat:
		handleProxiesList(cfg, opts, logger)
	case opts.ProxyAddr != "":
		handleSingleCheck(cfg, opts, logger)
	default:
		fmt.Println(i18n.T("cli.err_no_action"))
	}
}

func handleSingleCheck(cfg *config.Config, opts *Options, _ common.LoggerInterface) {
	fmt.Printf("%s: %s (%s)...\n", i18n.T("cli.checking"), opts.ProxyAddr, cfg.Type)
	ctx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer sigCancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer timeoutCancel()

	checker := services.NewProxyChecker()
	res := checker.CheckProxy(ctx, opts.ProxyAddr, cfg.DestAddr, string(cfg.Type), cfg.CheckHTTP2)
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", i18n.T("cli.fail"), res.Error)
		return
	}

	fmt.Printf("%s: TCP: %s, HTTP: %s, %s: %d\n", i18n.T("cli.ok"), res.ProxyLatency, res.ReqLatency, i18n.T("cli.status_label"), res.StatusCode)
}

func handleProxiesList(cfg *config.Config, opts *Options, logger common.LoggerInterface) {
	fmt.Printf(
		"%s %s: %s, %s: %s, %s: %d, %s: %d\n",
		i18n.T("cli.mode_start"),
		i18n.T("cli.mode_type"),
		cfg.Type,
		i18n.T("cli.mode_source"),
		cfg.Source,
		i18n.T("cli.mode_rtt"),
		cfg.RTT,
		i18n.T("cli.mode_pages"),
		cfg.Pages,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if !opts.Check {
		handleListFetch(ctx, cfg, logger)
	} else {
		handleListCheck(ctx, cfg, logger)
	}
}

// handleListFetch handles fetching the list only, without validation
func handleListFetch(ctx context.Context, cfg *config.Config, logger common.LoggerInterface) {
	fetcherInstance := fetcher.NewFetcher(cfg.Source, logger)

	settings := fetcher.Settings{
		Type:   cfg.Type,
		MaxRTT: cfg.RTT,
		Pages:  cfg.Pages,
	}

	allProxies, err := fetcherInstance.Fetch(ctx, settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("cli.parse_error"), err)
		return
	}

	fmt.Printf("%s: %d\n", i18n.T("cli.total_found"), len(allProxies))
	PrintTable(allProxies)
}

// handleListCheck handles fetching and validation of the list
func handleListCheck(ctx context.Context, cfg *config.Config, logger common.LoggerInterface) {
	fmt.Printf("%s (Workers: %d)...\n", i18n.T("cli.starting_workers"), cfg.Workers)

	var resolver common.GeoIPResolver

	if len(common.GeoIPData) > 0 {
		if r, err := common.NewMaxMindDBResolverFromBytes(common.GeoIPData); err == nil {
			resolver = r
		}
	}

	if resolver == nil && cfg.GeoIPDBPath != "" {
		if r, err := common.NewMaxMindDBResolverFromFile(cfg.GeoIPDBPath); err == nil {
			resolver = r
		}
	}

	verifierInstance := services.NewDefaultVerifier()
	fetcherInstance := fetcher.NewFetcher(cfg.Source, logger)

	validProxies, err := services.RunPipeline(
		ctx,
		fetcherInstance,
		verifierInstance,
		cfg,
		resolver,
		services.PipelineCallbacks{
			OnFetched: func(total int) {
				fmt.Printf("%s: %d\n", i18n.T("cli.total_found"), total)
			},
			OnProgress: func(current, total int) {
				fmt.Printf("%s: %d/%d\n", i18n.T("cli.progress"), current, total)
			},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("cli.pipeline_error"), err)
		return
	}

	if resolver != nil {
		defer resolver.Close()
	}

	if len(validProxies) == 0 {
		fmt.Println(i18n.T("cli.no_valid"))
		return
	}

	fmt.Printf("%s: %d\n", i18n.T("cli.valid_found"), len(validProxies))
	PrintFullTable(validProxies)
}

func PrintTable(proxies []*fetcher.ProxyItem) {
	sep := strings.Repeat("-", 70)
	fmt.Printf(
		"%-25s %-6s %-10s %-15s %-10s\n",
		i18n.T("cli.table_host"),
		i18n.T("cli.table_port"),
		i18n.T("cli.table_type"),
		i18n.T("cli.table_country"),
		i18n.T("cli.table_rtt"),
	)
	fmt.Println(sep)
	for _, p := range proxies {
		fmt.Printf("%-25s %-6s %-10s %-15s %-10s\n", p.Host, p.Port, p.Type, p.Country, p.RTT)
	}
}

func PrintFullTable(proxies []*services.ProxyItemFull) {
	sep := strings.Repeat("-", 95)
	fmt.Printf(
		"%-25s %-6s %-10s %-15s %-15s %-15s\n",
		i18n.T("cli.table_host"),
		i18n.T("cli.table_port"),
		i18n.T("cli.table_type"),
		i18n.T("cli.table_country"),
		i18n.T("cli.table_tcp"),
		i18n.T("cli.table_http"),
	)
	fmt.Println(sep)

	for _, p := range proxies {
		fmt.Printf("%-25s %-6s %-10s %-15s %-15s %-15s\n",
			p.Host,
			p.Port,
			p.Type,
			p.Country,
			p.CheckResult.ProxyLatencyStr,
			p.CheckResult.ReqLatencyStr,
		)
	}
}
