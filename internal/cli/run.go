package cli

import (
	"context"
	"strings"

	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"proxy-checker/internal/services/fetcher"

	"go.uber.org/zap"
)

func Run(cfg *config.Config, opts *Options) {
	switch {
	case opts.ProxiesStat:
		handleProxiesList(cfg, opts)
	case opts.ProxyAddr != "":
		handleSingleCheck(cfg, opts)
	default:
		zap.S().Info(i18n.T("cli.err_no_action"))
	}
}

func handleSingleCheck(cfg *config.Config, opts *Options) {
	zap.S().Infof(i18n.T("cli.checking"), opts.ProxyAddr, cfg.Type)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	res := services.CheckProxy(ctx, opts.ProxyAddr, cfg.DestAddr, string(cfg.Type), cfg.CheckHTTP2)
	if res.Error != nil {
		zap.S().Errorf(i18n.T("cli.fail"), res.Error)
		return
	}

	zap.S().Infof(i18n.T("cli.ok"), res.ProxyLatency, res.ReqLatency, res.StatusCode)
}

func handleProxiesList(cfg *config.Config, opts *Options) {
	zap.S().Infof(i18n.T("cli.mode"), cfg.Type, cfg.Source, cfg.RTT, cfg.Pages)

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
			zap.S().Errorf(i18n.T("cli.parse_error"), err)
			return
		}

		zap.S().Infof(i18n.T("cli.total_found"), len(allProxies))
		printTable(allProxies)
		return
	}

	zap.S().Infof(i18n.T("cli.starting_workers"), cfg.Workers)

	validProxies, err := services.RunPipeline(
		context.Background(),
		cfg,
		services.PipelineCallbacks{
			OnFetched: func(total int) {
				zap.S().Infof(i18n.T("cli.total_found"), total)
			},
			OnProgress: func(current, total int32) {
				zap.S().Infof(i18n.T("cli.progress"), current, total)
			},
		},
	)

	if err != nil {
		zap.S().Errorf(i18n.T("cli.pipeline_error"), err)
		return
	}

	if len(validProxies) == 0 {
		zap.S().Info(i18n.T("cli.no_valid"))
		return
	}

	zap.S().Infof(i18n.T("cli.valid_found"), len(validProxies))
	printFullTable(validProxies)
}

func printTable(proxies []fetcher.ProxyItem) {
	sep := strings.Repeat("-", 70)
	zap.S().Infof("%-25s %-6s %-10s %-15s %-10s", i18n.T("cli.table_host"), i18n.T("cli.table_port"), i18n.T("cli.table_type"), i18n.T("cli.table_country"), i18n.T("cli.table_rtt"))
	zap.S().Info(sep)
	for _, p := range proxies {
		zap.S().Infof("%-25s %-6s %-10s %-15s %-10s", p.Host, p.Port, p.Type, p.Country, p.RTT)
	}
}

func printFullTable(proxies []services.ProxyItemFull) {
	sep := strings.Repeat("-", 95)
	zap.S().Infof("%-25s %-6s %-10s %-15s %-15s %-15s", i18n.T("cli.table_host"), i18n.T("cli.table_port"), i18n.T("cli.table_type"), i18n.T("cli.table_country"), i18n.T("cli.table_tcp"), i18n.T("cli.table_http"))
	zap.S().Info(sep)

	for _, p := range proxies {
		zap.S().Infof("%-25s %-6s %-10s %-15s %-15s %-15s",
			p.Host,
			p.Port,
			p.Type,
			p.Country,
			p.CheckResult.ProxyLatencyStr,
			p.CheckResult.ReqLatencyStr,
		)
	}
}
