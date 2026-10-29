package sysproxy

import (
	"errors"
	"fmt"
	"proxy-checker/internal/common/i18n"
	"runtime"
	"strings"
)

// networkSetupProxyManager manages system proxies via the macOS
// networksetup utility. All orchestration is platform-independent:
// the only macOS-specific part is that the darwin factory wires this
// manager to the real CommandRunner.
type networkSetupProxyManager struct {
	runner CommandRunner
}

func newNetworkSetupProxyManager(runner CommandRunner) SystemProxyManager {
	return &networkSetupProxyManager{runner: runner}
}

func (m *networkSetupProxyManager) IsSupported() bool {
	return runtime.GOOS == "darwin"
}

func (m *networkSetupProxyManager) GetMode() (string, error) {
	service, err := m.getActiveNetworkService()
	if err != nil {
		return ProxyModeNone, err
	}

	webEnabled, _, _, err := m.getWebProxyState(service)
	if err != nil {
		return ProxyModeNone, err
	}

	if webEnabled {
		return ProxyModeManual, nil
	}

	socksEnabled, _, _, err := m.getSOCKSProxyState(service)
	if err != nil {
		return ProxyModeNone, err
	}

	if socksEnabled {
		return ProxyModeManual, nil
	}

	return ProxyModeNone, nil
}

func (m *networkSetupProxyManager) SetMode(mode string) error {
	service, err := m.getActiveNetworkService()
	if err != nil {
		return err
	}
	return m.setModeForService(service, mode)
}

// setModeForService disables web and SOCKS proxies for the given network service.
func (m *networkSetupProxyManager) setModeForService(service, mode string) error {
	if mode != ProxyModeNone {
		return nil
	}
	if err := m.setNetworkSetup(service, "-setwebproxystate", "off"); err != nil {
		return err
	}
	return m.setNetworkSetup(service, "-setsocksfirewallproxystate", "off")
}

func (m *networkSetupProxyManager) SetProxy(host, port, proxyType string) error {
	service, err := m.getActiveNetworkService()
	if err != nil {
		return err
	}

	pType := strings.ToLower(proxyType)

	if err := m.setModeForService(service, ProxyModeNone); err != nil {
		return err
	}

	switch pType {
	case "http", "https":
		if err := m.setNetworkSetup(service, "-setwebproxy", host, port); err != nil {
			return err
		}
		return m.setNetworkSetup(service, "-setwebproxystate", "on")
	case "socks4", "socks5":
		if err := m.setNetworkSetup(service, "-setsocksfirewallproxy", host, port); err != nil {
			return err
		}
		return m.setNetworkSetup(service, "-setsocksfirewallproxystate", "on")
	default:
		if err := m.setNetworkSetup(service, "-setwebproxy", host, port); err != nil {
			return err
		}
		return m.setNetworkSetup(service, "-setwebproxystate", "on")
	}
}

func (m *networkSetupProxyManager) GetIgnoreHosts() (string, error) {
	service, err := m.getActiveNetworkService()
	if err != nil {
		return "", err
	}

	out, err := m.networkSetup("-getproxybypassdomains", service)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(out) == "There aren't any bypass domains set" {
		return "", nil
	}

	return strings.TrimSpace(out), nil
}

func (m *networkSetupProxyManager) SetIgnoreHosts(ignoreHostsText string) error {
	service, err := m.getActiveNetworkService()
	if err != nil {
		return err
	}

	hosts := strings.Split(ignoreHostsText, "\n")
	var args []string
	args = append(args, "-setproxybypassdomains")

	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h != "" {
			args = append(args, h)
		}
	}

	if len(args) == 1 {
		args = append(args, "Empty")
	}

	return m.setNetworkSetup(service, args...)
}

