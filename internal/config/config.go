package config

import (
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Type        common.ProxyType `toml:"type"`
	Timeout     time.Duration    `toml:"timeout"`
	Workers     int              `toml:"workers"`
	DestAddr    string           `toml:"dest_addr"`
	Sources     []common.Source  `toml:"sources"`
	RTT         int              `toml:"rtt"`
	Pages       int              `toml:"pages"`
	Theme       string           `toml:"theme"`
	MinHeight   int              `toml:"min_height"`
	MinWidth    int              `toml:"min_width"`
	CheckHTTP2  bool             `toml:"check_http2"`
	LogPath     string           `toml:"log_path"`
	CacheTTL    int              `toml:"cache_ttl"`
	Lang        string           `toml:"lang"`
	GeoIPDBPath string           `toml:"geoip_db_path"`
}

func DefaultConfig() *Config {
	return &Config{
		Theme:      "system",
		MinWidth:   900,
		MinHeight:  400,
		Sources:    []common.Source{common.SourceProxyMania},
		Type:       common.ProxySOCKS5,
		Timeout:    10 * time.Second,
		Workers:    256,
		Pages:      4,
		RTT:        150,
		CheckHTTP2: false,
		DestAddr:   "google.com",
		LogPath:    common.DefaultLogPath(),
		CacheTTL:   3600,
		Lang:       "en",
	}
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".config")

	_, err = os.Stat(configDir)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0o755)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(configDir, common.AppName+".conf"), nil
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	path, err := getConfigPath()
	if err != nil {
		return cfg, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	cfg.Sources = nil

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	// Ensure we always have at least one source to prevent empty state errors
	if len(cfg.Sources) == 0 {
		cfg.Sources = []common.Source{common.SourceProxyMania}
	}

	return cfg, nil
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

func EnsureConfigExists() error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			return err
		}
		return DefaultConfig().SaveToFile()
	}

	return nil
}
