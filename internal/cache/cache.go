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
	Load(source string) ([]*services.ProxyItemFull, error)
	Save(source string, items []*services.ProxyItemFull, ttl int) error
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

func (c *FileStorage) Load(source string) ([]*services.ProxyItemFull, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fileData, err := os.ReadFile(c.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*services.ProxyItemFull{}, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cacheFile Data
	if err := json.Unmarshal(fileData, &cacheFile); err != nil {
		c.Logger.Warnf("failed to unmarshal cache file: %v", err)
		return []*services.ProxyItemFull{}, nil
	}

	if cacheFile.Sources == nil {
		return []*services.ProxyItemFull{}, nil
	}

	entry, exists := cacheFile.Sources[source]
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

func (c *FileStorage) Save(source string, items []*services.ProxyItemFull, ttl int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cacheFile Data
	fileData, err := os.ReadFile(c.FilePath)
	if err == nil {
		_ = json.Unmarshal(fileData, &cacheFile)
	}

	if cacheFile.Sources == nil {
		cacheFile.Sources = make(map[string]*Record)
	}

	cacheFile.Sources[source] = &Record{
		ExpireAt: time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
		Data:     items,
	}

	newData, err := json.Marshal(cacheFile)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	return os.WriteFile(c.FilePath, newData, 0o600)
}
