//go:build darwin

package sysproxy

// NewSystemProxyManager returns the macOS implementation on darwin.
func NewSystemProxyManager() SystemProxyManager {
	return newMacOSProxyManager()
}
