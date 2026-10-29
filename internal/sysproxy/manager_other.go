//go:build !linux && !darwin

package sysproxy

func NewSystemProxyManager() SystemProxyManager {
    return &NoOpProxyManager{}
}
