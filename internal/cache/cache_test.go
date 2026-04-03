package cache_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/services"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Компиляторная проверка того, что структура реализует интерфейс
var _ cache.Storage = (*cache.FileCache)(nil)

func TestCacheFile_GetFilePath(t *testing.T) {
	customPath := "/tmp/my_custom_cache.data"
	cache := &cache.FileCache{FilePath: customPath}

	actualPath := cache.GetFilePath()
	assert.Equal(t, customPath, actualPath, "GetFilePath должен возвращать установленный путь")
}

func TestCacheFile_NewFileCache(t *testing.T) {
	cache := cache.NewFileCache()
	require.NotNil(t, cache)

	assert.Contains(t, cache.GetFilePath(), common.AppName+"-cache.data")
	assert.Contains(t, cache.GetFilePath(), os.TempDir())
}

func TestCacheFile_SaveAndLoad_ValidData(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "valid_cache.data")
	cache := &cache.FileCache{FilePath: tempCacheFile}

	inputItems := []*services.ProxyItemFull{
		{
			ProxyItem: services.ProxyItem{
				Host:  "1.1.1.1",
				Port:  "8080",
				Type:  common.ProxyHTTP,
				RTT:   "",
				RTTms: 0,
			},
			CheckResult: services.Result{},
		},
		{
			ProxyItem: services.ProxyItem{
				Host:    "2.2.2.2",
				Port:    "3128",
				Type:    common.ProxySOCKS5,
				Country: "US",
				RTT:     "",
				RTTms:   0,
			},
			CheckResult: services.Result{},
		},
	}

	err := cache.Save(inputItems)
	require.NoError(t, err)
	assert.FileExists(t, tempCacheFile)

	cfg := &config.Config{CacheTTL: 3600}
	loadedItems, err := cache.Load(cfg)
	require.NoError(t, err)

	require.Len(t, loadedItems, 2, "Должно быть загружено 2 элемента")
	assert.Equal(t, "1.1.1.1", loadedItems[0].Host)
	assert.Equal(t, common.ProxySOCKS5, loadedItems[1].Type)
	assert.Equal(t, "US", loadedItems[1].Country)
}

func TestCacheFile_Save_EmptySlice(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "empty_cache.data")
	cache := &cache.FileCache{FilePath: tempCacheFile}

	err := cache.Save([]*services.ProxyItemFull{})
	require.NoError(t, err)

	data, err := os.ReadFile(tempCacheFile)
	require.NoError(t, err)

	assert.Equal(t, "[]", string(data))
}

func TestCacheFile_Load_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(t *testing.T, path string)
		expectedSize   int
		expectedLength int
		expectError    bool
	}{
		{
			name: "File does not exist",
			setupFile: func(_ *testing.T, _ string) {
				// do nothing
			},
			expectedSize:   0,
			expectError:    false,
			expectedLength: 0,
		},
		{
			name: "File contains corrupted JSON",
			setupFile: func(t *testing.T, path string) {
				err := os.WriteFile(path, []byte("this is definitely not json {"), 0o600)
				require.NoError(t, err)
			},
			expectedSize:   0,
			expectError:    true,
			expectedLength: 0,
		},
		{
			name: "Cache TTL is expired",
			setupFile: func(t *testing.T, path string) {
				items := []*services.ProxyItemFull{
					{
						ProxyItem: services.ProxyItem{Host: "3.3.3.3", Port: "80"},
					},
				}
				data, _ := json.Marshal(items)
				require.NoError(t, os.WriteFile(path, data, 0o600))

				pastTime := time.Now().Add(-time.Hour * 2)
				require.NoError(t, os.Chtimes(path, pastTime, pastTime))
			},
			expectedSize:   0,
			expectError:    false,
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempCacheFile := filepath.Join(t.TempDir(), "edge_cache.data")
			cache := &cache.FileCache{FilePath: tempCacheFile}

			tt.setupFile(t, tempCacheFile)

			cfg := &config.Config{CacheTTL: 3600}

			loadedItems, err := cache.Load(cfg)

			if tt.expectError {
				require.Error(t, err, "Ожидается ошибка для кейса: "+tt.name)
			} else {
				require.NoError(t, err)
				assert.Len(t, loadedItems, tt.expectedLength, "Ожидается пустой результат для кейса: "+tt.name)
			}
		})
	}
}
