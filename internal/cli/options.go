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

	flag.StringVar(&opts.ProxyAddr, "proxy", "", i18n.T("cli.help_proxy"))
	flag.BoolVar(&opts.ProxiesStat, "proxies-stat", false, i18n.T("cli.help_proxies_stat"))
	flag.BoolVar(&opts.Check, "check", false, i18n.T("cli.help_check"))

	flag.StringVar(&cfg.DestAddr, "dest", cfg.DestAddr, i18n.T("cli.help_dest"))
	flag.StringVar(&cfg.LogPath, "log", cfg.LogPath, i18n.T("cli.help_log"))
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, i18n.T("cli.help_timeout"))
	flag.IntVar(&cfg.Workers, "workers", cfg.Workers, i18n.T("cli.help_workers"))
	flag.StringVar((*string)(&cfg.Type), "type", string(cfg.Type), i18n.T("cli.help_type"))
	flag.StringVar((*string)(&cfg.Source), "source", string(cfg.Source), i18n.T("cli.help_source"))
	flag.IntVar(&cfg.RTT, "rtt", cfg.RTT, i18n.T("cli.help_rtt"))
	flag.IntVar(&cfg.Pages, "pages", cfg.Pages, i18n.T("cli.help_pages"))
	flag.BoolVar(&cfg.CheckHTTP2, "http2", cfg.CheckHTTP2, i18n.T("cli.help_http2"))
	flag.StringVar(&cfg.Lang, "lang", cfg.Lang, i18n.T("cli.help_lang"))

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
