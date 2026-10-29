//go:build darwin

package sysproxy_test

import (
	"proxy-checker/internal/sysproxy"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemProxyManager_ReturnsValidInterface_darwin(t *testing.T) {
	manager := sysproxy.NewSystemProxyManager()
	require.NotNil(t, manager, "Factory always must return the interface")
}

func TestNewSystemProxyManager_PlatformConsistency_darwin(t *testing.T) {
	manager := sysproxy.NewSystemProxyManager()
	assert.IsNotType(t, &sysproxy.NoOpProxyManager{}, manager, "Must use networkSetupProxyManager on darwin")
}
