package cli

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"strings"
)

type (
	Options struct {
		ProxyAddr   string `json:"proxy_addr"`
		ProxiesStat bool   `json:"proxies_stat"`
		Check       bool   `json:"check"`
	}

	// sourceValue implements flag.Value for comma-separated sources.
	sourceValue []common.Source
)

func ParseFlags(cfg *config.Config, args []string) (*Options, error) {
	var opts Options

	fs := flag.NewFlagSet("proxy-checker", flag.ContinueOnError)

	fs.StringVar(&opts.ProxyAddr, "proxy", "", i18n.T("cli.help_proxy"))
	fs.BoolVar(&opts.ProxiesStat, "proxies-stat", false, i18n.T("cli.help_proxies_stat"))
	fs.BoolVar(&opts.Check, "check", false, i18n.T("cli.help_check"))

	fs.StringVar(&cfg.DestAddr, "dest", cfg.DestAddr, i18n.T("cli.help_dest"))
	fs.StringVar(&cfg.LogPath, "log", cfg.LogPath, i18n.T("cli.help_log"))
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, i18n.T("cli.help_timeout"))
	fs.IntVar(&cfg.Workers, "workers", cfg.Workers, i18n.T("cli.help_workers"))
	fs.StringVar((*string)(&cfg.Type), "type", string(cfg.Type), i18n.T("cli.help_type"))
	fs.Var(newSourceValue(&cfg.Sources), "source", i18n.T("cli.help_source"))
	fs.IntVar(&cfg.RTT, "rtt", cfg.RTT, i18n.T("cli.help_rtt"))
	fs.IntVar(&cfg.Pages, "pages", cfg.Pages, i18n.T("cli.help_pages"))
	fs.BoolVar(&cfg.CheckHTTP2, "http2", cfg.CheckHTTP2, i18n.T("cli.help_http2"))
	fs.StringVar(&cfg.Lang, "lang", cfg.Lang, i18n.T("cli.help_lang"))
	fs.StringVar(&cfg.GeoIPDBPath, "geoip-db", cfg.GeoIPDBPath, i18n.T("cli.help_geoip_db"))

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if opts.ProxyAddr != "" && opts.ProxiesStat {
		return nil, errors.New(i18n.T("cli.err_mutual_exclusive"))
	}

	if opts.ProxyAddr != "" {
		if _, _, err := net.SplitHostPort(opts.ProxyAddr); err != nil {
			return nil, fmt.Errorf("%s: %w", i18n.T("cli.err_invalid_addr"), err)
		}
	}

	if opts.ProxiesStat {
		if len(cfg.Sources) == 0 {
			return nil, errors.New(i18n.T("cli.err_no_sources"))
		}
		if cfg.RTT <= 0 {
			return nil, errors.New(i18n.T("cli.err_rtt_positive"))
		}
		if cfg.Workers < 1 {
			return nil, errors.New(i18n.T("cli.err_workers_min"))
		} else if cfg.Workers > common.MaxWorkers {
			return nil, fmt.Errorf("%s %d", i18n.T("cli.err_workers_max"), common.MaxWorkers)
		}
	}

	validTypes := map[common.ProxyType]bool{
		common.ProxySOCKS5: true, common.ProxySOCKS4: true,
		common.ProxyHTTP: true, common.ProxyHTTPS: true, common.ProxyAll: true,
	}
	if !validTypes[cfg.Type] {
		return nil, fmt.Errorf("%s: %s", i18n.T("cli.err_invalid_type"), cfg.Type)
	}

	for _, src := range cfg.Sources {
		if !common.IsKnownSource(src) {
			return nil, fmt.Errorf("%s: %s", i18n.T("cli.err_invalid_source"), src)
		}
	}

	return &opts, nil
}

func newSourceValue(p *[]common.Source) *sourceValue {
	return (*sourceValue)(p)
}

func (s *sourceValue) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(common.SourcesToStrings(*s), ",")
}

func (s *sourceValue) Set(val string) error {
	*s = nil
	parts := strings.Split(val, ",")
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		*s = append(*s, common.Source(trimmed))
	}
	return nil
}
