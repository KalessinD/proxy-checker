package gui

import (
	"encoding/json"
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
		Load(cfg *config.Config) []*ProxyItemWrapper
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

func (c *CacheFile) Load(cfg *config.Config) []*ProxyItemWrapper {
	cacheFilePath := c.GetFilePath()

	fileInfo, err := os.Stat(cacheFilePath)
	if err != nil { // it's not an error if there is no cache file yet
		return nil
	}

	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	if time.Since(fileInfo.ModTime()) > cacheTTL {
		return nil
	}

	fileData, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil
	}

	var cachedItems []*ProxyItemWrapper
	if err := json.Unmarshal(fileData, &cachedItems); err != nil {
		return nil
	}

	return cachedItems
}

func (c *CacheFile) Save(items []*ProxyItemWrapper) error {
	cacheFilePath := c.GetFilePath()

	fileData, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFilePath, fileData, 0o600)
}
