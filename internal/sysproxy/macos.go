//go:build darwin

package sysproxy

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type macOSProxyManager struct{}

func newMacOSProxyManager() SystemProxyManager {
    return &macOSProxyManager{}
}

func (m *macOSProxyManager) IsSupported() bool {
    return runtime.GOOS == "darwin"
}

func (m *macOSProxyManager) GetMode() (string, error) {
    service, err := getActiveNetworkService()
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

func (m *macOSProxyManager) SetMode(mode string) error {
    service, err := getActiveNetworkService()
    if err != nil {
        return err
    }

    if mode == ProxyModeNone {
        if err := setNetworkSetup(service, "-setwebproxystate", "off"); err != nil {
            return err
        }
        return setNetworkSetup(service, "-setsocksfirewallproxystate", "off")
    }
    return nil
}

func (m *macOSProxyManager) SetProxy(host, port, proxyType string) error {
    service, err := getActiveNetworkService()
    if err != nil {
        return err
    }

    pType := strings.ToLower(proxyType)

    if err := m.SetMode(ProxyModeNone); err != nil {
        return err
    }

    switch pType {
    case "http", "https":
        if err := setNetworkSetup(service, "-setwebproxy", host, port); err != nil {
            return err
        }
        return setNetworkSetup(service, "-setwebproxystate", "on")
    case "socks4", "socks5":
        if err := setNetworkSetup(service, "-setsocksfirewallproxy", host, port); err != nil {
            return err
        }
        return setNetworkSetup(service, "-setsocksfirewallproxystate", "on")
    default:
        if err := setNetworkSetup(service, "-setwebproxy", host, port); err != nil {
            return err
        }
        return setNetworkSetup(service, "-setwebproxystate", "on")
    }
}

func (m *macOSProxyManager) GetIgnoreHosts() (string, error) {
    service, err := getActiveNetworkService()
    if err != nil {
        return "", err
    }

    out, err := execNetworkSetup("-getproxybypassdomains", service)
    if err != nil {
        return "", err
    }

    if strings.TrimSpace(out) == "There aren't any bypass domains set" {
        return "", nil
    }

    return strings.TrimSpace(out), nil
}

func (m *macOSProxyManager) SetIgnoreHosts(ignoreHostsText string) error {
    service, err := getActiveNetworkService()
    if err != nil {
        return err
    }

    hosts := strings.Split(ignoreHostsText, "\n")
    var args []string
    args = append(args, "-setproxybypassdomains", service)

    for _, h := range hosts {
        h = strings.TrimSpace(h)
        if h != "" {
            args = append(args, h)
        }
    }

    if len(args) == 2 {
        args = append(args, "Empty")
    }

    return setNetworkSetup(service, args...)
}

func (m *macOSProxyManager) GetActiveProxy() (string, string, error) {
    service, err := getActiveNetworkService()
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

func (m *macOSProxyManager) getWebProxyState(service string) (bool, string, string, error) {
    out, err := execNetworkSetup("-getwebproxy", service)
    if err != nil {
        return false, "", "", err
    }
    return parseNetworkSetupProxyOutput(out)
}

func (m *macOSProxyManager) getSOCKSProxyState(service string) (bool, string, string, error) {
    out, err := execNetworkSetup("-getsocksfirewallproxy", service)
    if err != nil {
        return false, "", "", err
    }
    return parseNetworkSetupProxyOutput(out)
}

func execNetworkSetup(args ...string) (string, error) {
    cmd := exec.Command("networksetup", args...)
    var out bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("networksetup failed: %v, %s", err, stderr.String())
    }
    return out.String(), nil
}

func setNetworkSetup(service string, args ...string) error {
    fullArgs := append([]string{service}, args...)
    _, err := execNetworkSetup(fullArgs...)
    return err
}

func getActiveNetworkService() (string, error) {
    out, err := exec.Command("route", "-n", "get", "default").Output()
    if err != nil {
        return "", fmt.Errorf("failed to get default route: %w", err)
    }

    iface := ParseDefaultRouteInterface(string(out))
    if iface == "" {
        return "", fmt.Errorf("failed to parse default interface from route output")
    }

    hwOut, err := exec.Command("networksetup", "-listallhardwareports").Output()
    if err != nil {
        return "", fmt.Errorf("failed to list hardware ports: %w", err)
    }

    mapping := parseHardwarePortMapping(string(hwOut))
    serviceName, exists := mapping[iface]
    if !exists {
        return "", fmt.Errorf("network service not found for interface: %s", iface)
    }

    return serviceName, nil
}

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

func parseHardwarePortMapping(output string) map[string]string {
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

// parseNetworkSetupProxyOutput extracts Enabled, Server, and Port from networksetup output.
func parseNetworkSetupProxyOutput(output string) (bool, string, string, error) {
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
