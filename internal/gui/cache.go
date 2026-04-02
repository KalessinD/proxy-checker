package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"time"
)

type (
	CacheFile struct {
		Items    []*ProxyItemWrapper `json:"items"`
		FilePath string              `json:"-"`
	}

	CacheInterface interface {
		GetFilePath() string
		Load(cfg *config.Config) ([]*ProxyItemWrapper, error)
		Save(items []*ProxyItemWrapper) error
	}
)

func NewCacheFile() CacheInterface {
	return &CacheFile{
		FilePath: filepath.Join(os.TempDir(), common.AppName+"-cache.data"),
	}
}

func (c *CacheFile) GetFilePath() string {
	return c.FilePath
}

func (c *CacheFile) Load(cfg *config.Config) ([]*ProxyItemWrapper, error) {
	cacheFilePath := c.GetFilePath()

	fileInfo, err := os.Stat(cacheFilePath)
	if err != nil { // it's not an error if there is no cache file yet
		if os.IsNotExist(err) {
			return []*ProxyItemWrapper{}, nil
		}
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}

	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	if time.Since(fileInfo.ModTime()) > cacheTTL {
		return []*ProxyItemWrapper{}, nil
	}

	fileData, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cachedItems []*ProxyItemWrapper
	if err := json.Unmarshal(fileData, &cachedItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	if cachedItems == nil {
		return []*ProxyItemWrapper{}, nil
	}

	return cachedItems, nil
}

func (c *CacheFile) Save(items []*ProxyItemWrapper) error {
	cacheFilePath := c.GetFilePath()

	fileData, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFilePath, fileData, 0o600)
}
