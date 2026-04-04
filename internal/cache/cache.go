package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/services"
	"sync"
	"time"
)

type (
	Record struct {
		ExpireAt int64                     `json:"expire_at"`
		Data     []*services.ProxyItemFull `json:"data"`
	}

	Data struct {
		Sources map[string]*Record `json:"sources"`
	}

	FileStorage struct {
		FilePath string
		Logger   common.LoggerInterface
		mu       sync.Mutex
	}
)

type StorageInterface interface {
	Load(source common.Source, proxyType common.ProxyType) ([]*services.ProxyItemFull, error)
	Save(source common.Source, proxyType common.ProxyType, items []*services.ProxyItemFull, ttl int) error
	GetFilePath() string
}

func NewFileStorage(logger common.LoggerInterface) StorageInterface {
	return &FileStorage{
		FilePath: filepath.Join(os.TempDir(), common.AppName+"-cache.data"),
		Logger:   logger,
	}
}

func (c *FileStorage) GetFilePath() string {
	return c.FilePath
}

func generateCacheKey(source common.Source, proxyType common.ProxyType) string {
	return string(source) + ":" + string(proxyType)
}

func groupItemsByType(items []*services.ProxyItemFull) map[common.ProxyType][]*services.ProxyItemFull {
	groups := make(map[common.ProxyType][]*services.ProxyItemFull)
	for _, item := range items {
		groups[item.Type] = append(groups[item.Type], item)
	}
	return groups
}

func mergeAndDeduplicate(slices ...[]*services.ProxyItemFull) []*services.ProxyItemFull {
	seen := make(map[string]struct{})
	var result []*services.ProxyItemFull
	for _, slice := range slices {
		for _, item := range slice {
			key := item.Host + ":" + item.Port
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				result = append(result, item)
			}
		}
	}
	return result
}

func (c *FileStorage) Load(source common.Source, proxyType common.ProxyType) ([]*services.ProxyItemFull, error) {
	if proxyType == common.ProxyAll {
		return c.loadAllTypes(source)
	}
	return c.loadSingle(source, proxyType)
}

func (c *FileStorage) loadAllTypes(source common.Source) ([]*services.ProxyItemFull, error) {
	var allItems []*services.ProxyItemFull
	typesToLoad := []common.ProxyType{common.ProxyHTTP, common.ProxyHTTPS, common.ProxySOCKS4, common.ProxySOCKS5}

	for _, pt := range typesToLoad {
		items, err := c.loadSingle(source, pt)
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, items...)
	}

	return mergeAndDeduplicate(allItems), nil
}

func (c *FileStorage) loadSingle(source common.Source, proxyType common.ProxyType) ([]*services.ProxyItemFull, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheData, err := c.readCacheData()
	if err != nil {
		return nil, err
	}

	key := generateCacheKey(source, proxyType)
	entry, exists := cacheData.Sources[key]
	if !exists {
		return []*services.ProxyItemFull{}, nil
	}

	if time.Now().Unix() > entry.ExpireAt {
		return []*services.ProxyItemFull{}, nil
	}

	if entry.Data == nil {
		return []*services.ProxyItemFull{}, nil
	}

	return entry.Data, nil
}

func (c *FileStorage) Save(source common.Source, proxyType common.ProxyType, items []*services.ProxyItemFull, ttl int) error {
	if proxyType == common.ProxyAll {
		return c.saveSplitByType(source, items, ttl)
	}
	return c.saveSingle(source, proxyType, items, ttl)
}

func (c *FileStorage) saveSplitByType(source common.Source, items []*services.ProxyItemFull, ttl int) error {
	groups := groupItemsByType(items)
	for pt, groupItems := range groups {
		if err := c.saveSingle(source, pt, groupItems, ttl); err != nil {
			return err
		}
	}
	return nil
}

func (c *FileStorage) saveSingle(source common.Source, proxyType common.ProxyType, items []*services.ProxyItemFull, ttl int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheData, err := c.readCacheData()
	if err != nil {
		return err
	}

	key := generateCacheKey(source, proxyType)
	cacheData.Sources[key] = &Record{
		ExpireAt: time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
		Data:     items,
	}

	return c.writeCacheData(cacheData)
}

func (c *FileStorage) readCacheData() (*Data, error) {
	fileData, err := os.ReadFile(c.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Data{Sources: make(map[string]*Record)}, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cacheData Data
	if err := json.Unmarshal(fileData, &cacheData); err != nil {
		c.Logger.Warnf("failed to unmarshal cache file: %v", err)
		return &Data{Sources: make(map[string]*Record)}, nil
	}

	if cacheData.Sources == nil {
		cacheData.Sources = make(map[string]*Record)
	}

	return &cacheData, nil
}

func (c *FileStorage) writeCacheData(cacheData *Data) error {
	newData, err := json.Marshal(cacheData)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}
	return os.WriteFile(c.FilePath, newData, 0o600)
}
