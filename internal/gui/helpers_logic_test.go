// nolint testpackage
package gui

import (
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/config"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSortProxyItems_AscendingOrder(t *testing.T) {
	items := []*ProxyItemWrapper{
		{Host: "2.2.2.2", Port: "8080"},
		{Host: "1.1.1.1", Port: "3128"},
		{Host: "3.3.3.3", Port: "1080"},
	}

	sortProxyItems(items, 1, true)

	assert.Equal(t, "1.1.1.1", items[0].Host)
	assert.Equal(t, "2.2.2.2", items[1].Host)
	assert.Equal(t, "3.3.3.3", items[2].Host)
}

func TestSortProxyItems_DescendingOrder(t *testing.T) {
	items := []*ProxyItemWrapper{
		{Host: "1.1.1.1", Port: "3128"},
		{Host: "3.3.3.3", Port: "1080"},
	}

	sortProxyItems(items, 1, false)

	assert.Equal(t, "3.3.3.3", items[0].Host)
	assert.Equal(t, "1.1.1.1", items[1].Host)
}

func TestSortProxyItems_ByDifferentColumns(t *testing.T) {
	tests := []struct {
		name          string
		items         []*ProxyItemWrapper
		sortCol       int
		expectedFirst string
	}{
		{
			name:          "Sort by Source",
			items:         []*ProxyItemWrapper{{Source: "thespeedx"}, {Source: "proxymania"}},
			sortCol:       0,
			expectedFirst: "proxymania",
		},
		{
			name:          "Sort by Port as string",
			items:         []*ProxyItemWrapper{{Port: "8080"}, {Port: "3128"}},
			sortCol:       2,
			expectedFirst: "3128",
		},
		{
			name:          "Sort by Type",
			items:         []*ProxyItemWrapper{{Type: common.ProxySOCKS5}, {Type: common.ProxyHTTP}},
			sortCol:       3,
			expectedFirst: "http",
		},
		{
			name:          "Sort by TCP latency numerically",
			items:         []*ProxyItemWrapper{{TCP: "1000ms", TCPMs: 1000}, {TCP: "18ms", TCPMs: 18}},
			sortCol:       5,
			expectedFirst: "18ms",
		},
		{
			name:          "Sort by HTTP latency numerically",
			items:         []*ProxyItemWrapper{{HTTP: "2000ms", HTTPMs: 2000}, {HTTP: "150ms", HTTPMs: 150}},
			sortCol:       6,
			expectedFirst: "150ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortProxyItems(tt.items, tt.sortCol, true)

			combined := tt.items[0].Host + tt.items[0].Port + string(tt.items[0].Type) + tt.items[0].Source + tt.items[0].TCP + tt.items[0].HTTP
			assert.Contains(t, combined, tt.expectedFirst)
		})
	}
}

func TestSortProxyItems_EmptySlice(t *testing.T) {
	items := []*ProxyItemWrapper{}
	assert.NotPanics(t, func() {
		sortProxyItems(items, 0, true)
	})
	assert.Empty(t, items)
}

func TestSourcesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []common.Source
		b        []common.Source
		expected bool
	}{
		{name: "Exactly equal", a: []common.Source{common.SourceProxyMania, common.SourceTheSpeedX}, b: []common.Source{common.SourceProxyMania, common.SourceTheSpeedX}, expected: true},
		{name: "Different length", a: []common.Source{common.SourceProxyMania}, b: []common.Source{common.SourceProxyMania, common.SourceTheSpeedX}, expected: false},
		{name: "Different order", a: []common.Source{common.SourceProxyMania, common.SourceTheSpeedX}, b: []common.Source{common.SourceTheSpeedX, common.SourceProxyMania}, expected: false},
		{name: "Both empty", a: []common.Source{}, b: []common.Source{}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sourcesEqual(tt.a, tt.b))
		})
	}
}

func TestFilterBySource_MatchesAndExcludes(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	g.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Source: "proxymania"},
		{Host: "2.2.2.2", Source: "thespeedx"},
		{Host: "3.3.3.3", Source: "proxymania"},
	}

	// Filter by specific source
	filtered := g.filterBySource("proxymania")
	require.Len(t, filtered, 2, "Must return only items matching the source")
	assert.Equal(t, "1.1.1.1", filtered[0].Host)
	assert.Equal(t, "3.3.3.3", filtered[1].Host)

	// Filter by "All"
	filtered = g.filterBySource(i18n.T("gui.filter_all"))
	assert.Len(t, filtered, 3, "Filter All must return all items")

	// Filter by non-existent source
	filtered = g.filterBySource("unknown_source")
	assert.Empty(t, filtered, "Unknown source must return empty slice")
}

func TestFilterByCountry_MatchesAndExcludes(t *testing.T) {
	testApp := test.NewTempApp(t)
	defer testApp.Quit()

	cfg := config.DefaultConfig()
	logger := common.NewZapLogger(zap.NewNop().Sugar())
	g := NewAppGUI(testApp, cfg, logger, "dev")

	g.proxyItems = []*ProxyItemWrapper{
		{Host: "1.1.1.1", Country: "US"},
		{Host: "2.2.2.2", Country: "GB"},
		{Host: "3.3.3.3", Country: "US"},
	}

	// Filter by specific country
	filtered := g.filterByCountry("US")
	require.Len(t, filtered, 2, "Must return only items matching the country")
	assert.Equal(t, "1.1.1.1", filtered[0].Host)
	assert.Equal(t, "3.3.3.3", filtered[1].Host)

	// Filter by "All"
	filtered = g.filterByCountry(i18n.T("gui.filter_all"))
	assert.Len(t, filtered, 3, "Filter All must return all items")

	// Filter by non-existent country
	filtered = g.filterByCountry("Unknown")
	assert.Empty(t, filtered, "Unknown country must return empty slice")
}
