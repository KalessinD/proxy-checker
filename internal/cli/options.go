package cli

import (
	"errors"
	"flag"
	"fmt"
	"net"

	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
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
	flag.StringVar(&cfg.LogPath, "log", cfg.LogPath, "Путь к файлу журнала") // НОВОЕ
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Таймаут ожидания ответа")
	flag.IntVar(&cfg.Workers, "workers", cfg.Workers, "Количество потоков для проверки")
	flag.StringVar((*string)(&cfg.Type), "type", string(cfg.Type), "Тип прокси (socks5, socks4, http, https, all)")
	flag.StringVar((*string)(&cfg.Source), "source", string(cfg.Source), "Источник прокси (proxymania, thespeedx)")
	flag.IntVar(&cfg.RTT, "rtt", cfg.RTT, "Максимальное время отклика (мс)")
	flag.IntVar(&cfg.Pages, "pages", cfg.Pages, "Количество страниц для парсинга")
	flag.BoolVar(&cfg.CheckHTTP2, "http2", cfg.CheckHTTP2, "Проверять поддержку HTTP/2 (рекомендуется только для https/socks5)")
	flag.StringVar(&cfg.Lang, "lang", cfg.Lang, "Язык интерфейса (ru, en)")

	flag.Parse()

	if opts.ProxyAddr != "" && opts.ProxiesStat {
		return nil, errors.New(i18n.T("cli.err_mutual_exclusive"))
	}

	if opts.ProxyAddr != "" {
		if _, _, err := net.SplitHostPort(opts.ProxyAddr); err != nil {
			return nil, fmt.Errorf(i18n.T("cli.err_invalid_addr"), err)
		}
	}

	if opts.ProxiesStat {
		if cfg.RTT <= 0 {
			return nil, errors.New(i18n.T("cli.err_rtt_positive"))
		}
		if cfg.Workers < 1 {
			return nil, errors.New(i18n.T("cli.err_workers_min"))
		} else if cfg.Workers > 256 {
			return nil, errors.New(i18n.T("cli.err_workers_max"))
		}
	}

	validTypes := map[common.ProxyType]bool{
		common.ProxySOCKS5: true, common.ProxySOCKS4: true,
		common.ProxyHTTP: true, common.ProxyHTTPS: true, common.ProxyAll: true,
	}
	if !validTypes[cfg.Type] {
		return nil, fmt.Errorf(i18n.T("cli.err_invalid_type"), cfg.Type)
	}

	if cfg.Source != common.SourceProxyMania && cfg.Source != common.SourceTheSpeedX {
		return nil, fmt.Errorf(i18n.T("cli.err_invalid_source"), cfg.Source)
	}

	return &opts, nil
}
