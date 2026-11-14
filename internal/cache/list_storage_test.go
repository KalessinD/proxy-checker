package cache_test

import (
	"os"
	"path/filepath"
	"proxy-checker/internal/cache"
	"proxy-checker/internal/common"
	"proxy-checker/internal/fetcher"
	"proxy-checker/internal/services"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newListStorage creates a ListStorage rooted in an isolated temporary directory.
func newListStorage(t *testing.T) *cache.ListStorage {
	t.Helper()

	storage := cache.NewListStorage(common.NewZapLogger(zap.NewNop().Sugar()))
	storage.DirPath = filepath.Join(t.TempDir(), "lists")
	return storage
}

func sampleListItem(host string, proxyType common.ProxyType) *fetcher.ProxyItem {
	return &fetcher.ProxyItem{
		Host:    host,
		Port:    "8080",
		Type:    proxyType,
		Country: "US",
	}
}

func TestListStorage_SaveLoadRoundTrip(t *testing.T) {
	storage := newListStorage(t)
	source := common.SourceProxyMania

	savedErr := storage.Save(source, common.ProxySOCKS5, []*fetcher.ProxyItem{
		sampleListItem("1.2.3.4", common.ProxySOCKS5),
		sampleListItem("5.6.7.8", common.ProxySOCKS5),
	})
	require.NoError(t, savedErr)

	loaded, loadErr := storage.Load(source, common.ProxySOCKS5)
	require.NoError(t, loadErr)
	require.Len(t, loaded, 2)
	assert.Equal(t, "1.2.3.4", loaded[0].Host)
	assert.Equal(t, "8080", loaded[0].Port)
	assert.Equal(t, common.ProxySOCKS5, loaded[0].Type)
	assert.Equal(t, "US", loaded[0].Country)
}

func TestListStorage_LoadMissing(t *testing.T) {
	storage := newListStorage(t)

	loaded, loadErr := storage.Load(common.SourceProxyMania, common.ProxyHTTP)
	require.ErrorIs(t, loadErr, services.ErrListCacheMiss)
	assert.Nil(t, loaded)
}

func TestListStorage_LoadExpired(t *testing.T) {
	storage := newListStorage(t)
	source := common.SourceProxyMania

	savedErr := storage.Save(source, common.ProxyHTTP, []*fetcher.ProxyItem{
		sampleListItem("1.2.3.4", common.ProxyHTTP),
	})
	require.NoError(t, savedErr)

	// Backdate the file to just past the TTL.
	path := filepath.Join(storage.DirPath, string(source)+"-"+string(common.ProxyHTTP)+".txt")
	stale := time.Now().Add(-cache.ListCacheTTL - time.Minute)
	require.NoError(t, os.Chtimes(path, stale, stale))

	loaded, loadErr := storage.Load(source, common.ProxyHTTP)
	require.ErrorIs(t, loadErr, services.ErrListCacheMiss)
	assert.Nil(t, loaded)
}

func TestListStorage_SaveAllSplitsByType(t *testing.T) {
	storage := newListStorage(t)
	source := common.SourceTheSpeedX

	items := []*fetcher.ProxyItem{
		sampleListItem("1.2.3.4", common.ProxyHTTP),
		sampleListItem("5.6.7.8", common.ProxySOCKS5),
		sampleListItem("9.9.9.9", common.ProxySOCKS5),
	}

	savedErr := storage.Save(source, common.ProxyAll, items)
	require.NoError(t, savedErr)

	// Every known type gets a file, even when empty, so an "all" load stays valid.
	for _, proxyType := range []common.ProxyType{common.ProxyHTTP, common.ProxyHTTPS, common.ProxySOCKS4, common.ProxySOCKS5} {
		loaded, loadErr := storage.Load(source, proxyType)
		require.NoError(t, loadErr, proxyType)
		switch proxyType {
		case common.ProxyHTTP:
			require.Len(t, loaded, 1)
			assert.Equal(t, "1.2.3.4", loaded[0].Host)
		case common.ProxySOCKS5:
			require.Len(t, loaded, 2)
		case common.ProxyHTTPS, common.ProxySOCKS4, common.ProxyAll:
			assert.Empty(t, loaded, proxyType)
		}
	}

	allLoaded, allErr := storage.Load(source, common.ProxyAll)
	require.NoError(t, allErr)
	require.Len(t, allLoaded, 3)
}

func TestListStorage_LoadAllWithMissingType(t *testing.T) {
	storage := newListStorage(t)
	source := common.SourceProxifly

	// Only one type file written; the "all" load must treat the batch as invalid.
	savedErr := storage.Save(source, common.ProxySOCKS4, []*fetcher.ProxyItem{
		sampleListItem("1.2.3.4", common.ProxySOCKS4),
	})
	require.NoError(t, savedErr)

	loaded, loadErr := storage.Load(source, common.ProxyAll)
	require.ErrorIs(t, loadErr, services.ErrListCacheMiss)
	assert.Nil(t, loaded)
}

func TestListStorage_ParseSkipsMalformedLines(t *testing.T) {
	storage := newListStorage(t)
	source := common.SourceProxyMania

	dirPath := storage.DirPath
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	path := filepath.Join(dirPath, string(source)+"-"+string(common.ProxyHTTP)+".txt")
	content := "1.2.3.4\t8080\thttp\tUS\n" +
		"broken-line-without-tabs\n" +
		"5.6.7.8\t3128\tnotatype\tUS\n" +
		"6.6.6.6\t9999\thttp\t\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	loaded, loadErr := storage.Load(source, common.ProxyHTTP)
	require.NoError(t, loadErr)
	require.Len(t, loaded, 2)
	assert.Equal(t, "1.2.3.4", loaded[0].Host)
	assert.Equal(t, "6.6.6.6", loaded[1].Host)
	assert.Empty(t, loaded[1].Country)
}
