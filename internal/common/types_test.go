package common_test

import (
	"proxy-checker/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsKnownSource(t *testing.T) {
	tests := []struct {
		name     string
		source   common.Source
		expected bool
	}{
		{name: "Known source ProxyMania", source: common.SourceProxyMania, expected: true},
		{name: "Known source TheSpeedX", source: common.SourceTheSpeedX, expected: true},
		{name: "Known source Proxifly", source: common.SourceProxifly, expected: true},
		{name: "Unknown source", source: common.Source("unknown"), expected: false},
		{name: "Empty source", source: common.Source(""), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, common.IsKnownProxySource(tt.source))
		})
	}
}

func TestAllowedProxyTypesStrings(t *testing.T) {
	result := common.AllowedProxyTypesStrings()
	require.Len(t, result, 4, "Must return exactly 4 proxy types")
	assert.Equal(t, "http", result[0])
	assert.Equal(t, "https", result[1])
	assert.Equal(t, "socks4", result[2])
	assert.Equal(t, "socks5", result[3])
}

func TestIsKnownProxyType(t *testing.T) {
	tests := []struct {
		name      string
		proxyType common.ProxyType
		expected  bool
	}{
		{name: "HTTP type", proxyType: common.ProxyHTTP, expected: true},
		{name: "HTTPS type", proxyType: common.ProxyHTTPS, expected: true},
		{name: "SOCKS4 type", proxyType: common.ProxySOCKS4, expected: true},
		{name: "SOCKS5 type", proxyType: common.ProxySOCKS5, expected: true},
		{name: "ALL type", proxyType: common.ProxyAll, expected: true},
		{name: "Unknown type", proxyType: common.ProxyType("ftp"), expected: false},
		{name: "Empty type", proxyType: common.ProxyType(""), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, common.IsKnownProxyType(tt.proxyType))
		})
	}
}

func TestSourcesToStrings(t *testing.T) {
	sources := []common.Source{common.SourceProxyMania, common.SourceTheSpeedX}
	result := common.SourcesToStrings(sources)

	assert.Equal(t, []string{"proxymania", "thespeedx"}, result)
}

func TestJoinSources(t *testing.T) {
	sources := []common.Source{common.SourceProxyMania, common.SourceProxifly}

	result := common.JoinSources(sources, ", ")

	assert.Equal(t, "proxymania, proxifly", result)
}
