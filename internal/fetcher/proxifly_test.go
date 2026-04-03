package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/fetcher"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxiflyFetcher_Fetch_Success(t *testing.T) {
	err := i18n.Init("en")
	if err != nil {
		t.Fatal(err)
	}

	// Fake response
	socks5List := `socks5://1.1.1.1:1080
socks5://2.2.2.2:1080
2.2.2.4:1080

# комментарий должен быть проигнорирован
invalid_line_without_port
socks5://3.3.3.3:1080`

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "socks5/data.txt") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(socks5List))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	fetcherInstance := fetcher.NewTextListFetcher(testServer.URL+"/", fetcher.NewProxiflyProvider())

	settings := fetcher.Settings{
		Type: common.ProxySOCKS5,
		Lang: "en",
	}

	items, err := fetcherInstance.Fetch(t.Context(), settings)
	require.NoError(t, err)

	require.Len(t, items, 3, "Должно быть распарсено 3 валидных прокси")

	assert.Equal(t, "1.1.1.1", items[0].Host)
	assert.Equal(t, "1080", items[0].Port)
	assert.Equal(t, common.ProxySOCKS5, items[0].Type)
	assert.Equal(t, "N/A", items[0].Country, "Для Proxifly страна всегда N/A")

	assert.Equal(t, "3.3.3.3", items[2].Host)
	assert.Equal(t, "1080", items[2].Port)
}

func TestProxiflyFetcher_Fetch_HttpError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer testServer.Close()

	fetcherInstance := fetcher.NewTextListFetcher(testServer.URL+"/", fetcher.NewProxiflyProvider())

	settings := fetcher.Settings{
		Type: common.ProxySOCKS5,
		Lang: "en",
	}

	items, err := fetcherInstance.Fetch(t.Context(), settings)

	require.Error(t, err)
	assert.Nil(t, items)
	assert.Contains(t, err.Error(), i18n.T("fetcher.err_fetch_list_status"))
	assert.Contains(t, err.Error(), "403")
}

func TestProxiflyProvider_GetFilesByType(t *testing.T) {
	provider := fetcher.NewProxiflyProvider()

	tests := []struct {
		name      string
		proxyType common.ProxyType
		expected  []string
	}{
		{name: "SOCKS5", proxyType: common.ProxySOCKS5, expected: []string{fetcher.ProxiflyHSocks5FileName}},
		{name: "HTTP", proxyType: common.ProxyHTTP, expected: []string{fetcher.ProxiflyHTTPFileName}},
		{name: "All types", proxyType: common.ProxyAll, expected: []string{
			fetcher.ProxiflyHTTPFileName,
			fetcher.ProxiflyHTTPSFileName,
			fetcher.ProxiflySsocks4FileName,
			fetcher.ProxiflyHSocks5FileName,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, provider.GetFilesByType(tt.proxyType))
		})
	}
}

func TestProxiflyProvider_GetTypeFromFilename(t *testing.T) {
	provider := fetcher.NewProxiflyProvider()

	assert.Equal(t, common.ProxySOCKS5, provider.GetTypeFromFilename("socks5/data.txt"))
	assert.Equal(t, common.ProxyHTTP, provider.GetTypeFromFilename("http/data.txt"))
	assert.Equal(t, common.ProxySOCKS4, provider.GetTypeFromFilename("socks4/data.txt"))
	assert.Equal(t, common.ProxyType("invalid"), provider.GetTypeFromFilename("invalid/data.txt"))
}
