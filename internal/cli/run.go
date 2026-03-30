package cli

import (
	"context"
	"strings"

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
		zap.S().Info("Укажите действие: -proxy (для проверки) или -proxies-stat (для получения списка)")
	}
}

func handleSingleCheck(cfg *config.Config, opts *Options) {
	zap.S().Infof("Проверка прокси %s (тип: %s)...", opts.ProxyAddr, cfg.Type)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	res := services.CheckProxy(ctx, opts.ProxyAddr, cfg.DestAddr, string(cfg.Type), cfg.CheckHTTP2)
	if res.Error != nil {
		zap.S().Errorf("[FAIL] %v", res.Error)
		return
	}

	zap.S().Infof("[OK] TCP: %v | HTTP: %v | Status: %d", res.ProxyLatency, res.ReqLatency, res.StatusCode)
}

func handleProxiesList(cfg *config.Config, opts *Options) {
	zap.S().Infof("Режим: %s (Source: %s, RTT < %dms, Pages: %d)", cfg.Type, cfg.Source, cfg.RTT, cfg.Pages)

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
			zap.S().Errorf("Ошибка при парсинге: %v", err)
			return
		}

		zap.S().Infof("Найдено всего: %d прокси.", len(allProxies))
		printTable(allProxies)
		return
	}

	zap.S().Infof("Запуск проверки (Workers: %d)...", cfg.Workers)

	validProxies, err := services.RunPipeline(
		context.Background(),
		cfg,
		services.PipelineCallbacks{
			OnFetched: func(total int) {
				zap.S().Infof("Найдено всего: %d прокси.", total)
			},
			OnProgress: func(current, total int32) {
				zap.S().Infof("Прогресс: %d/%d", current, total)
			},
		},
	)

	if err != nil {
		zap.S().Errorf("Ошибка при выполнении пайплайна: %v", err)
		return
	}

	if len(validProxies) == 0 {
		zap.S().Info("Работоспособных прокси не найдено.")
		return
	}

	zap.S().Infof("Найдено рабочих: %d", len(validProxies))
	printFullTable(validProxies)
}

func printTable(proxies []fetcher.ProxyItem) {
	sep := strings.Repeat("-", 70)
	zap.S().Infof("%-25s %-6s %-10s %-15s %-10s", "Host", "Port", "Type", "Country", "RTT")
	zap.S().Info(sep)
	for _, p := range proxies {
		zap.S().Infof("%-25s %-6s %-10s %-15s %-10s", p.Host, p.Port, p.Type, p.Country, p.RTT)
	}
}

func printFullTable(proxies []services.ProxyItemFull) {
	sep := strings.Repeat("-", 95)
	zap.S().Infof("%-25s %-6s %-10s %-15s %-15s %-15s", "Host", "Port", "Type", "Country", "TCP", "HTTP")
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
