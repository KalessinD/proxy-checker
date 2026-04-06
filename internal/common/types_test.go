package common_test

import (
	"proxy-checker/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
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
