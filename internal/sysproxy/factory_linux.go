//go:build linux

package sysproxy

// NewSystemProxyManager returns the GNOME implementation on Linux.
func NewSystemProxyManager() SystemProxyManager {
	return newGnomeProxyManager()
}
