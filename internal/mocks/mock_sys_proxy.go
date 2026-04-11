package mocks

// MockSystemProxyManager is a mock implementation of sysproxy.SystemProxyManager for testing.
type MockSystemProxyManager struct {
	MockIsSupported    func() bool
	MockGetMode        func() (string, error)
	MockSetMode        func(mode string) error
	MockSetProxy       func(host, port, proxyType string) error
	MockGetIgnoreHosts func() (string, error)
	MockSetIgnoreHosts func(ignoreHostsText string) error
	MockGetActiveProxy func() (host string, port string, err error)
}

func (m *MockSystemProxyManager) IsSupported() bool {
	if m.MockIsSupported != nil {
		return m.MockIsSupported()
	}
	return false
}

func (m *MockSystemProxyManager) GetMode() (string, error) {
	if m.MockGetMode != nil {
		return m.MockGetMode()
	}
	return "", nil
}

func (m *MockSystemProxyManager) SetMode(mode string) error {
	if m.MockSetMode != nil {
		return m.MockSetMode(mode)
	}
	return nil
}

func (m *MockSystemProxyManager) SetProxy(host, port, proxyType string) error {
	if m.MockSetProxy != nil {
		return m.MockSetProxy(host, port, proxyType)
	}
	return nil
}

func (m *MockSystemProxyManager) GetIgnoreHosts() (string, error) {
	if m.MockGetIgnoreHosts != nil {
		return m.MockGetIgnoreHosts()
	}
	return "", nil
}

func (m *MockSystemProxyManager) SetIgnoreHosts(ignoreHostsText string) error {
	if m.MockSetIgnoreHosts != nil {
		return m.MockSetIgnoreHosts(ignoreHostsText)
	}
	return nil
}

func (m *MockSystemProxyManager) GetActiveProxy() (string, string, error) {
	if m.MockGetActiveProxy != nil {
		return m.MockGetActiveProxy()
	}
	return "", "", nil
}
