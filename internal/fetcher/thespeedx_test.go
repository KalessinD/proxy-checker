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
	"go.uber.org/zap"
)

func TestTheSpeedXFetcher_Fetch_Success(t *testing.T) {
	err := i18n.Init("en")
	if err != nil {
		t.Fatal(err)
	}

	// Fake response
	socks5List := `1.1.1.1:1080
2.2.2.2:1080

# комментарий должен быть проигнорирован
invalid_line_without_port
3.3.3.3:1080`

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "socks5.txt") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(socks5List))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	logger := common.NewZapLogger(zap.NewNop().Sugar())
	fetcherInstance := fetcher.NewTextListFetcher(testServer.URL+"/", fetcher.NewTheSpeedXProvider(), logger)

	settings := fetcher.Settings{
		Type: common.ProxySOCKS5,
		Lang: "en",
	}

	items, err := fetcherInstance.Fetch(t.Context(), settings)
	require.NoError(t, err)

	require.Len(t, items, 3, "Must be parsed 3 valid proxies")

	assert.Equal(t, "1.1.1.1", items[0].Host)
	assert.Equal(t, "1080", items[0].Port)
	assert.Equal(t, common.ProxySOCKS5, items[0].Type)
	assert.Equal(t, "N/A", items[0].Country, "Country is always N/A for TheSpeedX")
	assert.Equal(t, "3.3.3.3", items[2].Host)
	assert.Equal(t, "1080", items[2].Port)
}

func TestTheSpeedXFetcher_Fetch_HttpError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer testServer.Close()

	logger := common.NewZapLogger(zap.NewNop().Sugar())
	fetcherInstance := fetcher.NewTextListFetcher(testServer.URL+"/", fetcher.NewTheSpeedXProvider(), logger)

	settings := fetcher.Settings{
		Type: common.ProxySOCKS5,
		Lang: "en",
	}

	items, err := fetcherInstance.Fetch(t.Context(), settings)

	require.Error(t, err)
	assert.Nil(t, items)
	assert.Contains(t, err.Error(), "failed to fetch all sources")
	assert.Contains(t, err.Error(), "status 403")
}

func TestTheSpeedXProvider_GetFilesByType(t *testing.T) {
	provider := fetcher.NewTheSpeedXProvider()

	tests := []struct {
		name      string
		proxyType common.ProxyType
		expected  []string
	}{
		{name: "SOCKS5", proxyType: common.ProxySOCKS5, expected: []string{fetcher.ThespeedxSocks5FileName}},
		{name: "HTTP", proxyType: common.ProxyHTTP, expected: []string{fetcher.ThespeedxHTTPFileName}},
		{name: "All types", proxyType: common.ProxyAll, expected: []string{
			fetcher.ThespeedxHTTPFileName,
			fetcher.ThespeedxHTTPSFileName,
			fetcher.ThespeedxSocks4FileName,
			fetcher.ThespeedxSocks5FileName,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, provider.GetFilesByType(tt.proxyType))
		})
	}
}

func TestTheSpeedXProvider_GetTypeFromFilename(t *testing.T) {
	provider := fetcher.NewTheSpeedXProvider()

	assert.Equal(t, common.ProxySOCKS5, provider.GetTypeFromFilename("socks5.txt"))
	assert.Equal(t, common.ProxyHTTP, provider.GetTypeFromFilename("http.txt"))
	assert.Equal(t, common.ProxyType("unknown"), provider.GetTypeFromFilename("unknown.txt"))
}
