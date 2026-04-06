// nolint testpackage
package gui

import (
	"proxy-checker/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortProxyItems(tt.items, tt.sortCol, true)
			assert.Equal(t, tt.expectedFirst, tt.items[0].Host+tt.items[0].Port+string(tt.items[0].Type)+tt.items[0].Source)
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
