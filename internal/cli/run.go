package cli

import (
	"context"
	"fmt"
	"strings"

	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"proxy-checker/internal/services/fetcher"
)

func Run(cfg *config.Config, opts *Options) {
	switch {
	case opts.ProxiesStat:
		handleProxiesList(cfg, opts)
	case opts.ProxyAddr != "":
		handleSingleCheck(cfg, opts)
	default:
		println("Укажите действие: -proxy (для проверки) или -proxies-stat (для получения списка)")
	}
}

func handleSingleCheck(cfg *config.Config, opts *Options) {
	fmt.Printf("Проверка прокси %s (тип: %s)...\n", opts.ProxyAddr, cfg.Type)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	res := services.CheckProxy(ctx, opts.ProxyAddr, cfg.DestAddr, string(cfg.Type), cfg.CheckHTTP2)
	if res.Error != nil {
		fmt.Printf("[FAIL] %v\n", res.Error)
		return
	}

	fmt.Printf("[OK] TCP: %v | HTTP: %v | Status: %d\n", res.ProxyLatency, res.ReqLatency, res.StatusCode)
}

func handleProxiesList(cfg *config.Config, opts *Options) {
	fmt.Printf("Режим: %s (Source: %s, RTT < %dms, Pages: %d)\n", cfg.Type, cfg.Source, cfg.RTT, cfg.Pages)

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
			fmt.Printf("Ошибка при парсинге: %v\n", err)
			return
		}

		fmt.Printf("Найдено всего: %d прокси.\n", len(allProxies))
		printTable(allProxies)
		return
	}

	fmt.Printf("Запуск проверки (Workers: %d)...\n", cfg.Workers)

	validProxies, err := services.RunPipeline(
		context.Background(),
		cfg,
		services.PipelineCallbacks{
			OnFetched: func(total int) {
				fmt.Printf("Найдено всего: %d прокси.\n", total)
			},
			OnProgress: func(current, total int32) {
				fmt.Printf("\rПрогресс: %d/%d", current, total)
			},
		},
	)

	fmt.Println()

	if err != nil {
		fmt.Printf("Ошибка при выполнении пайплайна: %v\n", err)
		return
	}

	if len(validProxies) == 0 {
		fmt.Println("Работоспособных прокси не найдено.")
		return
	}

	fmt.Printf("Найдено рабочих: %d\n", len(validProxies))
	printFullTable(validProxies)
}

func printTable(proxies []fetcher.ProxyItem) {
	fmt.Println()
	fmt.Printf("%-25s %-6s %-10s %-15s %-10s\n", "Host", "Port", "Type", "Country", "RTT")
	fmt.Println(strings.Repeat("-", 70))
	for _, p := range proxies {
		fmt.Printf("%-25s %-6s %-10s %-15s %-10s\n", p.Host, p.Port, p.Type, p.Country, p.RTT)
	}
}

func printFullTable(proxies []services.ProxyItemFull) {
	fmt.Println()
	fmt.Printf("%-25s %-6s %-10s %-15s %-15s %-15s\n", "Host", "Port", "Type", "Country", "TCP", "HTTP")
	fmt.Println(strings.Repeat("-", 95))

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
