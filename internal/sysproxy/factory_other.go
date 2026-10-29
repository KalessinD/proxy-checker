//go:build !linux && !darwin

package sysproxy

// NewSystemProxyManager returns the no-op implementation on unsupported platforms.
func NewSystemProxyManager() SystemProxyManager {
	return &NoOpProxyManager{}
}
