package services_test

import (
	"proxy-checker/internal/services"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveSchema_DefaultBehavior(t *testing.T) {
	tests := []struct {
		name           string
		proxyMode      string
		forceHTTP2     bool
		expectedSchema string
	}{
		{
			name:           "HTTP mode without force",
			proxyMode:      "http",
			forceHTTP2:     false,
			expectedSchema: "http://",
		},
		{
			name:           "HTTPS mode without force",
			proxyMode:      "https",
			forceHTTP2:     false,
			expectedSchema: "https://",
		},
		{
			name:           "SOCKS4 mode without force",
			proxyMode:      "socks4",
			forceHTTP2:     false,
			expectedSchema: "http://",
		},
		{
			name:           "SOCKS5 mode without force",
			proxyMode:      "socks5",
			forceHTTP2:     false,
			expectedSchema: "https://",
		},
		{
			name:           "Unknown mode fallback without force",
			proxyMode:      "unknown",
			forceHTTP2:     false,
			expectedSchema: "http://",
		},
		{
			name:           "HTTP mode WITH force HTTP2",
			proxyMode:      "http",
			forceHTTP2:     true,
			expectedSchema: "https://",
		},
		{
			name:           "SOCKS4 mode WITH force HTTP2",
			proxyMode:      "socks4",
			forceHTTP2:     true,
			expectedSchema: "https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSchema := services.ResolveSchema(tt.proxyMode, tt.forceHTTP2)
			assert.Equal(t, tt.expectedSchema, actualSchema)
		})
	}
}