func (m *networkSetupProxyManager) GetActiveProxy() (string, string, error) {
	service, err := m.getActiveNetworkService()
	if err != nil {
		return "", "", err
	}

	webEnabled, webHost, webPort, err := m.getWebProxyState(service)
	if err != nil {
		return "", "", err
	}

	if webEnabled && webHost != "" && webPort != "" {
		return webHost, webPort, nil
	}

	socksEnabled, socksHost, socksPort, err := m.getSOCKSProxyState(service)
	if err != nil {
		return "", "", err
	}

	if socksEnabled && socksHost != "" && socksPort != "" {
		return socksHost, socksPort, nil
	}

	return "", "", nil
}

func (m *networkSetupProxyManager) getWebProxyState(service string) (bool, string, string, error) {
	out, err := m.networkSetup("-getwebproxy", service)
	if err != nil {
		return false, "", "", err
	}
	return ParseNetworkSetupProxyOutput(out)
}

func (m *networkSetupProxyManager) getSOCKSProxyState(service string) (bool, string, string, error) {
	out, err := m.networkSetup("-getsocksfirewallproxy", service)
	if err != nil {
		return false, "", "", err
	}
	return ParseNetworkSetupProxyOutput(out)
}

func (m *networkSetupProxyManager) networkSetup(args ...string) (string, error) {
	stdout, stderr, err := m.runner.Run("networksetup", args...)
	if err != nil {
		// networksetup prints its errors to stdout, not stderr.
		return "", fmt.Errorf("%s: %w (stderr: %s, stdout: %s)", i18n.T("sysproxy.err_networksetup"), err, stderr, stdout)
	}
	return stdout, nil
}

// setNetworkSetup runs networksetup with the correct argument order:
// <command flag> <service> [values...]. The service must follow the flag.
func (m *networkSetupProxyManager) setNetworkSetup(service string, args ...string) error {
	if len(args) == 0 {
		return errors.New(i18n.T("sysproxy.err_networksetup_no_command"))
	}
	fullArgs := append([]string{args[0], service}, args[1:]...)
	_, err := m.networkSetup(fullArgs...)
	return err
}

func (m *networkSetupProxyManager) getActiveNetworkService() (string, error) {
	routeOut, _, err := m.runner.Run("route", "-n", "get", "default")
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("sysproxy.err_default_route"), err)
	}

	iface := ParseDefaultRouteInterface(routeOut)
	if iface == "" {
		return "", errors.New(i18n.T("sysproxy.err_default_route_parse"))
	}

	hwOut, err := m.networkSetup("-listallhardwareports")
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("sysproxy.err_list_hw_ports"), err)
	}

	mapping := ParseHardwarePortMapping(hwOut)
	serviceName, exists := mapping[iface]
	if !exists {
		return "", fmt.Errorf("%s: %s", i18n.T("sysproxy.err_service_not_found"), iface)
	}

	return serviceName, nil
}

// ParseDefaultRouteInterface extracts the default interface name from
// the output of `route -n get default`.
func ParseDefaultRouteInterface(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// ParseHardwarePortMapping builds a device -> service name map from
// the output of `networksetup -listallhardwareports`.
func ParseHardwarePortMapping(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")

	var currentDevice string
	var currentPort string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Device:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentDevice = parts[1]
			}
		} else if strings.HasPrefix(line, "Hardware Port:") {
			currentPort = strings.TrimPrefix(line, "Hardware Port:")
			currentPort = strings.TrimSpace(currentPort)
		}

		if currentDevice != "" && currentPort != "" {
			result[currentDevice] = currentPort
			currentDevice = ""
			currentPort = ""
		}
	}

	return result
}

// ParseNetworkSetupProxyOutput extracts Enabled, Server, and Port from networksetup output.
func ParseNetworkSetupProxyOutput(output string) (bool, string, string, error) {
	var enabled bool
	var server string
	var port string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Enabled:") {
			enabled = strings.Contains(line, "Yes")
		} else if strings.HasPrefix(line, "Server:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				server = parts[1]
			}
		} else if strings.HasPrefix(line, "Port:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				port = parts[1]
			}
		}
	}

	return enabled, server, port, nil
}
