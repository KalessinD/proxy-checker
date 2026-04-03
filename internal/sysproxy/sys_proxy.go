package gui

import (
	"runtime"
	"sort"
	"strings"
)

const (
	ProxyModeManual = "manual"
	ProxyModeNone   = "none"
)

type SystemProxyManager interface {
	IsSupported() bool
	GetMode() (string, error)
	SetMode(mode string) error
	SetProxy(host, port, proxyType string) error
	GetIgnoreHosts() (string, error)
	SetIgnoreHosts(ignoreHostsText string) error
}

type NoOpProxyManager struct{}

func (m *NoOpProxyManager) IsSupported() bool               { return false }
func (m *NoOpProxyManager) GetMode() (string, error)        { return "", nil }
func (m *NoOpProxyManager) SetMode(_ string) error          { return nil }
func (m *NoOpProxyManager) SetProxy(_, _, _ string) error   { return nil }
func (m *NoOpProxyManager) GetIgnoreHosts() (string, error) { return "", nil }
func (m *NoOpProxyManager) SetIgnoreHosts(_ string) error   { return nil }

func NewSystemProxyManager() SystemProxyManager {
	if runtime.GOOS == "linux" {
		return newLinuxProxyManager()
	}
	return &NoOpProxyManager{}
}

func ParseGVariantStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "[]" {
		return nil
	}

	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")

	elements := strings.Split(raw, ",")

	var result []string
	for _, elem := range elements {
		cleaned := strings.TrimSpace(elem)
		cleaned = strings.Trim(cleaned, "'")
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}

	sort.Strings(result)
	return result
}

func FormatGVariantStringArray(hosts []string) string {
	if len(hosts) == 0 {
		return "[]"
	}

	sort.Strings(hosts)

	quoted := make([]string, len(hosts))
	for i, host := range hosts {
		quoted[i] = "'" + host + "'"
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}
