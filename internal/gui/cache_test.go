package gui_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"proxy-checker/internal/common"
	"proxy-checker/internal/config"
	"proxy-checker/internal/gui"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Компиляторная проверка того, что структура реализует интерфейс
var _ gui.CacheInterface = (*gui.CacheFile)(nil)

func TestCacheFile_GetFilePath(t *testing.T) {
	customPath := "/tmp/my_custom_cache.data"
	cache := &gui.CacheFile{FilePath: customPath}

	actualPath := cache.GetFilePath()
	assert.Equal(t, customPath, actualPath, "GetFilePath должен возвращать установленный путь")
}

func TestCacheFile_NewCacheFile(t *testing.T) {
	cache := gui.NewCacheFile()
	require.NotNil(t, cache)

	assert.Contains(t, cache.GetFilePath(), common.AppName+"-cache.data")
	assert.Contains(t, cache.GetFilePath(), os.TempDir())
}

func TestCacheFile_SaveAndLoad_ValidData(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "valid_cache.data")
	cache := &gui.CacheFile{FilePath: tempCacheFile}

	inputItems := []*gui.ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "8080", Type: common.ProxyHTTP},
		{Host: "2.2.2.2", Port: "3128", Type: common.ProxySOCKS5, Country: "US"},
	}

	err := cache.Save(inputItems)
	require.NoError(t, err)
	assert.FileExists(t, tempCacheFile)

	cfg := &config.Config{CacheTTL: 3600}
	loadedItems := cache.Load(cfg)

	require.Len(t, loadedItems, 2, "Должно быть загружено 2 элемента")
	assert.Equal(t, "1.1.1.1", loadedItems[0].Host)
	assert.Equal(t, common.ProxySOCKS5, loadedItems[1].Type)
	assert.Equal(t, "US", loadedItems[1].Country)
}

func TestCacheFile_Save_EmptySlice(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "empty_cache.data")
	cache := &gui.CacheFile{FilePath: tempCacheFile}

	err := cache.Save([]*gui.ProxyItemWrapper{})
	require.NoError(t, err)

	data, err := os.ReadFile(tempCacheFile)
	require.NoError(t, err)

	assert.Equal(t, "[]", string(data))
}

func TestCacheFile_Load_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		setupFile    func(t *testing.T, path string)
		expectedSize int
	}{
		{
			name: "File does not exist",
			setupFile: func(_ *testing.T, _ string) {
				// do nothing
			},
			expectedSize: 0,
		},
		{
			name: "File contains corrupted JSON",
			setupFile: func(t *testing.T, path string) {
				err := os.WriteFile(path, []byte("this is definitely not json {"), 0o600)
				require.NoError(t, err)
			},
			expectedSize: 0,
		},
		{
			name: "Cache TTL is expired",
			setupFile: func(t *testing.T, path string) {
				items := []*gui.ProxyItemWrapper{{Host: "3.3.3.3", Port: "80"}}
				data, _ := json.Marshal(items)
				require.NoError(t, os.WriteFile(path, data, 0o600))

				pastTime := time.Now().Add(-time.Hour * 2)
				require.NoError(t, os.Chtimes(path, pastTime, pastTime))
			},
			expectedSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempCacheFile := filepath.Join(t.TempDir(), "edge_cache.data")
			cache := &gui.CacheFile{FilePath: tempCacheFile}

			tt.setupFile(t, tempCacheFile)

			cfg := &config.Config{CacheTTL: 3600}

			loadedItems := cache.Load(cfg)

			assert.Len(t, loadedItems, tt.expectedSize, "Ожидается пустой результат для кейса: "+tt.name)
		})
	}
}
