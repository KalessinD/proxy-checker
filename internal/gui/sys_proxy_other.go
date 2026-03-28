//go:build !linux
// +build !linux

package gui

func isSystemProxySupported() bool {
	return false
}

func setSystemProxy(host, port, proxyType string) error {
	return nil
}

// Добавлен заглушка для новой функции
func setSystemProxyMode(mode string) error {
	return nil
}
