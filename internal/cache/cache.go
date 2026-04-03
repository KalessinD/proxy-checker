package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"time"
)

type FileCache struct {
	FilePath string
}

type Storage interface {
	Load(cfg *config.Config) ([]*services.ProxyItemFull, error)
	Save(items []*services.ProxyItemFull) error
	GetFilePath() string
}

func NewFileCache() Storage {
	return &FileCache{
		FilePath: filepath.Join(os.TempDir(), common.AppName+"-cache.data"),
	}
}

func (c *FileCache) GetFilePath() string {
	return c.FilePath
}

func (c *FileCache) Load(cfg *config.Config) ([]*services.ProxyItemFull, error) {
	fileInfo, err := os.Stat(c.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*services.ProxyItemFull{}, nil
		}
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	if time.Since(fileInfo.ModTime()) > cacheTTL {
		return []*services.ProxyItemFull{}, nil
	}

	fileData, err := os.ReadFile(c.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cachedItems []*services.ProxyItemFull
	if err := json.Unmarshal(fileData, &cachedItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	if cachedItems == nil {
		return []*services.ProxyItemFull{}, nil
	}

	return cachedItems, nil
}

func (c *FileCache) Save(items []*services.ProxyItemFull) error {
	fileData, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return os.WriteFile(c.FilePath, fileData, 0o600)
}
