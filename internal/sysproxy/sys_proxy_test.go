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

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		assert.IsNotType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use platform proxy manager on Linux/macOS")
	} else {
		assert.IsType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use noopProxyManager on other platforms")
	}
}
