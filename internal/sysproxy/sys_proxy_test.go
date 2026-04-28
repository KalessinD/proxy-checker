package sysproxy_test

import (
	"os"
	"proxy-checker/internal/common/i18n"
	"proxy-checker/internal/sysproxy"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

func TestNewSystemProxyManager_ReturnsValidInterface(t *testing.T) {
	manager := sysproxy.NewSystemProxyManager()
	require.NotNil(t, manager, "Factory always must return the interface|")
}

func TestNoopProxyManager_Behavior(t *testing.T) {
	manager := &sysproxy.NoOpProxyManager{}

	assert.False(t, manager.IsSupported(), "Noop manager doesn't not support proxies")

	mode, err := manager.GetMode()
	assert.NoError(t, err)
	assert.Empty(t, mode)

	assert.NoError(t, manager.SetMode("manual"))
	assert.NoError(t, manager.SetProxy("host", "port", "http"))

	hosts, err := manager.GetIgnoreHosts()
	assert.NoError(t, err)
	assert.Empty(t, hosts)

	assert.NoError(t, manager.SetIgnoreHosts("localhost"))
}

func TestNewSystemProxyManager_PlatformConsistency(t *testing.T) {
	manager := sysproxy.NewSystemProxyManager()

	if runtime.GOOS == "linux" {
		assert.IsNotType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use linuxProxyManager on Linux")
	} else {
		assert.IsType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use noopProxyManager on non-Linux")
	}
}

func TestParseGVariantStringArray(t *testing.T) {
	tests := []struct {
		name         string
		rawInput     string
		expectedList []string
	}{
		{name: "Empty array", rawInput: "[]", expectedList: nil},
		{name: "Single element", rawInput: "['localhost']", expectedList: []string{"localhost"}},
		{name: "Multiple elements sorted", rawInput: "['localhost', '127.0.0.0/8', '::1']", expectedList: []string{"127.0.0.0/8", "::1", "localhost"}},
		{name: "Elements without quotes", rawInput: "[localhost, 192.168.1.1]", expectedList: []string{"192.168.1.1", "localhost"}},
		{name: "Empty strings inside", rawInput: "['', 'test']", expectedList: []string{"test"}},
		{name: "Trailing comma", rawInput: "['localhost', ]", expectedList: []string{"localhost"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sysproxy.ParseGVariantStringArray(tt.rawInput)
			assert.Equal(t, tt.expectedList, result)
		})
	}
}

func TestFormatGVariantStringArray(t *testing.T) {
	tests := []struct {
		name           string
		inputHosts     []string
		expectedString string
	}{
		{name: "Nil slice", inputHosts: nil, expectedString: "[]"},
		{name: "Empty slice", inputHosts: []string{}, expectedString: "[]"},
		{name: "Single element", inputHosts: []string{"localhost"}, expectedString: "['localhost']"},
		{name: "Sorted and quoted", inputHosts: []string{"192.168.1.1", "localhost", "::1"}, expectedString: "['192.168.1.1', '::1', 'localhost']"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sysproxy.FormatGVariantStringArray(tt.inputHosts)
			assert.Equal(t, tt.expectedString, result)
		})
	}
}
