//go:build darwin

package sysproxy_test

import (
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/sysproxy"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
    _ = i18n.Init("en")
}

func TestNewSystemProxyManager_ReturnsValidInterface_darwin(t *testing.T) {
    manager := sysproxy.NewSystemProxyManager()
    require.NotNil(t, manager, "Factory always must return the interface")
}

func TestNewSystemProxyManager_PlatformConsistency_darwin(t *testing.T) {
    manager := sysproxy.NewSystemProxyManager()
    assert.IsNotType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use macOSProxyManager on darwin")
}

func TestParseDefaultRouteInterface_darwin(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "Valid route output",
            input:    "   route to: default\ndestination: default\n       mask: default\n    gateway: 192.168.1.1\n  interface: en0\n",
            expected: "en0",
        },
        {
            name:     "Missing interface line",
            input:    "   route to: default\ndestination: default\n",
            expected: "",
        },
        {
            name:     "Malformed interface line",
            input:    "interface:\n",
            expected: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := sysproxy.ParseDefaultRouteInterface(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestParseHardwarePortMapping(t *testing.T) {
    input := `Hardware Port: Wi-Fi
Device: en0
Ethernet Address: aa:bb:cc:dd:ee:ff

Hardware Port: Ethernet
Device: en1
Ethernet Address: 11:22:33:44:55:66
`

    result := sysproxy.ParseHardwarePortMapping(input)
    require.Len(t, result, 2, "Must parse exactly two interfaces")
    assert.Equal(t, "Wi-Fi", result["en0"], "Must map en0 to Wi-Fi")
    assert.Equal(t, "Ethernet", result["en1"], "Must map en1 to Ethernet")
}

func TestParseHardwarePortMapping_Empty(t *testing.T) {
    result := sysproxy.ParseHardwarePortMapping("")
    assert.Empty(t, result)
}

func TestParseNetworkSetupProxyOutput(t *testing.T) {
    tests := []struct {
        name          string
        input         string
        expectEnabled bool
        expectHost    string
        expectPort    string
    }{
        {
            name:          "Enabled proxy",
            input:         "Enabled: Yes\nServer: 192.168.1.100\nPort: 8080\nAuthenticated Proxy Enabled: 0",
            expectEnabled: true,
            expectHost:    "192.168.1.100",
            expectPort:    "8080",
        },
        {
            name:          "Disabled proxy",
            input:         "Enabled: No\nServer: 0.0.0.0\nPort: 0",
            expectEnabled: false,
            expectHost:    "0.0.0.0",
            expectPort:    "0",
        },
        {
            name:          "Missing fields defaults",
            input:         "Enabled: Yes",
            expectEnabled: true,
            expectHost:    "",
            expectPort:    "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            enabled, host, port, err := sysproxy.ParseNetworkSetupProxyOutput(tt.input)
            require.NoError(t, err)
            assert.Equal(t, tt.expectEnabled, enabled)
            assert.Equal(t, tt.expectHost, host)
            assert.Equal(t, tt.expectPort, port)
        })
    }
}
