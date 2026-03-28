package cli

import (
	"context"
	"fmt"
	"strings"

	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
)

// Run запускает консольный интерфейс
func Run(cfg *config.Config) {
    if cfg.ProxiesStat {
        handleProxiesList(cfg)
    } else {
        handleSingleCheck(cfg)
    }
}

func handleSingleCheck(cfg *config.Config) {
    fmt.Printf("Проверка прокси %s (тип: %s)...\n", cfg.ProxyAddr, cfg.Type)
    ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
    defer cancel()

    res := services.CheckProxy(ctx, cfg.ProxyAddr, cfg.DestAddr, cfg.Type)

    if res.Error != nil {
        fmt.Printf("[FAIL] %v\n", res.Error)
        return // Возвращаем управление, а не os.Exit, чтобы быть вежливым к библиотеке
    }

    fmt.Printf("[OK] TCP: %v | HTTP: %v | Status: %d\n", res.ProxyLatency, res.ReqLatency, res.StatusCode)
}

func handleProxiesList(cfg *config.Config) {
    targetURL, err := cfg.GetFinalURL()
    if err != nil {
        fmt.Printf("Ошибка генерации URL: %v\n", err)
        return
    }

    fmt.Printf("Режим: %s (RTT < %dms, Pages: %d)\n", cfg.Type, cfg.RTT, cfg.Pages)

    ctxParse := context.Background()
    allProxies, err := services.FetchAllPages(ctxParse, targetURL, cfg.Pages)
    if err != nil {
        fmt.Printf("Ошибка при парсинге: %v\n", err)
    }

    fmt.Printf("Найдено всего: %d прокси.\n", len(allProxies))

    if !cfg.Check {
        printTable(allProxies)
        return
    }

    fmt.Printf("Запуск проверки (Workers: %d)...\n", cfg.Workers)

    validProxies := services.CheckBatch(
        context.Background(),
        allProxies,
        cfg.DestAddr,
        cfg.Type,
        cfg.Timeout,
        cfg.Workers,
        func(current, total int32) {
            fmt.Printf("\rПрогресс: %d/%d", current, total)
        },
    )

    fmt.Println()

    if len(validProxies) == 0 {
        fmt.Println("Работоспособных прокси не найдено.")
        return
    }

    fmt.Printf("Найдено рабочих: %d\n", len(validProxies))
    printFullTable(validProxies)
}

func printTable(proxies []services.ProxyItem) {
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
