package cache_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/services"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var _ cache.StorageInterface = (*cache.FileStorage)(nil)

func TestCacheFile_GetFilePath(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "my_custom_cache.data")
	c := &cache.FileStorage{FilePath: customPath}

	actualPath := c.GetFilePath()
	assert.Equal(t, customPath, actualPath, "GetFilePath должен возвращать установленный путь")
}

func TestCacheFile_NewFileCache(t *testing.T) {
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	c := cache.NewFileStorage(logger)
	require.NotNil(t, c)

	assert.Contains(t, c.GetFilePath(), common.AppName+"-cache.data")
	assert.Contains(t, c.GetFilePath(), os.TempDir())
}

func TestCacheFile_SaveAndLoad_ValidData(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "valid_cache.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	sourceName := "test_source"
	inputItems := []*services.ProxyItemFull{
		{
			ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "8080"},
		},
	}

	ttl := 3600
	err := c.Save(sourceName, inputItems, ttl)
	require.NoError(t, err)
	assert.FileExists(t, tempCacheFile)

	loadedItems, err := c.Load(sourceName)
	require.NoError(t, err)

	require.Len(t, loadedItems, 1, "Должно быть загружено 1 элемент")
	assert.Equal(t, "1.1.1.1", loadedItems[0].Host)
}

func TestCacheFile_Save_MultipleSources(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "multi_source.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	items1 := []*services.ProxyItemFull{{ProxyItem: services.ProxyItem{Host: "1.1.1.1"}}}
	items2 := []*services.ProxyItemFull{{ProxyItem: services.ProxyItem{Host: "2.2.2.2"}}}

	err := c.Save("source1", items1, 3600)
	require.NoError(t, err)

	err = c.Save("source2", items2, 3600)
	require.NoError(t, err)

	loaded1, err := c.Load("source1")
	require.NoError(t, err)
	assert.Equal(t, "1.1.1.1", loaded1[0].Host)

	loaded2, err := c.Load("source2")
	require.NoError(t, err)
	assert.Equal(t, "2.2.2.2", loaded2[0].Host)

	var cf cache.Data
	data, _ := os.ReadFile(tempCacheFile)
	err = json.Unmarshal(data, &cf)
	require.NoError(t, err)
	assert.Len(t, cf.Sources, 2)
}

func TestCacheFile_Save_EmptySlice(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "empty_cache.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	err := c.Save("empty_source", []*services.ProxyItemFull{}, 3600)
	require.NoError(t, err)

	loaded, err := c.Load("empty_source")
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestCacheFile_Load_EdgeCases(t *testing.T) {
	t.Run("Source not found", func(t *testing.T) {
		tempCacheFile := filepath.Join(t.TempDir(), "not_found.data")
		c := &cache.FileStorage{
			FilePath: tempCacheFile,
			Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
		}
		initialData := cache.Data{Sources: make(map[string]*cache.Record)}
		data, err := json.Marshal(initialData)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tempCacheFile, data, 0o600))

		items, err := c.Load("non_existent_source")
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("Cache expired", func(t *testing.T) {
		tempCacheFile := filepath.Join(t.TempDir(), "expired.data")
		c := &cache.FileStorage{
			FilePath: tempCacheFile,
			Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
		}

		err := c.Save("expired_source", []*services.ProxyItemFull{{ProxyItem: services.ProxyItem{Host: "1.1.1.1"}}}, -1)
		require.NoError(t, err)

		items, err := c.Load("expired_source")
		require.NoError(t, err)
		assert.Empty(t, items, "Просроченный кэш должен возвращать пустой список")
	})

	t.Run("File does not exist", func(t *testing.T) {
		c := &cache.FileStorage{FilePath: "/nonexistent/path/cache.data"}
		items, err := c.Load("any_source")
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("Corrupted JSON", func(t *testing.T) {
		tempCacheFile := filepath.Join(t.TempDir(), "corrupted.data")
		require.NoError(t, os.WriteFile(tempCacheFile, []byte("{bad json"), 0o600))
		c := &cache.FileStorage{
			FilePath: tempCacheFile,
			Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
		}

		items, err := c.Load("any_source")
		require.NoError(t, err, "При ошибке парсинга должны вернуть пустой список без ошибки")
		assert.Empty(t, items)
	})
}
