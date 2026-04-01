//go:build !linux
// +build !linux

package gui

const (
	ProxyModeManual = "manual"
	ProxyModeNone   = "none"
)

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

func getSystemProxyMode() (string, error) {
	return "", nil
}
