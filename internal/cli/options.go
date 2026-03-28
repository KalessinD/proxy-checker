package cli

import (
	"flag"
	"fmt"
	"net"

	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
)

// Options содержит флаги, которые актуальны только для CLI
type Options struct {
	ProxyAddr   string
	ProxiesStat bool
	Check       bool
}

// ParseFlags парсит аргументы командной строки и валидирует их.
// Принимает *config.Config, чтобы пользователь мог переопределить настройки из файла через флаги (опционально).
func ParseFlags(cfg *config.Config) (*Options, error) {
	var opts Options

	// Привязываем флаги. Дефолтные значения берем из загруженного cfg (если нужно)
	flag.StringVar(&opts.ProxyAddr, "proxy", "", "Адрес прокси-сервера для проверки (host:port)")
	flag.BoolVar(&opts.ProxiesStat, "proxies-stat", false, "Режим получения списка прокси")
	flag.BoolVar(&opts.Check, "check", false, "Проверить доступность найденных прокси")

	// Разрешаем переопределять файловые настройки через CLI (опционально, но удобно)
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Таймаут ожидания ответа")
	flag.IntVar(&cfg.Workers, "workers", cfg.Workers, "Количество потоков для проверки")
	flag.StringVar((*string)(&cfg.Type), "type", string(cfg.Type), "Тип прокси (socks5, socks4, http, https, all)")
	flag.StringVar((*string)(&cfg.Source), "source", string(cfg.Source), "Источник прокси (proxymania, thespeedx)")
	flag.IntVar(&cfg.RTT, "rtt", cfg.RTT, "Максимальное время отклика (мс)")
	flag.IntVar(&cfg.Pages, "pages", cfg.Pages, "Количество страниц для парсинга")

	flag.Parse()

	// Валидация ТОЛЬКО CLI логики
	if opts.ProxyAddr != "" && opts.ProxiesStat {
		return nil, fmt.Errorf("нельзя одновременно использовать -proxy и -proxies-stat")
	}

	if opts.ProxyAddr != "" {
		if cfg.DestAddr == "" {
			return nil, fmt.Errorf("для проверки прокси (-proxy) необходимо указать -dest в конфиге")
		}
		if _, _, err := net.SplitHostPort(opts.ProxyAddr); err != nil {
			return nil, fmt.Errorf("некорректный формат адреса прокси: %w", err)
		}
	}

	if opts.ProxiesStat {
		if cfg.RTT <= 0 {
			return nil, fmt.Errorf("rtt должен быть больше 0")
		}
		if cfg.Workers < 1 {
			return nil, fmt.Errorf("workers должен быть не меньше 1")
		} else if cfg.Workers > 256 {
			return nil, fmt.Errorf("workers должен быть не больше 256")
		}
	}

	// Проверка типа (так как мы приняли его как строку через flag)
	validTypes := map[common.ProxyType]bool{
		common.ProxySOCKS5: true, common.ProxySOCKS4: true,
		common.ProxyHTTP: true, common.ProxyHTTPS: true, common.ProxyAll: true,
	}
	if !validTypes[cfg.Type] {
		return nil, fmt.Errorf("неверный тип прокси: %s", cfg.Type)
	}

	// Проверка источника
	if cfg.Source != common.SourceProxyMania && cfg.Source != common.SourceTheSpeedX {
		return nil, fmt.Errorf("неверный источник: %s", cfg.Source)
	}

	return &opts, nil
}
