package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
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

	Theme     string `toml:"theme"`
	MinHeight int    `toml:"min_height"`
}

// NewConfig — конструктор конфигурации.
// Реализует цепочку приоритетов:
// 1. Дефолтные значения
// 2. Значения из файла (переопределяют дефолт)
// 3. Флаги CLI (переопределяют файл и дефолт)
// 4. Валидация
func NewConfig() (*Config, error) {
	// 1. Инициализация дефолтными значениями
	cfg := DefaultConfig()

	// 2. Загрузка из файла (игнорируем ошибку "файл не найден", но обрабатываем ошибки парсинга)
	if err := cfg.loadFromFile(); err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфига: %w", err)
	}

	// 3. Парсинг флагов (переопределяют значения из файла)
	if err := cfg.parseFlags(); err != nil {
		return nil, fmt.Errorf("ошибка парсинга аргументов: %w", err)
	}

	// 4. Валидация бизнес-логики
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфига: %w", err)
	}

	return cfg, nil
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

// loadFromFile загружает конфиг из файла, если он существует
func (c *Config) loadFromFile() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Файла нет — это нормально, используем дефолты
	}

	_, err = toml.DecodeFile(path, c)
	if err != nil {
		return err
	}

	// Миграция: если Source пустой, ставим дефолт
	if c.Source == "" {
		c.Source = "proxymania"
	}

	return nil
}

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

// parseFlags определяет флаги, используя текущие значения (из файла/дефолта) как базу
func (c *Config) parseFlags() error {
	// Защита от пустых значений, которые могли прийти из файла,
	// чтобы флаги корректно отработали дефолты валидации ниже
	if c.Theme == "" {
		c.Theme = "system"
	}
	if c.MinHeight == 0 {
		c.MinHeight = 300
	}
	if c.Source == "" {
		c.Source = "proxymania"
	}

	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Таймаут ожидания ответа")
	flag.IntVar(&c.Workers, "workers", c.Workers, "Количество потоков для проверки")
	flag.StringVar(&c.Type, "type", c.Type, "Тип прокси (socks5, socks4, http, https, all)")
	flag.StringVar(&c.ProxyAddr, "proxy", c.ProxyAddr, "Адрес прокси-сервера для проверки")
	flag.StringVar(&c.DestAddr, "dest", c.DestAddr, "Адрес целевого узла")
	flag.BoolVar(&c.ProxiesStat, "proxies-stat", c.ProxiesStat, "Режим получения списка прокси")
	flag.BoolVar(&c.Check, "check", c.Check, "Проверить доступность найденных прокси")

	flag.StringVar(&c.Source, "source", c.Source, "Источник прокси (proxymania, thespeedx)")

	flag.IntVar(&c.RTT, "rtt", c.RTT, "Максимальное время отклика (мс)")
	flag.IntVar(&c.Pages, "pages", c.Pages, "Количество страниц для парсинга")
	flag.BoolVar(&c.GUI, "gui", c.GUI, "Запустить графический интерфейс")
	flag.StringVar(&c.Theme, "theme", c.Theme, "Цветовая тема GUI: light, dark, system")
	flag.IntVar(&c.MinHeight, "min-height", c.MinHeight, "Минимальная высота таблицы в пикселях (мин. 100)")

	flag.Parse()

	// Валидация значений аргументов
	validTypes := map[string]bool{
		"socks5": true, "socks4": true,
		"http": true, "https": true, "all": true,
	}
	if !validTypes[c.Type] {
		return errors.New("неверный тип прокси")
	}

	c.Theme = strings.ToLower(c.Theme)
	validThemes := map[string]bool{"light": true, "dark": true, "system": true}
	if !validThemes[c.Theme] {
		return errors.New("неверное значение темы, используйте: light, dark или system")
	}

	if c.MinHeight < 100 {
		return errors.New("минимальная высота таблицы не может быть меньше 100px")
	}

	return nil
}

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
		_, err := url.Parse("http://dummy.com")
		if err != nil {
			return fmt.Errorf("ошибка конфигурации: %w", err)
		}
		return nil
	}

	return errors.New("укажите -proxy (для проверки) или -proxies-stat (для получения списка)")
}

func DefaultConfig() *Config {
	return &Config{
		Theme:     "system",
		MinHeight: 300,
		Source:    "proxymania",
		Type:      "socks5",
		Timeout:   10 * time.Second,
		Workers:   256,
		Pages:     4,
		RTT:       150,
	}
}

func EnsureConfigExists() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		return cfg.SaveToFile()
	}

	return nil
}
