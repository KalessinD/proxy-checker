//go:build linux

package sysproxy

func NewSystemProxyManager() SystemProxyManager {
    return newGnomeProxyManager()
}
