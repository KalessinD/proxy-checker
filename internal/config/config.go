package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Type      string        `toml:"type"`
	Timeout   time.Duration `toml:"timeout"`
	Workers   int           `toml:"workers"`
	ProxyAddr string        `toml:"-"`
	DestAddr  string        `toml:"-"`

	ProxiesStat bool `toml:"-"`
	Check       bool `toml:"-"`
	GUI         bool `toml:"-"`

	Source string `toml:"source"`
	RTT    int    `toml:"rtt"`
	Pages  int    `toml:"pages"`

	// Новые поля
	Theme     string `toml:"theme"`
	MinHeight int    `toml:"min_height"`
}

// getConfigPath возвращает путь к файлу конфигурации
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, ".config")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}
	return filepath.Join(configDir, "proxy-checker.conf"), nil
}

// LoadFromFile загружает конфигурацию из файла
func (c *Config) LoadFromFile() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	_, err = toml.DecodeFile(path, c)
	return err
}

// SaveToFile сохраняет конфигурацию в файл
func (c *Config) SaveToFile() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(c)
}

// GetFinalURL формирует итоговый URL для парсинга с параметрами
func (c *Config) GetFinalURL() (string, error) {
	u, err := url.Parse(c.Source)
	if err != nil {
		return "", err
	}
	q := u.Query()

	// Обработка типа
	if c.Type == "all" {
		q.Del("type")
	} else {
		// Маппинг для URL (нужны заглавные буквы: SOCKS5, HTTP)
		typeMap := map[string]string{
			"socks5": "SOCKS5", "socks4": "SOCKS4",
			"http": "HTTP", "https": "HTTPS",
		}
		if t, ok := typeMap[c.Type]; ok {
			q.Set("type", t)
		}
	}

	q.Set("speed", strconv.Itoa(c.RTT))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Parse считывает аргументы командной строки.
func (c *Config) Parse() error {
	// Дефолты
	if c.Theme == "" {
		c.Theme = "system"
	}
	if c.MinHeight == 0 {
		c.MinHeight = 300 // Значение по умолчанию изменено на 300
	}

	// Флаги
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Таймаут ожидания ответа")
	flag.IntVar(&c.Workers, "workers", c.Workers, "Количество потоков для проверки")
	flag.StringVar(&c.Type, "type", c.Type, "Тип прокси (socks5, socks4, http, https, all)")
	flag.StringVar(&c.ProxyAddr, "proxy", c.ProxyAddr, "Адрес прокси-сервера для проверки")
	flag.StringVar(&c.DestAddr, "dest", c.DestAddr, "Адрес целевого узла")
	flag.BoolVar(&c.ProxiesStat, "proxies-stat", c.ProxiesStat, "Режим получения списка прокси")
	flag.BoolVar(&c.Check, "check", c.Check, "Проверить доступность найденных прокси")
	flag.StringVar(&c.Source, "url", c.Source, "URL источника прокси")
	flag.IntVar(&c.RTT, "rtt", c.RTT, "Максимальное время отклика (мс)")
	flag.IntVar(&c.Pages, "pages", c.Pages, "Количество страниц для парсинга")
	flag.BoolVar(&c.GUI, "gui", c.GUI, "Запустить графический интерфейс")

	// Новые флаги
	flag.StringVar(&c.Theme, "theme", c.Theme, "Цветовая тема GUI: light, dark, system")
	flag.IntVar(&c.MinHeight, "min-height", c.MinHeight, "Минимальная высота таблицы в пикселях (мин. 100)")

	flag.Parse()

	// Валидация типа прокси
	validTypes := map[string]bool{
		"socks5": true, "socks4": true,
		"http": true, "https": true, "all": true,
	}
	if !validTypes[c.Type] {
		return errors.New("неверный тип прокси")
	}

	// Валидация темы
	c.Theme = strings.ToLower(c.Theme)
	validThemes := map[string]bool{"light": true, "dark": true, "system": true}
	if !validThemes[c.Theme] {
		return errors.New("неверное значение темы, используйте: light, dark или system")
	}

	// Валидация высоты
	if c.MinHeight < 100 {
		return errors.New("минимальная высота таблицы не может быть меньше 100px")
	}

	return nil
}

// Validate проверяет логику работы.
// Для GUI режима проверка не требуется, так как параметры задаются в интерфейсе.
func (c *Config) Validate() error {
	if c.GUI {
		return nil
	}

	if c.ProxyAddr != "" && c.ProxiesStat {
		return errors.New("нельзя одновременно использовать -proxy и -proxies-stat")
	}

	if c.ProxyAddr != "" {
		if c.DestAddr == "" {
			return errors.New("для проверки прокси (-proxy) необходимо указать -dest")
		}
		_, _, err := net.SplitHostPort(c.ProxyAddr)
		if err != nil {
			return fmt.Errorf("некорректный формат адреса прокси: %w", err)
		}
		return nil
	}

	if c.ProxiesStat {
		if c.RTT <= 0 {
			return errors.New("rtt должен быть больше 0")
		}
		if c.Workers < 1 {
			return errors.New("workers должен быть не меньше 1")
		}
		_, err := url.Parse(c.Source)
		if err != nil {
			return fmt.Errorf("некорректный URL: %w", err)
		}
		return nil
	}

	return errors.New("укажите -proxy (для проверки) или -proxies-stat (для получения списка)")
}
