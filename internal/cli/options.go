package cli

import (
	"errors"
	"flag"
	"fmt"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
)

type Options struct {
	ProxyAddr   string
	ProxiesStat bool
	Check       bool
}

func ParseFlags(cfg *config.Config) (*Options, error) {
	var opts Options

	flag.StringVar(&opts.ProxyAddr, "proxy", "", "Адрес прокси-сервера для проверки (host:port)")
	flag.BoolVar(&opts.ProxiesStat, "proxies-stat", false, "Режим получения списка прокси")
	flag.BoolVar(&opts.Check, "check", false, "Проверить доступность найденных прокси")
	flag.StringVar(&cfg.DestAddr, "dest", cfg.DestAddr, "Целевой сайт для проверки (без схемы, например google.com)")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Таймаут ожидания ответа")

	if opts.ProxyAddr != "" && opts.ProxiesStat {
		return nil, errors.New("нельзя одновременно использовать -proxy и -proxies-stat")
	}

	if opts.ProxiesStat {
		if cfg.RTT <= 0 {
			return nil, errors.New("rtt должен быть больше 0")
		}
		if cfg.Workers < 1 {
			return nil, errors.New("workers должен быть не меньше 1")
		} else if cfg.Workers > 256 {
			return nil, errors.New("workers должен быть не больше 256")
		}
	}

	validTypes := map[common.ProxyType]bool{
		common.ProxySOCKS5: true, common.ProxySOCKS4: true,
		common.ProxyHTTP: true, common.ProxyHTTPS: true, common.ProxyAll: true,
	}
	if !validTypes[cfg.Type] {
		return nil, fmt.Errorf("неверный тип прокси: %s", cfg.Type)
	}

	if cfg.Source != common.SourceProxyMania && cfg.Source != common.SourceTheSpeedX {
		return nil, fmt.Errorf("неверный источник: %s", cfg.Source)
	}

	return &opts, nil
}
