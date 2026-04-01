package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"time"
)

func getCacheFilePath() string {
	return filepath.Join(os.TempDir(), common.AppName+"-cache.data")
}

func loadCache(cfg *config.Config) []ProxyItemWrapper {
	cacheFilePath := getCacheFilePath()

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

	var cachedItems []ProxyItemWrapper
	if err := json.Unmarshal(fileData, &cachedItems); err != nil {
		return nil
	}

	return cachedItems
}

func saveCache(items []ProxyItemWrapper) error {
	cacheFilePath := getCacheFilePath()

	fileData, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFilePath, fileData, 0o600)
}
