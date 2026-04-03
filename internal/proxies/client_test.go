package proxies_test

import (
	"proxy-checker/internal/proxies"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "HTTP mode", mode: "http"},
		{name: "HTTPS mode", mode: "https"},
		{name: "SOCKS5 mode", mode: "socks5"},
		{name: "SOCKS4 mode", mode: "socks4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := proxies.NewClient("127.0.0.1:8080", tt.mode, false)
			require.NoError(t, err)
			require.NotNil(t, client)
			require.NotNil(t, client.Transport)
		})
	}
}

func TestNewClient_HTTP2Forced(t *testing.T) {
	client, err := proxies.NewClient("127.0.0.1:8080", "http", true)
	require.NoError(t, err)
	require.NotNil(t, client)
}
