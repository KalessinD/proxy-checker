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

	fetcherInstance := &fetcher.TheSpeedXFetcher{
		BaseURL: testServer.URL + "/",
	}

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
	assert.Equal(t, "N/A", items[0].Country, "Для TheSpeedX страна всегда N/A")

	assert.Equal(t, "3.3.3.3", items[2].Host)
	assert.Equal(t, "1080", items[2].Port)
}

func TestTheSpeedXFetcher_Fetch_HttpError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer testServer.Close()

	fetcherInstance := &fetcher.TheSpeedXFetcher{
		BaseURL: testServer.URL + "/",
	}

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
