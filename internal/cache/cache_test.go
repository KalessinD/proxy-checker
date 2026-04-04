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
	assert.Equal(t, customPath, actualPath, "GetFilePath must return the set path")
}

func TestCacheFile_NewFileCache(t *testing.T) {
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	c := cache.NewFileStorage(logger)
	require.NotNil(t, c)

	assert.Contains(t, c.GetFilePath(), common.AppName+"-cache.data")
	assert.Contains(t, c.GetFilePath(), os.TempDir())
}

func TestCacheFile_SaveAndLoad_ValidData_SpecificType(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "valid_cache.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	inputItems := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5}},
	}

	err := c.Save(common.SourceProxyMania, common.ProxySOCKS5, inputItems, 3600)
	require.NoError(t, err)
	assert.FileExists(t, tempCacheFile)

	loadedItems, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
	require.NoError(t, err)

	require.Len(t, loadedItems, 1, "Must be loadead exactly 1 element")
	assert.Equal(t, "1.1.1.1", loadedItems[0].Host)
	assert.Equal(t, common.ProxySOCKS5, loadedItems[0].Type)
}

func TestCacheFile_Save_MultipleTypesForSameSource(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "multi_type.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	itemsSOCKS5 := []*services.ProxyItemFull{{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Type: common.ProxySOCKS5}}}
	itemsHTTP := []*services.ProxyItemFull{{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Type: common.ProxyHTTP}}}

	require.NoError(t, c.Save(common.SourceProxyMania, common.ProxySOCKS5, itemsSOCKS5, 3600))
	require.NoError(t, c.Save(common.SourceProxyMania, common.ProxyHTTP, itemsHTTP, 3600))

	loadedSOCKS5, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
	require.NoError(t, err)
	assert.Equal(t, "1.1.1.1", loadedSOCKS5[0].Host)

	loadedHTTP, err := c.Load(common.SourceProxyMania, common.ProxyHTTP)
	require.NoError(t, err)
	assert.Equal(t, "2.2.2.2", loadedHTTP[0].Host)

	var cf cache.Data
	data, err := os.ReadFile(tempCacheFile)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &cf))
	assert.Len(t, cf.Sources, 2, "Must be two separate keys in the file")
}

func TestCacheFile_Save_AllType_SplitsData(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "split_all.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	itemsAll := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Type: common.ProxySOCKS5}},
		{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Type: common.ProxyHTTP}},
	}

	err := c.Save(common.SourceProxyMania, common.ProxyAll, itemsAll, 3600)
	require.NoError(t, err)

	loadedSOCKS5, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
	require.NoError(t, err)
	assert.Len(t, loadedSOCKS5, 1, "ALL type: SOCKS5 must be saved separately")
	assert.Equal(t, "1.1.1.1", loadedSOCKS5[0].Host)

	loadedHTTP, err := c.Load(common.SourceProxyMania, common.ProxyHTTP)
	require.NoError(t, err)
	assert.Len(t, loadedHTTP, 1, "ALL type: HTTP must be saved separately")
	assert.Equal(t, "2.2.2.2", loadedHTTP[0].Host)
}

func TestCacheFile_Load_AllType_MergesData(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "merge_all.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	require.NoError(t, c.Save(common.SourceProxyMania, common.ProxySOCKS5, []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Type: common.ProxySOCKS5}},
	}, 3600))
	require.NoError(t, c.Save(common.SourceProxyMania, common.ProxyHTTP, []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "2.2.2.2", Type: common.ProxyHTTP}},
	}, 3600))

	loadedAll, err := c.Load(common.SourceProxyMania, common.ProxyAll)
	require.NoError(t, err)
	assert.Len(t, loadedAll, 2, "ALL type: all types must be merged to be used in source")
}

func TestCacheFile_Load_AllType_DeduplicatesData(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "dedup_all.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	require.NoError(t, c.Save(common.SourceProxyMania, common.ProxySOCKS5, []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "8080", Type: common.ProxySOCKS5}},
	}, 3600))
	require.NoError(t, c.Save(common.SourceProxyMania, common.ProxyHTTP, []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Port: "8080", Type: common.ProxyHTTP}},
	}, 3600))

	loadedAll, err := c.Load(common.SourceProxyMania, common.ProxyAll)
	require.NoError(t, err)
	assert.Len(t, loadedAll, 1, "ALL type: deduplication by Host:Port")
}

func TestCacheFile_Save_EmptySlice(t *testing.T) {
	tempCacheFile := filepath.Join(t.TempDir(), "empty_cache.data")
	c := &cache.FileStorage{
		FilePath: tempCacheFile,
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	err := c.Save(common.SourceProxyMania, common.ProxySOCKS5, []*services.ProxyItemFull{}, 3600)
	require.NoError(t, err)

	loaded, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestCacheFile_Load_EdgeCases(t *testing.T) {
	t.Run("Source and type not found", func(t *testing.T) {
		tempCacheFile := filepath.Join(t.TempDir(), "not_found.data")
		c := &cache.FileStorage{
			FilePath: tempCacheFile,
			Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
		}
		initialData := cache.Data{Sources: make(map[string]*cache.Record)}
		data, err := json.Marshal(initialData)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tempCacheFile, data, 0o600))

		items, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("Cache expired", func(t *testing.T) {
		tempCacheFile := filepath.Join(t.TempDir(), "expired.data")
		c := &cache.FileStorage{
			FilePath: tempCacheFile,
			Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
		}

		err := c.Save(common.SourceProxyMania, common.ProxySOCKS5, []*services.ProxyItemFull{{
			ProxyItem: services.ProxyItem{Host: "1.1.1.1"},
		}}, -1)
		require.NoError(t, err)

		items, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
		require.NoError(t, err)
		assert.Empty(t, items, "Expired cache must return an empty list")
	})

	t.Run("File does not exist", func(t *testing.T) {
		c := &cache.FileStorage{FilePath: "/nonexistent/path/cache.data"}
		items, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
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

		items, err := c.Load(common.SourceProxyMania, common.ProxySOCKS5)
		require.NoError(t, err, "Unexpected error")
		assert.Empty(t, items, "Corrupted JSON must return empty list")
	})
}

func TestCacheFile_Save_AllType_SplitWriteError(t *testing.T) {
	c := &cache.FileStorage{
		FilePath: "/nonexistent/deep/dir/cache.data",
		Logger:   common.NewZapLogger(zap.NewNop().Sugar()),
	}

	itemsAll := []*services.ProxyItemFull{
		{ProxyItem: services.ProxyItem{Host: "1.1.1.1", Type: common.ProxySOCKS5}},
	}

	err := c.Save(common.SourceProxyMania, common.ProxyAll, itemsAll, 3600)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write cache")
}
