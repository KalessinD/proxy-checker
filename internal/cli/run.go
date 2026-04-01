package cli

import (
	"context"
	"fmt"
	"os"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"proxy-checker/internal/services/fetcher"
	"strings"
)

func Run(cfg *config.Config, opts *Options) {
	switch {
	case opts.ProxiesStat:
		handleProxiesList(cfg, opts)
	case opts.ProxyAddr != "":
		handleSingleCheck(cfg, opts)
	default:
		fmt.Println(i18n.T("cli.err_no_action"))
	}
}

func handleSingleCheck(cfg *config.Config, opts *Options) {
	fmt.Printf(i18n.T("cli.checking")+"\n", opts.ProxyAddr, cfg.Type)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	res := services.CheckProxy(ctx, opts.ProxyAddr, cfg.DestAddr, string(cfg.Type), cfg.CheckHTTP2)
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, i18n.T("cli.fail")+"\n", res.Error)
		return
	}

	fmt.Printf(i18n.T("cli.ok")+"\n", res.ProxyLatency, res.ReqLatency, res.StatusCode)
}

func handleProxiesList(cfg *config.Config, opts *Options) {
	fmt.Printf(i18n.T("cli.mode")+"\n", cfg.Type, cfg.Source, cfg.RTT, cfg.Pages)

	if !opts.Check {
		f := services.NewFetcher(cfg.Source)
		settings := fetcher.Settings{
			Type:   cfg.Type,
			MaxRTT: cfg.RTT,
			Pages:  cfg.Pages,
		}

		ctxParse := context.Background()
		allProxies, err := f.Fetch(ctxParse, settings)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("cli.parse_error")+"\n", err)
			return
		}

		fmt.Printf(i18n.T("cli.total_found")+"\n", len(allProxies))
		printTable(allProxies)
		return
	}

	fmt.Printf(i18n.T("cli.starting_workers")+"\n", cfg.Workers)

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

	validProxies, err := services.RunPipeline(
		context.Background(),
		cfg,
		resolver,
		services.PipelineCallbacks{
			OnFetched: func(total int) {
				fmt.Printf(i18n.T("cli.total_found")+"\n", total)
			},
			OnProgress: func(current, total int) {
				fmt.Printf(i18n.T("cli.progress")+"\n", current, total)
			},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("cli.pipeline_error")+"\n", err)
		return
	}

	if resolver != nil {
		defer resolver.Close()
	}

	if len(validProxies) == 0 {
		fmt.Println(i18n.T("cli.no_valid"))
		return
	}

	fmt.Printf(i18n.T("cli.valid_found")+"\n", len(validProxies))
	printFullTable(validProxies)
}

func printTable(proxies []fetcher.ProxyItem) {
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

func printFullTable(proxies []services.ProxyItemFull) {
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
