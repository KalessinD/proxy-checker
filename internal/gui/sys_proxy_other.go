//go:build !linux
// +build !linux

package gui

func isSystemProxySupported() bool {
	return false
}

func setSystemProxy(host, port, proxyType string) error {
	return nil
}
