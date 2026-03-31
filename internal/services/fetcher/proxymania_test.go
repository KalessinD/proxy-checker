package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"proxy-checker/internal/common"
	"proxy-checker/internal/services/fetcher"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyManiaFetcher_Fetch_Success(t *testing.T) {
	// Фейковый HTML-ответ, имитирующий структуру таблицы ProxyMania
	htmlResponse := `
	<html><body>
	<table class="table_proxychecker">
		<tbody>
			<tr>
				<td class="proxy-cell">192.168.1.1:8080</td>
				<td class="country-cell">US</td>
				<td>SOCKS5</td>
				<td class="speed-fast">50 ms</td>
			</tr>
			<tr>
				<td class="proxy-cell">10.0.0.1:3128</td>
				<td class="country-cell">GB</td>
				<td>HTTP</td>
				<td class="speed-fast">N/A</td>
			</tr>
		</tbody>
	</table>
	</body></html>`

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Query().Get("type"), "SOCKS5")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlResponse))
	}))
	defer testServer.Close()

	fetcherInstance := &fetcher.ProxyManiaFetcher{
		BaseURL: testServer.URL,
	}

	settings := fetcher.Settings{
		Type:   common.ProxySOCKS5,
		MaxRTT: 150,
		Pages:  1,
	}

	items, err := fetcherInstance.Fetch(t.Context(), settings)
	require.NoError(t, err)

	require.Len(t, items, 2, "Должно быть распарсено 2 прокси")

	assert.Equal(t, "192.168.1.1", items[0].Host)
	assert.Equal(t, "8080", items[0].Port)
	assert.Equal(t, "US", items[0].Country)
	assert.Equal(t, common.ProxySOCKS5, items[0].Type)
	assert.Equal(t, 50, items[0].RTTms)

	assert.Equal(t, "10.0.0.1", items[1].Host)
	assert.Equal(t, "3128", items[1].Port)
	assert.Equal(t, common.ProxyHTTP, items[1].Type)
	assert.Equal(t, fetcher.DefaultUnknownRTT, items[1].RTTms, "N/A должно парситься в дефолтный RTT")
}
