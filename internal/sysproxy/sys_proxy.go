package sysproxy

const (
	ProxyModeManual = "manual"
	ProxyModeNone   = "none"
)

//go:generate mockgen -destination=../../internal/mocks/mock_sys_proxy.go -package=mocks proxy-checker/internal/sysproxy SystemProxyManager

// SystemProxyManager encapsulates platform-specific system proxy management.
type SystemProxyManager interface {
	IsSupported() bool
	GetMode() (string, error)
	SetMode(mode string) error
	SetProxy(host, port, proxyType string) error
	GetIgnoreHosts() (string, error)
	SetIgnoreHosts(ignoreHostsText string) error
	GetActiveProxy() (host string, port string, err error)
}

type NoOpProxyManager struct{}

func (m *NoOpProxyManager) IsSupported() bool               { return false }
func (m *NoOpProxyManager) GetMode() (string, error)        { return "", nil }
func (m *NoOpProxyManager) SetMode(_ string) error          { return nil }
func (m *NoOpProxyManager) SetProxy(_, _, _ string) error   { return nil }
func (m *NoOpProxyManager) GetIgnoreHosts() (string, error) { return "", nil }
func (m *NoOpProxyManager) SetIgnoreHosts(_ string) error   { return nil }
func (m *NoOpProxyManager) GetActiveProxy() (string, string, error) {
	return "", "", nil
}
